package febox

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
)

func getMoviesList(start, end int) []models.Movie {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get working directory:", err)
	}

	filePath := filepath.Join(dir, "movies_final.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	var movies []models.Movie
	if err := json.Unmarshal(data, &movies); err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	if start < 0 || end >= len(movies) || start > end {
		log.Fatal("Invalid range for start or end index")
	}

	return movies[start : end+1]
}

func GetSeriesList(start, end int) []models.TV {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get working directory:", err)
	}

	filePath := filepath.Join(dir, "tv_final.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	var series []models.TV
	if err := json.Unmarshal(data, &series); err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	return series[start : end+1]
}

func parseHTMLToJSON(html string) []VideoQuality {
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

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "429") ||
		strings.Contains(err.Error(), "rate limit")
}

func readBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return io.ReadAll(body)
}

func checkResponseStatus(statusCode int) error {
	if statusCode == 429 {
		return ErrRateLimit
	}
	if statusCode != 200 {
		return ErrUnexpectedStatus
	}
	return nil
}

var (
	ErrRateLimit        = errors.New("rate limit exceeded")
	ErrUnexpectedStatus = errors.New("unexpected status code")
)
