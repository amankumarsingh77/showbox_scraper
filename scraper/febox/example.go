package febox

import (
	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"log"
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

	dbRepo := repository.NewMongoRepo(dbConn.Database("showbox").Collection("movies"))
	scraper := NewScraper(dbRepo, cfg)

	movies := getMoviesList(1701, 2000)
	scraper.ScrapeMoviesConcurrently(movies)
}
