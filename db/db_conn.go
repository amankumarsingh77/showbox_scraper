package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoConn() (*mongo.Client, error) {
	// Load .env but don't fail if it doesn't exist
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using existing environment variables")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI environment variable is not set")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, fmt.Errorf("DB_NAME environment variable is not set")
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping MongoDB
	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Ensure indexes on the "movies" collection
	if err := createMovieTextIndex(client, dbName); err != nil {
		return nil, err
	}

	// Ensure indexes on the "tv" collection
	if err := createTVTextIndex(client, dbName); err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB successfully")
	return client, nil
}

func createMovieTextIndex(client *mongo.Client, dbName string) error {
	db := client.Database(dbName)
	collection := db.Collection("movies")

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "title", Value: "text"},
			{Key: "description", Value: "text"},
		},
		Options: options.Index().SetDefaultLanguage("english"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create text index on movies collection: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.M{"movie_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create text index on movies collection: %w", err)
	}

	log.Println("Text index created on movies collection")
	return nil
}

func createTVTextIndex(client *mongo.Client, dbName string) error {
	db := client.Database(dbName)
	collection := db.Collection("tv")

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "title", Value: "text"},
			{Key: "description", Value: "text"},
		},
		Options: options.Index().SetDefaultLanguage("english"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create text index on tv collection: %w", err)
	}

	_, err = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.M{"tv_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create unique index on tv collection: %w", err)
	}

	log.Println("Text index created on tv collection")
	return nil
}
