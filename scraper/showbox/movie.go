package showbox

import "time"

type Movie struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ReleaseDate string    `json:"release_date"`
	Genre       string    `json:"genre"`
	Casts       string    `json:"casts"`
	Duration    string    `json:"duration"`
	Country     string    `json:"country"`
	Production  string    `json:"production"`
	IMDBRating  string    `json:"imdb_rating"`
	ScrapedAt   time.Time `json:"scraped_at"`
}
