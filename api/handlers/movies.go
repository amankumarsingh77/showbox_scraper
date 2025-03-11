package handlers

import (
	"net/http"
	"strconv"

	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/gin-gonic/gin"
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

func (h *Handler) GetTVById(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	tv, err := h.mongo.GetTVById(c, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tv)
}

// GetTVSeasonById handles GET /tv/:id/:season
func (h *Handler) GetTVSeasonById(c *gin.Context) {
	id := c.Param("id")
	seasonParam := c.Param("season")

	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	seasonNum, err := strconv.Atoi(seasonParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "season must be a number"})
		return
	}

	season, err := h.mongo.GetTVSeasonById(c, id, seasonNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, season)
}

// GetTVEpisodeById handles GET /tv/:id/:season/:episode
func (h *Handler) GetTVEpisodeById(c *gin.Context) {
	id := c.Param("id")
	seasonParam := c.Param("season")
	episodeParam := c.Param("episode")

	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	seasonNum, err := strconv.Atoi(seasonParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "season must be a number"})
		return
	}

	episodeNum, err := strconv.Atoi(episodeParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "episode must be a number"})
		return
	}

	episode, err := h.mongo.GetTVEpisodeById(c, id, seasonNum, episodeNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, episode)
}

func (h *Handler) GetTVByQuery(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(400, gin.H{"error": "query is required"})
		return
	}

	tvShows, err := h.mongo.SearchTVByQuery(c, query)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tvShows)
}

func (h *Handler) GetAllTVShows(c *gin.Context) {
	limit := 20
	skip := 0

	if c.Query("limit") != "" {
		limitInt, err := strconv.Atoi(c.Query("limit"))
		if err == nil && limitInt > 0 {
			limit = limitInt
		}
	}

	if c.Query("skip") != "" {
		skipInt, err := strconv.Atoi(c.Query("skip"))
		if err == nil && skipInt >= 0 {
			skip = skipInt
		}
	}

	tvShows, err := h.mongo.GetAllTVShows(c, int64(limit), int64(skip))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tvShows)
}
