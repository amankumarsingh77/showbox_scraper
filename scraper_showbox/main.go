package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gocolly/colly"
)

type Movie struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ScrapedAt   time.Time `json:"scraped_at"`
}

type Scraper struct {
	collector   *colly.Collector
	proxyurl    string
	movies      []Movie
	visited     sync.Map
	mu          sync.RWMutex
	shutdown    chan struct{}
	done        chan struct{}
	activeCount int
	activeJobs  sync.WaitGroup
	baseURL     string
	startPage   int
	endPage     int
}

func NewScraper(baseURL, proxyurl string, startPage, endPage int) (*Scraper, error) {
	c := colly.NewCollector(
		colly.AllowedDomains("www.showbox.media", "simple-proxy.xartpvt.workers.dev"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.Async(true),
	)
	c.SetRequestTimeout(120 * time.Second)

	err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		RandomDelay: 3 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set limit rule: %v", err)
	}

	return &Scraper{
		collector: c,
		shutdown:  make(chan struct{}),
		done:      make(chan struct{}),
		proxyurl:  proxyurl,
		baseURL:   baseURL,
		startPage: startPage,
		endPage:   endPage,
	}, nil
}

func (s *Scraper) setupCallbacks() {
	s.collector.OnRequest(func(r *colly.Request) {
		s.activeJobs.Add(1)
		s.mu.Lock()
		s.activeCount++
		log.Printf("Visiting: %s\n", r.URL)
		s.mu.Unlock()
	})

	s.collector.OnScraped(func(r *colly.Response) {
		defer s.activeJobs.Done()
		s.mu.Lock()
		s.activeCount--
		if s.activeCount == 0 {
			select {
			case <-s.shutdown:
			default:
				close(s.done)
			}
		}
		s.mu.Unlock()
		log.Printf("Visited: %s (Status: %d) (Active: %d)\n", r.Request.URL, r.StatusCode, s.activeCount)
	})

	s.collector.OnError(func(r *colly.Response, err error) {
		defer s.activeJobs.Done()
		s.mu.Lock()
		s.activeCount--
		s.mu.Unlock()
		if r.StatusCode == 429 {
			time.Sleep(4 * time.Second)
			log.Printf("Rate limit exceeded: %d (Status: %s)\n", r.StatusCode, r.Request.URL)
			r.Request.Retry()
		}
		log.Printf("Error on %s: %v (Status: %d)\n", r.Request.URL, err, r.StatusCode)
	})

	s.collector.OnHTML(".film_list-wrap .flw-item", func(e *colly.HTMLElement) {
		// Check if the item is inside the "film_related" section, indicating a "You may also like" section
		parentClass := e.DOM.Parents().HasClass("film_related")
		if parentClass {
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

			fullLink := s.proxyurl + s.baseURL + link

			if _, exists := s.visited.LoadOrStore(fullLink, true); !exists {
				s.activeJobs.Add(1)

				go func(url string) {
					defer s.activeJobs.Done()

					err := s.collector.Visit(url)
					if err != nil {
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
			movie := Movie{
				ID:          id,
				Title:       e.ChildText(".heading-name"),
				Description: e.ChildText(".description"),
				ScrapedAt:   time.Now(),
			}

			s.mu.Lock()

			s.movies = append(s.movies, movie)
			s.mu.Unlock()
		}
	})
}

func (s *Scraper) saveProgress() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.movies) == 0 {
		return
	}

	tempDir := "temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("Failed to create temp directory: %v\n", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	tempFile := filepath.Join(tempDir, fmt.Sprintf("movies_%s.json", timestamp))

	file, err := os.Create(tempFile)
	if err != nil {
		log.Printf("Failed to create temp file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.movies); err != nil {
		log.Printf("Failed to save progress: %v\n", err)
		return
	}

	log.Printf("Progress saved: %d movies\n", len(s.movies))
}

func (s *Scraper) mergeFiles() error {
	// Read the existing movies_final.json file
	allMovies := make(map[string]Movie)
	finalFilePath := "movies_final.json"

	if _, err := os.Stat(finalFilePath); err == nil {
		data, err := os.ReadFile(finalFilePath)
		if err != nil {
			return fmt.Errorf("failed to read existing final file: %v", err)
		}

		var existingMovies []Movie
		if err := json.Unmarshal(data, &existingMovies); err != nil {
			return fmt.Errorf("failed to parse existing final file: %v", err)
		}

		for _, movie := range existingMovies {
			allMovies[movie.ID] = movie
		}
	}

	// Process temp files
	files, err := filepath.Glob("temp/*.json")
	if err != nil {
		return fmt.Errorf("failed to list temp files: %v", err)
	}

	for _, file := range files {
		var movies []Movie
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", file, err)
			continue
		}

		if err := json.Unmarshal(data, &movies); err != nil {
			log.Printf("Error parsing file %s: %v\n", file, err)
			continue
		}

		for _, movie := range movies {
			allMovies[movie.ID] = movie
		}
	}

	// Write merged data to movies_final.json
	final := make([]Movie, 0, len(allMovies))
	for _, movie := range allMovies {
		final = append(final, movie)
	}

	finalFile, err := os.Create(finalFilePath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %v", err)
	}
	defer finalFile.Close()

	encoder := json.NewEncoder(finalFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(final); err != nil {
		return fmt.Errorf("failed to save final file: %v", err)
	}

	// Clean up temp files
	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			log.Printf("Failed to remove file %s: %v\n", file, err)
		}
	}

	log.Printf("Final merge complete: %d unique movies saved\n", len(final))
	return nil
}

func (s *Scraper) Run() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for page := s.startPage; page <= s.endPage; page++ {
			select {
			case <-s.shutdown:
				return
			default:
				url := fmt.Sprintf("%s%s/movie?page=%d", s.proxyurl, s.baseURL, page)

				if _, exists := s.visited.LoadOrStore(url, true); !exists {
					s.activeJobs.Add(1)
					go func(url string) {
						defer s.activeJobs.Done()
						err := s.collector.Visit(url)
						if err != nil {
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
	s.saveProgress()
	if err := s.mergeFiles(); err != nil {
		log.Printf("Error during final merge: %v\n", err)
	}
	log.Println("Scraper shutdown gracefully.")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	scraper, err := NewScraper("https://www.showbox.media", "https://simple-proxy.xartpvt.workers.dev?destination=", 1, 10)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}

	scraper.setupCallbacks()
	scraper.Run()
}
