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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gocolly/colly"
)

const (
	proxyURL    = "https://simple-proxy.xartpvt.workers.dev?destination="
	showboxBase = "https://www.showbox.media"
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

type showboxResponse struct {
	Data struct {
		Link string `json:"link"`
	} `json:"data"`
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
	visitedURLs map[string]bool // Move visitedURLs here to persist across all scrapes
}

func NewScraper(dbRepo *repository.MongoRepo) (*Scraper, error) {
	c := colly.NewCollector(
		colly.AllowedDomains("www.showbox.media", "simple-proxy.xartpvt.workers.dev", "www.febbox.com"),
		colly.UserAgent(userAgent),
	)

	c.SetRequestTimeout(120 * time.Second)

	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		RandomDelay: 3 * time.Second,
	}); err != nil {
		return nil, fmt.Errorf("failed to set rate limit: %w", err)
	}

	visitedURLs := make(map[string]bool)

	return &Scraper{
		collector:   c,
		dbRepo:      dbRepo,
		client:      &http.Client{},
		visitedURLs: visitedURLs,
	}, nil
}

func (s *Scraper) setupCollector(currentMovie *Movie) {

	s.collector.OnRequest(func(r *colly.Request) {
		if _, visited := s.visitedURLs[r.URL.String()]; visited == true {
			log.Printf("Skipping duplicate visit: %s", r.URL.String())
			r.Abort()
			return
		}
		log.Printf("Visiting: %s\n", r.URL.String())
	})

	s.collector.OnResponse(func(r *colly.Response) {
		visitedURL := r.Request.URL.String()
		// Mark the URL as visited after the response is received
		s.visitedURLs[visitedURL] = true
		log.Printf("Marked as visited: %s", visitedURL)
	})

	s.collector.OnScraped(func(r *colly.Response) {
		log.Printf("Finished scraping: %s", r.Request.URL.String())
	})

	s.collector.OnHTML(".f_list_scroll", func(e *colly.HTMLElement) {
		log.Println("reached")
		var files []models.File
		e.ForEach("div[data-id]", func(_ int, el *colly.HTMLElement) {
			fileId := el.Attr("data-id")
			if file := getFileDetails(fileId); file.FID != 0 {
				files = append(files, file)
			}
		})

		movie := &models.Movie{
			Title:       currentMovie.Title,
			Description: currentMovie.Description,
			MovieID:     currentMovie.ID,
			Files:       files,
		}

		if err := s.dbRepo.CreateMovie(context.Background(), movie); err != nil {
			log.Printf("Error saving movie to database: %v", err)
			return
		}

		log.Printf("Successfully saved movie: %s", movie.Title)
	})
}

func getFileDetails(fileid string) models.File {
	url := fmt.Sprintf("%s/file/file_info?fid=%s", feboxBase, fileid)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return models.File{}
	}
	defer resp.Body.Close()

	var data fileResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding file info: %v", err)
		return models.File{}
	}

	links := getQualities(fileid)

	return models.File{
		FID:      data.Data.File.Fid,
		FileName: data.Data.File.Filename,
		Size:     data.Data.File.Size,
		ThumbURL: data.Data.File.Thumbnail,
		Links:    links,
	}
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

func main() {
	dbcon, err := db.NewMongoConn()
	if err != nil {
		log.Fatal(err)
	}

	dbRepo := repository.NewMongoRepo(
		dbcon.Database("showbox").Collection("movies"),
		dbcon.Database("showbox").Collection("tv"),
	)

	scraper, err := NewScraper(dbRepo)
	if err != nil {
		log.Fatal(err)
	}

	movies := getMoviesList(2, 3)
	log.Println(len(movies))

	for _, movie := range movies {
		scraper.setupCollector(&movie)

		urllink := proxyURL + url.QueryEscape(fmt.Sprintf("%s/index/share_link?id=%s&type=1", showboxBase, movie.ID))
		req, err := http.NewRequest("GET", urllink, nil)
		if err != nil {
			log.Printf("Error creating request for movie %s: %v", movie.ID, err)
			continue
		}

		log.Println(urllink)

		req.Header.Set("User-Agent", userAgent)
		res, err := scraper.client.Do(req)
		if err != nil {
			log.Printf("Error getting response for movie %s: %v", movie.ID, err)
			continue
		}

		var output showboxResponse
		err = json.NewDecoder(res.Body).Decode(&output)
		res.Body.Close()
		if err != nil {
			log.Printf("Error decoding response for movie %s: %v", movie.ID, err)
			continue
		}

		if err := scraper.collector.Visit(output.Data.Link); err != nil {
			if err == colly.ErrAlreadyVisited {
				log.Printf("URL already visited: %s", output.Data.Link)
			} else {
				log.Printf("Error visiting link for movie %s: %v", movie.ID, err)
			}
		}
	}

	scraper.collector.Wait()
}
