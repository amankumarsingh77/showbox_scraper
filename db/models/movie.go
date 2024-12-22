package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Movie struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	MovieID     string             `bson:"movie_id" json:"movie_id"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Files       []File             `bson:"files,omitempty" json:"files,omitempty"`
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
