package febox

import (
	"time"
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

type FileResponse struct {
	Data struct {
		File FileInfo `json:"file"`
	} `json:"data"`
}

type VideoQuality struct {
	Quality string `json:"quality"`
	URL     string `json:"url"`
	Size    string `json:"size"`
}
