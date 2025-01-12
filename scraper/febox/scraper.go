package febox

import (
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"log"
	"net/http"
	"sync"
	"time"
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

func (s *Scraper) ScrapeMoviesConcurrently(movies []models.Movie) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.config.MaxConcurrency)
	ticker := time.NewTicker(time.Duration(s.config.RequestInterval) * time.Second)
	defer ticker.Stop()

	for idx, movie := range movies {
		wg.Add(1)
		go func(m models.Movie, idx int) {
			defer wg.Done()
			<-ticker.C
			sem <- struct{}{}
			defer func() { <-sem }()

			for retries := 0; retries < s.config.MaxRetries; retries++ {
				if err := s.scrapeMovie(&m, idx); err != nil {
					log.Printf("Error scraping movie %s: %v", m.Title, err)
					if isRateLimitError(err) {
						time.Sleep(time.Duration(s.config.RetryDelay<<retries) * time.Second)
						continue
					}
				}
				break
			}
		}(movie, idx)
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
