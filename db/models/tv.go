package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type TV struct {
	ID          string   `bson:"_id,omitempty" json:"_id,omitempty"`
	Title       string   `bson:"title" json:"title"`
	TVID        string   `bson:"tv_id" json:"id"`
	Description string   `bson:"description,omitempty" json:"description,omitempty"`
	Seasons     []Season `bson:"seasons,omitempty" json:"seasons,omitempty"`

	// TMDB related fields
	TMDBID           int                `bson:"tmdb_id,omitempty" json:"tmdb_id,omitempty"`
	PosterPath       string             `bson:"poster_path,omitempty" json:"poster_path,omitempty"`
	BackdropPath     string             `bson:"backdrop_path,omitempty" json:"backdrop_path,omitempty"`
	FirstAirDate     string             `bson:"first_air_date,omitempty" json:"first_air_date,omitempty"`
	LastAirDate      string             `bson:"last_air_date,omitempty" json:"last_air_date,omitempty"`
	Status           string             `bson:"status,omitempty" json:"status,omitempty"`
	VoteAverage      float64            `bson:"vote_average,omitempty" json:"vote_average,omitempty"`
	VoteCount        int                `bson:"vote_count,omitempty" json:"vote_count,omitempty"`
	Popularity       float64            `bson:"popularity,omitempty" json:"popularity,omitempty"`
	Genres           []Genre            `bson:"genres,omitempty" json:"genres,omitempty"`
	Networks         []Network          `bson:"networks,omitempty" json:"networks,omitempty"`
	NumberOfSeasons  int                `bson:"number_of_seasons,omitempty" json:"number_of_seasons,omitempty"`
	NumberOfEpisodes int                `bson:"number_of_episodes,omitempty" json:"number_of_episodes,omitempty"`
	Cast             []Cast             `bson:"cast,omitempty" json:"cast,omitempty"`
	Crew             []Crew             `bson:"crew,omitempty" json:"crew,omitempty"`
	Videos           []Video            `bson:"videos,omitempty" json:"videos,omitempty"`
	LastUpdated      primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type Season struct {
	SeasonID     string    `bson:"season_id" json:"season_id"`
	SeasonName   string    `bson:"season_name" json:"season_name"`
	SeasonNumber int       `bson:"season_number" json:"season_number"`
	Size         int       `bson:"size" json:"size"`
	Episodes     []Episode `bson:"episodes,omitempty" json:"episodes,omitempty"`

	// TMDB related fields
	TMDBID     int    `bson:"tmdb_id,omitempty" json:"tmdb_id,omitempty"`
	AirDate    string `bson:"air_date,omitempty" json:"air_date,omitempty"`
	PosterPath string `bson:"poster_path,omitempty" json:"poster_path,omitempty"`
}

type Episode struct {
	EpisodeID   string   `bson:"episode_id" json:"episode_id"`
	EpisodeName string   `bson:"episode_name" json:"episode_name"`
	EpisodeNo   int      `bson:"episode_no" json:"episode_no"`
	Size        int      `bson:"size" json:"size"`
	Sources     []Source `bson:"sources,omitempty" json:"sources,omitempty"`

	// TMDB related fields
	TMDBID      int     `bson:"tmdb_id,omitempty" json:"tmdb_id,omitempty"`
	AirDate     string  `bson:"air_date,omitempty" json:"air_date,omitempty"`
	StillPath   string  `bson:"still_path,omitempty" json:"still_path,omitempty"`
	Overview    string  `bson:"overview,omitempty" json:"overview,omitempty"`
	VoteAverage float64 `bson:"vote_average,omitempty" json:"vote_average,omitempty"`
	VoteCount   int     `bson:"vote_count,omitempty" json:"vote_count,omitempty"`
}

type Source struct {
	SourceID   string `bson:"source_id" json:"source_id"`
	SourceName string `bson:"source_name" json:"source_name"`
	Files      []File `bson:"files,omitempty" json:"files,omitempty"`
}

// TMDB related types

type Network struct {
	ID            int    `bson:"id" json:"id"`
	Name          string `bson:"name" json:"name"`
	LogoPath      string `bson:"logo_path,omitempty" json:"logo_path,omitempty"`
	OriginCountry string `bson:"origin_country,omitempty" json:"origin_country,omitempty"`
}
