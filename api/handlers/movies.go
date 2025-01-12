package handlers

import (
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	mongo *repository.MongoRepo
}

func NewHandler(db *repository.MongoRepo) *Handler {
	return &Handler{mongo: db}
}

func (h *Handler) GetMovieById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	movie, err := h.mongo.GetMovieById(c, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, movie)
}

func (h *Handler) GetMoviesByQuery(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query is required"})
		return
	}

	movies, err := h.mongo.SearchMovieByQuery(c, query)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, movies)
}
