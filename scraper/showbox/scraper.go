package showbox

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gocolly/colly"
)

type Scraper struct {
	config     *Config
	collector  *colly.Collector
	storage    *Storage
	movies     []Movie
	visited    sync.Map
	mu         sync.RWMutex
	shutdown   chan struct{}
	done       chan struct{}
	activeJobs sync.WaitGroup
}

func NewScraper(config *Config, storage *Storage) (*Scraper, error) {
	c := colly.NewCollector(
		colly.AllowedDomains("www.showbox.media", "simple-proxy.xartpvt.workers.dev"),
		colly.UserAgent(config.UserAgent),
		colly.Async(true),
	)

	c.SetRequestTimeout(time.Duration(config.Timeout) * time.Second)

	err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: config.Parallelism,
		RandomDelay: time.Duration(config.RandomDelay) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set limit rule: %v", err)
	}

	return &Scraper{
		config:    config,
		collector: c,
		storage:   storage,
		shutdown:  make(chan struct{}),
		done:      make(chan struct{}),
	}, nil
}

func (s *Scraper) setupCallbacks() {
	s.collector.OnRequest(func(r *colly.Request) {
		s.activeJobs.Add(1)
		log.Printf("Visiting: %s\n", r.URL)
	})

	s.collector.OnScraped(func(r *colly.Response) {
		defer s.activeJobs.Done()
		log.Printf("Visited: %s (Status: %d)\n", r.Request.URL, r.StatusCode)
	})

	s.collector.OnError(func(r *colly.Response, err error) {
		defer s.activeJobs.Done()
		if r.StatusCode == 429 {
			time.Sleep(4 * time.Second)
			log.Printf("Rate limit exceeded: %d (Status: %s)\n", r.StatusCode, r.Request.URL)
			r.Request.Retry()
		}
		log.Printf("Error on %s: %v (Status: %d)\n", r.Request.URL, err, r.StatusCode)
	})

	s.collector.OnHTML(".film_list-wrap .flw-item", func(e *colly.HTMLElement) {
		if e.DOM.Parents().HasClass("film_related") {
			return
		}

		select {
		case <-s.shutdown:
			return
		default:
			link := e.ChildAttr("div:nth-child(1) > a:nth-child(3)", "href")
			if link == "" {
				return
			}

			fullLink := s.config.ProxyURL + s.config.BaseURL + link

			if _, exists := s.visited.LoadOrStore(fullLink, true); !exists {
				s.activeJobs.Add(1)
				go func(url string) {
					defer s.activeJobs.Done()
					if err := s.collector.Visit(url); err != nil {
						log.Printf("Error visiting URL %s: %v", url, err)
					}
				}(fullLink)
			}
		}
	})

	s.collector.OnHTML(".dp-i-content", func(e *colly.HTMLElement) {
		select {
		case <-s.shutdown:
			return
		default:
			link := e.ChildAttr(".heading-name a", "href")
			if link == "" {
				return
			}

			id := strings.Split(link, "/")[3]

			cleanText := func(s string) string {
				return strings.TrimSpace(strings.ReplaceAll(s, "\n", ""))
			}

			imdbRating := ""
			imdbText := e.ChildText(".btn-imdb")
			if imdbText != "" {
				if parts := strings.Split(imdbText, ":"); len(parts) > 1 {
					imdbRating = strings.TrimSpace(parts[1])
				}
			}

			var releaseDate, genre, casts, duration, country, production string
			e.ForEach(".row-line", func(_ int, el *colly.HTMLElement) {
				label := strings.ToLower(strings.TrimSpace(el.ChildText(".type")))
				value := cleanText(el.Text[len(el.ChildText(".type")):])

				switch {
				case strings.Contains(label, "released"):
					releaseDate = value
				case strings.Contains(label, "genre"):
					genre = value
				case strings.Contains(label, "casts"):
					casts = value
				case strings.Contains(label, "duration"):
					duration = value
				case strings.Contains(label, "country"):
					country = value
				case strings.Contains(label, "production"):
					production = value
				}
			})

			movie := Movie{
				ID:          id,
				Title:       cleanText(e.ChildText(".heading-name")),
				Description: cleanText(e.ChildText(".description")),
				ReleaseDate: releaseDate,
				Genre:       genre,
				Casts:       casts,
				Duration:    duration,
				Country:     country,
				Production:  production,
				IMDBRating:  imdbRating,
				ScrapedAt:   time.Now(),
			}

			log.Println(movie)

			s.mu.Lock()
			s.movies = append(s.movies, movie)
			s.mu.Unlock()
		}
	})

}

func (s *Scraper) Run() error {
	s.setupCallbacks()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for page := s.config.StartPage; page <= s.config.EndPage; page++ {
			select {
			case <-s.shutdown:
				return
			default:
				url := fmt.Sprintf("%s/movie?page=%d", s.config.BaseURL, page)
				if _, exists := s.visited.LoadOrStore(url, true); !exists {
					s.activeJobs.Add(1)
					go func(url string) {
						defer s.activeJobs.Done()
						if err := s.collector.Visit(url); err != nil {
							log.Printf("Failed to visit %s: %v\n", url, err)
						}
					}(url)
				}
			}
		}
	}()

	select {
	case <-sigChan:
		log.Println("Signal received. Shutting down...")
		close(s.shutdown)
	case <-s.done:
		log.Println("All jobs completed. Shutting down...")
		close(s.shutdown)
	}

	s.collector.Wait()
	s.activeJobs.Wait()

	if err := s.storage.SaveProgress(s.movies); err != nil {
		log.Printf("Error saving progress: %v\n", err)
	}

	if err := s.storage.MergeFiles(); err != nil {
		log.Printf("Error during final merge: %v\n", err)
	}

	log.Println("Scraper shutdown gracefully.")
	return nil
}
