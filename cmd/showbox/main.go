package main

import (
	"log"

	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/amankumarsingh77/go-showbox-api/scraper/febox"
)

func main() {
	// log.SetFlags(log.LstdFlags | log.Lshortfile)

	// config := showbox.DefaultConfig()
	// storage := showbox.NewStorage()

	// scraper, err := showbox.NewScraper(config, storage)
	// if err != nil {
	// 	log.Fatalf("Failed to create scraper: %v", err)
	// }

	// if err := scraper.Run(); err != nil {
	// 	log.Fatalf("Scraper error: %v", err)
	// }

	cfg := &febox.Config{
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
	scraper := febox.NewScraper(dbRepo, cfg)

	// movies := getMoviesList(1701, 2000)
	// scraper.ScrapeMoviesConcurrently(movies)

	series := febox.GetSeriesList(3732, 7789)
	scraper.ScrapeSeriesConcurrently(series)
}
