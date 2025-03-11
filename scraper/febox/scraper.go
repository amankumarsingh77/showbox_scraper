package febox

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
)

type Scraper struct {
	client      *http.Client
	dbRepo      *repository.MongoRepo
	visitedURLs map[string]bool
	config      *Config
	mu          sync.Mutex
}

func NewScraper(dbRepo *repository.MongoRepo, cfg *Config) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: time.Duration(cfg.HTTPTimeout) * time.Second,
		},
		dbRepo:      dbRepo,
		visitedURLs: make(map[string]bool),
		config:      cfg,
	}
}

// ScrapeMoviesConcurrently scrapes multiple movies concurrently
// Maintained for backward compatibility - consider using ScrapeContentConcurrently for new code
func (s *Scraper) ScrapeMoviesConcurrently(movies []models.Movie) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.config.MaxConcurrency)
	ticker := time.NewTicker(time.Duration(s.config.RequestInterval) * time.Second)
	defer ticker.Stop()

	for idx := range movies {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-ticker.C
			sem <- struct{}{}
			defer func() { <-sem }()

			// Use a pointer to the movie in the original slice
			movie := &movies[idx]

			for retries := 0; retries < s.config.MaxRetries; retries++ {
				if err := s.ScrapeContent(movie, idx); err != nil {
					log.Printf("Error scraping movie %s: %v", movie.Title, err)
					if isRateLimitError(err) {
						time.Sleep(time.Duration(s.config.RetryDelay<<retries) * time.Second)
						continue
					}
				}
				break
			}
		}(idx)
	}
	wg.Wait()
}

// ScrapeSeriesConcurrently scrapes multiple TV series concurrently
// Maintained for backward compatibility - consider using ScrapeContentConcurrently for new code
func (s *Scraper) ScrapeSeriesConcurrently(series []models.TV) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.config.MaxConcurrency)
	ticker := time.NewTicker(time.Duration(s.config.RequestInterval) * time.Second)
	defer ticker.Stop()

	for idx := range series {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-ticker.C
			sem <- struct{}{}
			defer func() { <-sem }()

			// Use a pointer to the TV series in the original slice
			tv := &series[idx]

			for retries := 0; retries < s.config.MaxRetries; retries++ {
				if err := s.ScrapeContent(tv, idx); err != nil {
					log.Printf("Error scraping TV series %s: %v", tv.Title, err)
					if isRateLimitError(err) {
						time.Sleep(time.Duration(s.config.RetryDelay<<retries) * time.Second)
						continue
					}
				}
				break
			}
		}(idx)
	}
	wg.Wait()
}

func (s *Scraper) isVisited(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visitedURLs[url]
}

func (s *Scraper) markVisited(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visitedURLs[url] = true
}

// ScrapeContentConcurrently is a general function that can scrape both movies and TV series concurrently
func (s *Scraper) ScrapeContentConcurrently(contents []interface{}) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.config.MaxConcurrency)
	ticker := time.NewTicker(time.Duration(s.config.RequestInterval) * time.Second)
	defer ticker.Stop()

	// For interface{} slice, we need to be careful about pointers
	// Each element in contents should already be a pointer to the appropriate type
	for idx, content := range contents {
		wg.Add(1)
		go func(c interface{}, idx int) {
			defer wg.Done()
			<-ticker.C
			sem <- struct{}{}
			defer func() { <-sem }()

			var title string
			switch v := c.(type) {
			case *models.Movie:
				title = v.Title
			case *models.TV:
				title = v.Title
			default:
				log.Printf("Unsupported content type: %T", c)
				return
			}

			for retries := 0; retries < s.config.MaxRetries; retries++ {
				if err := s.ScrapeContent(c, idx); err != nil {
					log.Printf("Error scraping content %s: %v", title, err)
					if isRateLimitError(err) {
						time.Sleep(time.Duration(s.config.RetryDelay<<retries) * time.Second)
						continue
					}
				}
				break
			}
		}(content, idx)
	}
	wg.Wait()
}
