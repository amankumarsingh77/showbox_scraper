package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gocolly/colly"
)

const (
	proxyURL    = "https://simple-proxy-2.xartpvt.workers.dev?destination="
	showboxBase = "http://156.242.65.27/"
	feboxBase   = "https://www.febbox.com"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
)

type Movie struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ScrapedAt   time.Time `json:"scraped_at"`
}

type FileInfo struct {
	Fid       int64  `json:"fid"`
	Size      string `json:"size"`
	Filename  string `json:"file_name"`
	Thumbnail string `json:"thumb_big"`
}

type fileResponse struct {
	Data struct {
		File FileInfo `json:"file"`
	} `json:"data"`
}

type VideoQuality struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Size    string `json:"size"`
}

type Scraper struct {
	collector   *colly.Collector
	dbRepo      *repository.MongoRepo
	client      *http.Client
	visitedURLs map[string]bool
	mu          sync.Mutex
}

func NewScraper(dbRepo *repository.MongoRepo) *Scraper {
	return &Scraper{
		client:      &http.Client{Timeout: 120 * time.Second},
		dbRepo:      dbRepo,
		visitedURLs: make(map[string]bool),
	}
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

func (s *Scraper) setupCollector(currentMovie *Movie) {

	s.collector.OnRequest(func(r *colly.Request) {
		if s.isVisited(r.URL.String()) {
			//log.Printf("Skipping duplicate visit: %s", r.URL.String())
			r.Abort()
			return
		}
		log.Printf("Visiting: %s\n", r.URL.String())
		s.markVisited(r.URL.String())
	})

	s.collector.OnResponse(func(r *colly.Response) {
		log.Printf("Finished scraping: %s", r.Request.URL.String())
	})

	s.collector.OnScraped(func(r *colly.Response) {
		log.Printf("Finished scraping: %s", r.Request.URL.String())
	})

	s.collector.OnHTML(".f_list_scroll", func(e *colly.HTMLElement) {
		log.Println("reached")
		var files []models.File
		e.ForEach("div[data-id]", func(_ int, el *colly.HTMLElement) {
			fileId := el.Attr("data-id")
			if file, _ := getFileDetails(fileId); file.FID != 0 {
				files = append(files, file)
			}
		})

		movie := &models.Movie{
			Title:       currentMovie.Title,
			Description: currentMovie.Description,
			MovieID:     currentMovie.ID,
			Files:       files,
		}

		if err := s.dbRepo.CreateMovie(context.Background(), movie); err != nil && len(files) > 0 {
			log.Printf("Error saving movie to database: %v", err)
			return
		}

		log.Printf("Successfully saved movie: %s", movie.Title)
	})
}

func getFileDetails(fileid string) (models.File, error) {
	url := fmt.Sprintf("%s/file/file_info?fid=%s", feboxBase, fileid)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return models.File{}, nil
	}
	defer resp.Body.Close()

	var data fileResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding file info: %v", err)
		return models.File{}, nil
	}

	links := getQualities(fileid)
	//log.Println(links)

	return models.File{
		FID:      data.Data.File.Fid,
		FileName: data.Data.File.Filename,
		Size:     data.Data.File.Size,
		ThumbURL: data.Data.File.Thumbnail,
		Links:    links,
	}, nil
}

func getQualities(fileId string) []models.Link {
	url := fmt.Sprintf("%s/console/video_quality_list?fid=%s?type=1", feboxBase, fileId)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil
	}

	req.Header.Add("Cookie", os.Getenv("FEBBOX_COOKIE"))
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting qualities: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil
	}

	var input map[string]interface{}
	if err = json.Unmarshal(body, &input); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		return nil
	}

	html, ok := input["html"].(string)
	if !ok {
		log.Println("HTML field not found in response")
		return nil
	}

	data := parseHtmlToJson(html)
	var links []models.Link
	for _, movie := range data {
		link := models.Link{
			Quality: movie.Quality,
			URL:     movie.URL,
			Size:    movie.Size,
		}
		links = append(links, link)
	}
	return links
}

func parseHtmlToJson(html string) []VideoQuality {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return nil
	}

	var videos []VideoQuality
	doc.Find(".file_quality").Each(func(i int, s *goquery.Selection) {
		video := VideoQuality{
			Quality: s.AttrOr("data-quality", ""),
			URL:     s.AttrOr("data-url", ""),
			Size:    s.Find(".desc .size").Text(),
		}
		videos = append(videos, video)
	})
	return videos
}

func getMoviesList(start, end int) []Movie {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory:", err)
	}

	filePath := filepath.Join(dir, "movies_final.json")
	//var finalMovies []Movie

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	var movies []Movie
	if err := json.Unmarshal(data, &movies); err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	if start < 0 || end >= len(movies) || start > end {
		log.Fatal("Invalid range for start or end index")
	}

	return movies[start : end+1]
}

func (s *Scraper) scrapeMovie(movie *Movie, idx int) error {
	shoemediaUrl := fmt.Sprintf("%s/index/share_link?id=%s&type=1", showboxBase, movie.ID)
	req, err := http.NewRequest("GET", shoemediaUrl, nil)
	if err != nil {
		log.Printf("Error creating request for movie %s: %v", movie.ID, err)
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	res, err := s.client.Do(req)
	if err != nil {
		log.Printf("Error fetching data for movie %s: %v", movie.ID, err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		log.Printf("Rate limited while fetching movie %s: %s , %d", movie.ID, res.Status, idx)
		return fmt.Errorf("rate limited: status %d", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("Unexpected response for movie %s: %s", movie.ID, res.Status)
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	var output struct {
		Data struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&output); err != nil {
		log.Printf("Error decoding response for movie %s: %v", movie.ID, err)
		return fmt.Errorf("response decoding failed: %w", err)
	}

	if s.isVisited(output.Data.Link) {
		log.Printf("Already visited: %s", output.Data.Link)
		return nil
	}

	log.Printf("Scraping movie: %s", movie.Title)
	s.scrapeMovieDetails(output.Data.Link, movie, idx)
	return nil
}

func (s *Scraper) scrapeMovieDetails(link string, movie *Movie, idx int) {
	proxyurl := os.Getenv("PROXY_URL")
	proxy, err := url.Parse(proxyurl)
	if err != nil {
		log.Printf("Error parsing proxy URL: %v", err)
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Printf("Error creating request for link %s: %v", link, err)
		return
	}
	req.Header.Set("Cookie", os.Getenv("FEBBOX_COOKIE"))

	req.Header.Set("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error scraping link %s: %v %d", link, err, idx)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		time.Sleep(time.Duration(2) * time.Second)
		log.Printf("Rate limited while fetching movie %s: %s %s %d", movie.ID, res.Status, link, idx)
		s.scrapeMovieDetails(link, movie, idx)
		log.Printf("Retrying after 2 seconds : %s", link)
		return
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return
	}

	var files []models.File
	doc.Find(".f_list_scroll div[data-id]").Each(func(i int, s *goquery.Selection) {
		//log.Println("reached")
		fileID, exists := s.Attr("data-id")
		if exists {
			file, _ := getFileDetails(fileID)
			if file.FID != 0 {
				files = append(files, file)
			}
		}
	})

	movieModel := &models.Movie{
		Title:       movie.Title,
		Description: movie.Description,
		MovieID:     movie.ID,
		Files:       files,
	}

	if err := s.dbRepo.CreateMovie(context.Background(), movieModel); err != nil && len(files) > 0 {
		log.Printf("Error saving movie to database: %v", err)
		return
	}

	log.Printf("Successfully saved movie: %s", movieModel.Title)
}

func (s *Scraper) scrapeMoviesConcurrently(movies []Movie, maxConcurrency int, interval time.Duration) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency) // Semaphore to limit concurrency
	ticker := time.NewTicker(interval)         // Ticker to throttle requests
	defer ticker.Stop()

	for idx, movie := range movies {
		wg.Add(1)

		go func(m Movie) {
			defer wg.Done()

			<-ticker.C // Wait for the next ticker signal to throttle requests

			sem <- struct{}{}        // Acquire a slot
			defer func() { <-sem }() // Release the slot

			for retries := 0; retries < 3; retries++ { // Retry logic
				err := s.scrapeMovie(&m, idx)
				if err != nil {
					log.Printf("Error scraping movie %s: %v", m.Title, err)
					if isRateLimitError(err) {
						log.Println("Rate limited, retrying...")
						time.Sleep(time.Duration(2<<retries) * time.Second) // Exponential backoff
						continue
					}
				}
				break // Exit loop if successful or not rate-limited
			}
		}(movie)
	}

	wg.Wait()
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	// Check for HTTP 429 or other rate-limit indicators
	return strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit")
}

func main() {
	dbcon, err := db.NewMongoConn()
	if err != nil {
		log.Fatal(err)
	}

	dbRepo := repository.NewMongoRepo(
		dbcon.Database("showbox").Collection("movies"),
		//dbcon.Database("showbox").Collection("tv"),
	)

	scraper := NewScraper(dbRepo)

	movies := getMoviesList(1701, 2000)

	maxConcurrency := 5
	requestInterval := 2 * time.Second

	scraper.scrapeMoviesConcurrently(movies, maxConcurrency, requestInterval)
}
