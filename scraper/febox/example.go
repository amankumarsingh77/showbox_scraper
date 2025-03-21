package febox

import (
	"log"

	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
)

func main() {
	cfg := &Config{
		MaxConcurrency:  5,
		RequestInterval: 2,
		MaxRetries:      3,
		RetryDelay:      2,
		HTTPTimeout:     120,
	}

	dbConn, err := db.NewMongoConn()
	if err != nil {
		log.Fatal(err)
	}

	dbRepo := repository.NewMongoRepo(dbConn.Database("showbox").Collection("movies"), dbConn.Database("showbox").Collection("tv"))
	scraper := NewScraper(dbRepo, cfg)

	// movies := getMoviesList(1701, 2000)
	// scraper.ScrapeMoviesConcurrently(movies)

	series := GetSeriesList(1, 3)
	scraper.ScrapeSeriesConcurrently(series)
}
