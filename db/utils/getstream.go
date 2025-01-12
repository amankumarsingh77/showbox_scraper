package utils

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const feboxBase = "https://www.febbox.com"

type FileInfo struct {
	Fid       int64  `json:"fid"`
	Size      string `json:"size"`
	Filename  string `json:"file_name"`
	Thumbnail string `json:"thumb_big"`
}

type VideoQuality struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Size    string `json:"size"`
}

func UpdateStream(movie *models.Movie) error {
	for i := range movie.Files {
		file := &movie.Files[i]

		url := fmt.Sprintf("%s/console/video_quality_list?fid=%s?type=1", feboxBase, strconv.FormatInt(file.FID, 10))
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return err
		}

		req.Header.Add("Cookie", os.Getenv("FEBBOX_COOKIE"))
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error getting qualities: %v", err)
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return err
		}

		var input map[string]interface{}
		if err = json.Unmarshal(body, &input); err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			return err
		}

		html, ok := input["html"].(string)
		if !ok {
			log.Println("HTML field not found in response")
			return fmt.Errorf("HTML field not found in response")
		}

		data := parseHtmlToJson(html)
		var links []models.Link
		for _, moviee := range data {
			link := models.Link{
				Quality: moviee.Quality,
				URL:     moviee.URL,
				Size:    moviee.Size,
			}
			links = append(links, link)
		}
		file.Links = links
	}
	return nil
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
