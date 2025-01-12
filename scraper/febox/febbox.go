package febox

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func (s *Scraper) scrapeMovie(movie *Movie, idx int) error {
	shoemediaUrl := fmt.Sprintf("%s/index/share_link?id=%s&type=1", ShowboxBase, movie.ID)
	req, err := http.NewRequest("GET", shoemediaUrl, nil)
	if err != nil {
		log.Printf("Error creating request for movie %s: %v", movie.ID, err)
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

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
		MovieID:     movie.ID,
		Files:       files,
	}

	if err := s.dbRepo.CreateMovie(context.Background(), movieModel); err != nil && len(files) > 0 {
		log.Printf("Error saving movie to database: %v", err)
		return
	}

	log.Printf("Successfully saved movie: %s", movieModel.Title)
}

func getFileDetails(fileid string) (models.File, error) {
	url := fmt.Sprintf("%s/file/file_info?fid=%s", FebboxBase, fileid)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return models.File{}, nil
	}
	defer resp.Body.Close()

	var data FileResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding file info: %v", err)
		return models.File{}, nil
	}

	links := GetQualities(fileid)
	//log.Println(links)

	return models.File{
		FID:      data.Data.File.Fid,
		FileName: data.Data.File.Filename,
		Size:     data.Data.File.Size,
		ThumbURL: data.Data.File.Thumbnail,
		Links:    links,
	}, nil
}

func GetQualities(fileId string) []models.Link {
	url := fmt.Sprintf("%s/console/video_quality_list?fid=%s?type=1", FebboxBase, fileId)
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
