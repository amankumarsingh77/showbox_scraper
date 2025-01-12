package main

import (
	"github.com/amankumarsingh77/go-showbox-api/api/handlers"
	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	//cfg, err := LoadConfig()
	//if err != nil {
	//	log.Fatal(err)
	//}
	con, err := db.NewMongoConn()
	repo := repository.NewMongoRepo(con.Database("showbox").Collection("movies"))
	hanlers := handlers.NewHandler(repo)
	r := gin.Default()
	r.GET("/movies/:id", hanlers.GetMovieById)
	r.GET("/movies", hanlers.GetMoviesByQuery)
	//r.GET("/getStream", hanlers.GetStream)
	if err != nil {
		log.Fatal(err)
	}
	r.Run(":8080")
}
