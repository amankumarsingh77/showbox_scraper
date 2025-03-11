package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Movie struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	MovieID     string             `bson:"movie_id" json:"movie_id"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Files       []File             `bson:"files,omitempty" json:"files,omitempty"`

	// TMDB related fields
	TMDBID       int                `bson:"tmdb_id,omitempty" json:"tmdb_id,omitempty"`
	IMDbID       string             `bson:"imdb_id,omitempty" json:"imdb_id,omitempty"`
	PosterPath   string             `bson:"poster_path,omitempty" json:"poster_path,omitempty"`
	BackdropPath string             `bson:"backdrop_path,omitempty" json:"backdrop_path,omitempty"`
	ReleaseDate  string             `bson:"release_date,omitempty" json:"release_date,omitempty"`
	Runtime      int                `bson:"runtime,omitempty" json:"runtime,omitempty"`
	VoteAverage  float64            `bson:"vote_average,omitempty" json:"vote_average,omitempty"`
	VoteCount    int                `bson:"vote_count,omitempty" json:"vote_count,omitempty"`
	Popularity   float64            `bson:"popularity,omitempty" json:"popularity,omitempty"`
	Genres       []Genre            `bson:"genres,omitempty" json:"genres,omitempty"`
	Cast         []Cast             `bson:"cast,omitempty" json:"cast,omitempty"`
	Crew         []Crew             `bson:"crew,omitempty" json:"crew,omitempty"`
	Videos       []Video            `bson:"videos,omitempty" json:"videos,omitempty"`
	LastUpdated  primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type File struct {
	FileName string `bson:"file_name" json:"file_name"`
	FID      int64  `bson:"fid" json:"fid"`
	Size     string `bson:"size,omitempty" json:"size,omitempty"`
	ThumbURL string `bson:"thumb_url,omitempty" json:"thumb_url,omitempty"`
	Links    []Link `bson:"links,omitempty" json:"links,omitempty"`
}

type Link struct {
	Quality string `bson:"quality" json:"quality"`
	URL     string `bson:"url" json:"url"`
	Size    string `bson:"size,omitempty" json:"size,omitempty"`
}

// TMDB related types
type Genre struct {
	ID   int    `bson:"id" json:"id"`
	Name string `bson:"name" json:"name"`
}

type Cast struct {
	ID          int    `bson:"id" json:"id"`
	Name        string `bson:"name" json:"name"`
	Character   string `bson:"character" json:"character"`
	ProfilePath string `bson:"profile_path,omitempty" json:"profile_path,omitempty"`
}

type Crew struct {
	ID          int    `bson:"id" json:"id"`
	Name        string `bson:"name" json:"name"`
	Department  string `bson:"department" json:"department"`
	Job         string `bson:"job" json:"job"`
	ProfilePath string `bson:"profile_path,omitempty" json:"profile_path,omitempty"`
}

type Video struct {
	ID       string `bson:"id" json:"id"`
	Key      string `bson:"key" json:"key"`
	Name     string `bson:"name" json:"name"`
	Site     string `bson:"site" json:"site"`
	Type     string `bson:"type" json:"type"`
	Official bool   `bson:"official" json:"official"`
}
