package main

import (
	"log"

	"github.com/amankumarsingh77/go-showbox-api/api/handlers"
	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gin-gonic/gin"
)

func main() {
	//cfg, err := LoadConfig()
	//if err != nil {
	//	log.Fatal(err)
	//}
	con, err := db.NewMongoConn()
	if err != nil {
		log.Fatal(err)
	}

	db := con.Database("showbox")
	moviesCollection := db.Collection("movies")
	tvCollection := db.Collection("tv")

	repo := repository.NewMongoRepo(moviesCollection, tvCollection)
	handlers := handlers.NewHandler(repo)

	r := gin.Default()
	r.GET("/movies/:id", handlers.GetMovieById)
	r.GET("/movies", handlers.GetMoviesByQuery)

	// TV routes with nested structure
	r.GET("/tv/search", handlers.GetTVByQuery)                   // Search TV shows (no path parameter conflict)
	r.GET("/tv", handlers.GetAllTVShows)                         // Get all TV shows (no path parameter conflict)
	r.GET("/tv/:id/:season/:episode", handlers.GetTVEpisodeById) // Get episode details with links (most specific route)
	r.GET("/tv/:id/:season", handlers.GetTVSeasonById)           // Get season details without links (more specific route)
	r.GET("/tv/:id", handlers.GetTVById)                         // Get TV details without links (least specific)
	//r.GET("/getStream", handlers.GetStream)

	r.Run(":8080")
}
