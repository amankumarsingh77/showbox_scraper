package repository

import (
	"context"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type MongoRepo struct {
	moviecol *mongo.Collection
	tvcol    *mongo.Collection
}

func NewMongoRepo(moviecol *mongo.Collection, tvcol *mongo.Collection) *MongoRepo {
	return &MongoRepo{
		moviecol: moviecol,
		tvcol:    tvcol,
	}
}
func (m *MongoRepo) CreateMovie(ctx context.Context, movie *models.Movie) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.moviecol.InsertOne(ctx, movie)
	if err != nil {
		// Check if the error is a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			return nil // Ignore duplicates
		}
		return err
	}
	return nil
}

func (m *MongoRepo) GetMovieById(ctx context.Context, id string) (*models.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var movie models.Movie
	err := m.moviecol.FindOne(ctx, bson.M{"_id": id}).Decode(&movie)
	if err != nil {
		return nil, err
	}
	return &movie, nil
}

func (m *MongoRepo) SearchMovieByQuery(ctx context.Context, query string) ([]models.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var movies []models.Movie

	filter := bson.M{
		"$text": bson.M{"$search": query},
	}
	opt := options.Find().SetProjection(
		bson.M{
			"score": bson.M{
				"$meta": "textScore",
			},
		}).SetSort(
		bson.M{
			"score": bson.M{
				"$meta": "textScore",
			},
		})
	cursor, err := m.moviecol.Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var movie models.Movie
		if err = cursor.Decode(&movie); err != nil {
			return nil, err
		}
		movies = append(movies, movie)
	}
	return movies, nil
}
