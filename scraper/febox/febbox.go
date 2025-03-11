package febox

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
)

// FebboxResponse represents the top level response structure
type FebboxResponse struct {
	Code          int          `json:"code"`
	Msg           string       `json:"msg"`
	ServerRuntime float64      `json:"server_runtime"`
	ServerName    string       `json:"server_name"`
	Data          ResponseData `json:"data"`
}

// ResponseData represents the data field in the response
type ResponseData struct {
	FileList []FebboxFile `json:"file_list"`
}

// FebboxFile represents a single file in the response
type FebboxFile struct {
	Fid           int    `json:"fid"`
	UID           int    `json:"uid"`
	FileSize      string `json:"file_size"`
	FileName      string `json:"file_name"`
	Ext           string `json:"ext"`
	Hash          string `json:"hash"`
	ThumbSmall    string `json:"thumb_small"`
	Thumb         string `json:"thumb"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

// Episode info from filename
type EpisodeInfo struct {
	Season  int
	Episode int
	Quality string
	Codec   string
}

// ScrapeContentType defines the type of content being scraped
type ContentType int

const (
	MovieType ContentType = 1
	TVType    ContentType = 2
)

// ScrapeContent is a general function that can scrape both movies and TV series
func (s *Scraper) ScrapeContent(content interface{}, idx int) error {
	var contentID, contentTitle string
	var contentType ContentType

	// Determine the type of content and extract necessary information
	switch v := content.(type) {
	case *models.Movie:
		contentID = v.MovieID
		contentTitle = v.Title
		contentType = MovieType
	case *models.TV:
		contentID = v.TVID
		contentTitle = v.Title
		contentType = TVType
	default:
		return fmt.Errorf("unsupported content type: %T", content)
	}
	encodedurl := url.QueryEscape(fmt.Sprintf("%s/index/share_link?id=%s&type=%d", ShowboxBase, contentID, contentType))
	shoemediaUrl := fmt.Sprintf("%s%s", ProxyURL, encodedurl)
	req, err := http.NewRequest("GET", shoemediaUrl, nil)
	if err != nil {
		log.Printf("Error creating request for content %s: %v", contentID, err)
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	res, err := s.client.Do(req)
	if err != nil {
		log.Printf("Error fetching data for content %s: %v", contentID, err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		log.Printf("Rate limited while fetching content %s: %s, %d", contentID, res.Status, idx)
		return fmt.Errorf("rate limited: status %d", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("Unexpected response for content %s: %s", contentID, res.Status)
		return fmt.Errorf("unexpected status: %d", res.StatusCode)
	}

	var output struct {
		Data struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&output); err != nil {
		log.Printf("Error decoding response for content %s: %v", contentID, err)
		return fmt.Errorf("response decoding failed: %w", err)
	}

	if s.isVisited(output.Data.Link) {
		log.Printf("Already visited: %s", output.Data.Link)
		return nil
	}

	// Process the specific content type
	switch contentType {
	case MovieType:
		log.Printf("Scraping movie: %s", contentTitle)
		s.scrapeMovieDetails(output.Data.Link, content.(*models.Movie), idx)
	case TVType:
		log.Printf("Scraping TV series: %s", contentTitle)
		s.scrapeSeriesDetails(output.Data.Link, content.(*models.TV))
	}

	return nil
}

// scrapeMovie is kept for backward compatibility
func (s *Scraper) scrapeMovie(movie *models.Movie, idx int) error {
	return s.ScrapeContent(movie, idx)
}

func (s *Scraper) scrapeSeriesDetails(link string, tv *models.TV) error {
	var err error
	maxRetries := s.config.MaxRetries
	baseDelay := time.Duration(s.config.RetryDelay) * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Printf("Retry attempt %d/%d for TV series %s, waiting for %v",
				attempt, maxRetries, tv.Title, delay)
			time.Sleep(delay)
		}

		// Attempt to scrape the series
		err = s.doScrapeSeriesDetails(link, tv)

		// If successful or it's a non-retryable error, return
		if err == nil {
			if attempt > 0 {
				log.Printf("Successfully scraped TV series %s after %d retries", tv.Title, attempt)
			}
			return nil
		}

		// If it's not a retryable error, don't retry
		if !isRetryableError(err) {
			log.Printf("Non-retryable error encountered for TV series %s: %v", tv.Title, err)
			return err
		}

		log.Printf("Retryable error encountered for TV series %s: %v", tv.Title, err)

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			log.Printf("Failed to scrape TV series %s after %d retries: %v",
				tv.Title, maxRetries, err)
			return fmt.Errorf("maximum retry attempts reached: %w", err)
		}
	}

	return err
}

// Helper function to determine if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Rate limiting is always retryable
	if strings.Contains(err.Error(), "rate limited") ||
		strings.Contains(err.Error(), "429") {
		return true
	}

	// Network-related errors are usually retryable
	if strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "reset by peer") ||
		strings.Contains(err.Error(), "EOF") {
		return true
	}

	// 5xx server errors are retryable
	if strings.Contains(err.Error(), "5") &&
		(strings.Contains(err.Error(), "status") || strings.Contains(err.Error(), "code")) {
		return true
	}

	return false
}

// Actual implementation of series details scraping
func (s *Scraper) doScrapeSeriesDetails(link string, tv *models.TV) error {
	proxyurl := os.Getenv("PROXY_URL")
	contentID := strings.Split(link, "/")[len(strings.Split(link, "/"))-1]
	proxy, err := url.Parse(proxyurl)
	if err != nil {
		log.Printf("Error parsing proxy URL: %v", err)
		return fmt.Errorf("error parsing proxy URL: %w", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
	encodedurl := url.QueryEscape(link)
	shoemediaUrl := fmt.Sprintf("%s%s", ProxyURL, encodedurl)
	log.Printf("Request URL: %s", shoemediaUrl)
	req, err := http.NewRequest("GET", shoemediaUrl, nil)
	if err != nil {
		log.Printf("Error creating request for link %s: %v", link, err)
		return fmt.Errorf("error creating request for link %s: %w", link, err)
	}
	req.Header.Set("Cookie", os.Getenv("FEBBOX_COOKIE"))

	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching data for link %s: %v", link, err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		log.Printf("Rate limited while fetching link %s: %s", link, res.Status)
		return fmt.Errorf("rate limited: status %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return fmt.Errorf("error parsing HTML: %w", err)
	}

	seasonFound := false
	doc.Find(".f_list_scroll div[data-id]").Each(func(i int, s *goquery.Selection) {
		parentID, exists := s.Attr("data-id")
		seasonName := s.Find("p.file_name").Text()
		seasonNumber := i + 1

		if exists {
			episodes, episodeErr := getSeasonsEpisodes(contentID, parentID)
			if episodeErr != nil {
				log.Printf("Error getting episodes for season %d: %v", seasonNumber, episodeErr)
				return
			}

			if len(episodes) > 0 {
				seasonFound = true
				season := models.Season{
					SeasonID:     fmt.Sprintf("season_%d", seasonNumber),
					SeasonName:   seasonName,
					SeasonNumber: seasonNumber,
					Size:         calculateTotalEpisodesSize(episodes),
					Episodes:     episodes,
				}
				log.Printf("Found season %d with %d episodes for %s",
					seasonNumber, len(episodes), tv.Title)
				tv.Seasons = append(tv.Seasons, season)
			}
		}
	})

	if !seasonFound {
		return fmt.Errorf("no valid seasons found for TV series %s", tv.Title)
	}

	// Save TV series to database
	if s.dbRepo != nil && len(tv.Seasons) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Check if TV series already exists
		existingTV, err := s.dbRepo.GetTVById(ctx, tv.TVID)
		if err != nil && !strings.Contains(err.Error(), "no TV series found") {
			log.Printf("Error checking for existing TV series: %v", err)
		}

		if existingTV != nil {
			// TV series exists, update it
			err = s.dbRepo.UpdateTV(ctx, tv)
			if err != nil {
				log.Printf("Error updating TV series %s in database: %v", tv.TVID, err)
				return fmt.Errorf("database update failed: %w", err)
			} else {
				log.Printf("Updated TV series %s (%s) in database", tv.Title, tv.TVID)
			}
		} else {
			// TV series doesn't exist, create it
			err = s.dbRepo.CreateTV(ctx, tv)
			if err != nil {
				log.Printf("Error saving TV series %s to database: %v", tv.TVID, err)
				return fmt.Errorf("database save failed: %w", err)
			} else {
				log.Printf("Saved TV series %s (%s) to database with %d seasons", tv.Title, tv.TVID, len(tv.Seasons))
			}
		}
	} else if s.dbRepo == nil {
		log.Println("Database repository not initialized, skipping TV series save")
	} else if len(tv.Seasons) == 0 {
		log.Printf("No seasons found for TV series %s, skipping save", tv.Title)
		return fmt.Errorf("no seasons found for TV series %s", tv.Title)
	}

	return nil
}

// Helper function to calculate total season size from episodes
func calculateTotalEpisodesSize(episodes []models.Episode) int {
	var totalSize int
	for _, episode := range episodes {
		totalSize += episode.Size
	}
	return totalSize
}

func getSeasonsEpisodes(shareKey, parentID string) ([]models.Episode, error) {
	url := fmt.Sprintf("%s/file/file_share_list?share_key=%s&pwd=&parent_id=%s&is_html=0", FebboxBase, shareKey, parentID)

	maxRetries := 3
	baseDelay := 2 * time.Second
	var episodes []models.Episode
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Printf("Retry attempt %d/%d for getting episodes (parent_id: %s), waiting for %v",
				attempt, maxRetries, parentID, delay)
			time.Sleep(delay)
		}

		episodes, err = doGetSeasonsEpisodes(url)

		// If successful, return the episodes
		if err == nil {
			if attempt > 0 {
				log.Printf("Successfully retrieved episodes for parent_id %s after %d retries",
					parentID, attempt)
			}
			return episodes, nil
		}

		// If it's not a retryable error, don't retry
		if !isRetryableError(err) {
			log.Printf("Non-retryable error retrieving episodes: %v", err)
			return nil, err
		}

		log.Printf("Retryable error retrieving episodes: %v", err)

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			log.Printf("Failed to retrieve episodes after %d retries: %v", maxRetries, err)
			return nil, fmt.Errorf("maximum retry attempts reached: %w", err)
		}
	}

	return nil, err
}

func doGetSeasonsEpisodes(url string) ([]models.Episode, error) {
	log.Println("Fetching episodes from URL:", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited: status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var febboxResp FebboxResponse
	if err = json.NewDecoder(resp.Body).Decode(&febboxResp); err != nil {
		log.Printf("Error decoding file info: %v", err)
		return nil, err
	}

	if febboxResp.Code != 1 {
		return nil, fmt.Errorf("API error: %s (code: %d)", febboxResp.Msg, febboxResp.Code)
	}

	return processFileList(febboxResp.Data.FileList)
}

func (s *Scraper) scrapeMovieDetails(link string, movie *models.Movie, idx int) {
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
	req, err := http.NewRequest("GET", ProxyURL+link, nil)
	if err != nil {
		log.Printf("Error creating request for link %s: %v", link, err)
		return
	}
	req.Header.Set("Cookie", os.Getenv("FEBBOX_COOKIE"))

	req.Header.Set("User-Agent", UserAgent)
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
		MovieID:     movie.ID.String(),
		Files:       files,
	}

	if err := s.dbRepo.CreateMovie(context.Background(), movieModel); err != nil && len(files) > 0 {
		log.Printf("Error saving movie to database: %v", err)
		return
	}

	log.Printf("Successfully saved movie: %s", movieModel.Title)
}

func getFileDetails(fileid string) (models.File, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second
	var detailedFile models.File
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			log.Printf("Retry attempt %d/%d for file details (fid: %s), waiting for %v",
				attempt, maxRetries, fileid, delay)
			time.Sleep(delay)
		}

		detailedFile, err = doGetFileDetails(fileid)

		// If successful, return the file details
		if err == nil {
			if attempt > 0 {
				log.Printf("Successfully retrieved file details for fid %s after %d retries",
					fileid, attempt)
			}
			return detailedFile, nil
		}

		// If it's not a retryable error, don't retry
		if !isRetryableError(err) {
			log.Printf("Non-retryable error retrieving file details: %v", err)
			return models.File{}, err
		}

		log.Printf("Retryable error retrieving file details: %v", err)

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			log.Printf("Failed to retrieve file details after %d retries: %v", maxRetries, err)
			// Fall back to basic file information instead of failing completely
			return models.File{
				FID:      0,  // This will be replaced with the FID from FebboxFile
				FileName: "", // This will be replaced with filename from FebboxFile
			}, nil
		}
	}

	return models.File{}, err
}

func doGetFileDetails(fileid string) (models.File, error) {
	url := fmt.Sprintf("%s/file/file_info?fid=%s", FebboxBase, fileid)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return models.File{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return models.File{}, fmt.Errorf("rate limited: status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return models.File{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data FileResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding file info: %v", err)
		return models.File{}, err
	}

	if data.Data.File.Fid == 0 {
		return models.File{}, fmt.Errorf("invalid or empty file data returned")
	}

	links := GetQualities(fileid)

	return models.File{
		FID:      data.Data.File.Fid,
		FileName: data.Data.File.Filename,
		Size:     data.Data.File.Size,
		ThumbURL: data.Data.File.Thumbnail,
		Links:    links,
	}, nil
}

// Helper function to extract season and episode info from filename
func extractEpisodeInfo(filename string) (EpisodeInfo, error) {
	// Common regex patterns for TV episode filenames
	patterns := []string{
		`[Ss](\d+)[Ee](\d+)`, // S03E05, s03e05
		`\.(\d+)x(\d+)\.`,    // .3x05.
	}

	var season, episode int
	var matched bool

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(filename)
		if len(matches) == 3 {
			season, _ = strconv.Atoi(matches[1])
			episode, _ = strconv.Atoi(matches[2])
			matched = true
			break
		}
	}

	if !matched {
		return EpisodeInfo{}, fmt.Errorf("could not extract episode info from: %s", filename)
	}

	// Extract quality and codec info
	quality := "Standard"
	if strings.Contains(filename, "1080p") {
		quality = "1080p"
	} else if strings.Contains(filename, "720p") {
		quality = "720p"
	} else if strings.Contains(filename, "2160p") || strings.Contains(filename, "4K") {
		quality = "4K"
	}

	codec := "Unknown"
	if strings.Contains(filename, "x265") || strings.Contains(filename, "HEVC") {
		codec = "HEVC/x265"
	} else if strings.Contains(filename, "x264") || strings.Contains(filename, "h264") {
		codec = "H.264/x264"
	} else if strings.Contains(filename, "AV1") {
		codec = "AV1"
	}

	return EpisodeInfo{
		Season:  season,
		Episode: episode,
		Quality: quality,
		Codec:   codec,
	}, nil
}

// Process the file list and convert it to Episodes
func processFileList(files []FebboxFile) ([]models.Episode, error) {
	// Group files by episode
	episodeMap := make(map[string][]FebboxFile)

	for _, file := range files {
		info, err := extractEpisodeInfo(file.FileName)
		if err != nil {
			// Skip files where we can't extract episode info
			continue
		}

		// Create a key in format "S{season}E{episode}" (e.g., "S3E5")
		key := fmt.Sprintf("S%dE%d", info.Season, info.Episode)
		episodeMap[key] = append(episodeMap[key], file)
	}

	// Create episodes from grouped files
	var episodes []models.Episode

	for key, files := range episodeMap {
		// Extract season and episode numbers from the key
		var seasonNum, episodeNum int
		fmt.Sscanf(key, "S%dE%d", &seasonNum, &episodeNum)

		// Create a new episode
		episode := models.Episode{
			EpisodeID:   generateID(key),
			EpisodeName: fmt.Sprintf("Episode %d", episodeNum),
			EpisodeNo:   episodeNum,
			Size:        calculateTotalSize(files),
			Sources:     groupFilesBySource(files),
		}

		episodes = append(episodes, episode)
	}

	// Sort episodes by episode number
	sort.Slice(episodes, func(i, j int) bool {
		return episodes[i].EpisodeNo < episodes[j].EpisodeNo
	})

	return episodes, nil
}

// Group files by source, creating Source structs
func groupFilesBySource(files []FebboxFile) []models.Source {
	// Group files by source (using codec as the grouping factor)
	sourceMap := make(map[string][]FebboxFile)

	for _, file := range files {
		info, _ := extractEpisodeInfo(file.FileName)
		sourceMap[info.Codec] = append(sourceMap[info.Codec], file)
	}

	// Create Source structs from grouped files
	var sources []models.Source

	for codec, files := range sourceMap {
		source := models.Source{
			SourceID:   generateID(codec),
			SourceName: codec,
			Files:      createEpisodeFiles(files),
		}
		sources = append(sources, source)
	}

	return sources
}

// Create File structs from FebboxFile
func createEpisodeFiles(files []FebboxFile) []models.File {
	var episodeFiles []models.File

	// Use a channel with buffer to limit concurrent API calls
	sem := make(chan struct{}, 5)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go func(file FebboxFile) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			// Try to get detailed file information
			fileID := strconv.Itoa(file.Fid)
			detailedFile, err := getFileDetails(fileID)

			// If there was an error or if the FID is 0 (fallback empty file), create a basic file
			if err != nil || detailedFile.FID == 0 {
				// Fall back to basic information
				detailedFile = models.File{
					FileName: file.FileName,
					FID:      int64(file.Fid),
					Size:     file.FileSize,
					ThumbURL: file.Thumb,
					// Note: Links will be empty here
				}

				if err != nil {
					log.Printf("Using fallback file info for %s (fid: %d): %v",
						file.FileName, file.Fid, err)
				}
			}

			mu.Lock()
			episodeFiles = append(episodeFiles, detailedFile)
			mu.Unlock()

		}(file)
	}

	wg.Wait()

	// Sort the files to ensure consistent ordering despite concurrent processing
	sort.Slice(episodeFiles, func(i, j int) bool {
		return episodeFiles[i].FID < episodeFiles[j].FID
	})

	return episodeFiles
}

// Helper function to generate a unique ID
func generateID(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Calculate total size from all files
func calculateTotalSize(files []FebboxFile) int {
	var totalSize int64
	for _, file := range files {
		totalSize += file.FileSizeBytes
	}
	// Convert to MB for the Size field
	return int(totalSize / (1024 * 1024))
}

func GetQualities(fileId string) []models.Link {
	url := fmt.Sprintf("%s/console/video_quality_list?fid=%s?type=1", FebboxBase, fileId)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	req, err := http.NewRequest("GET", ProxyURL+url, nil)
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
