package repository

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	err := m.moviecol.FindOne(ctx, bson.M{"movie_id": id}).Decode(&movie)
	if movie.MovieID == "" {
		return nil, fmt.Errorf("no movie found with id %s", id)
	}
	getUpdatedStream(&movie)
	if err != nil {
		return nil, err
	}
	return &movie, nil
}

func getUpdatedStream(movie *models.Movie) {
	url := movie.Files[0].Links[0].URL
	resp, err := http.Head(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode == http.StatusGone {
		err = utils.UpdateStream(movie)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// getUpdatedEpisodeStream checks if episode links are valid and updates them if needed
func getUpdatedEpisodeStream(episode *models.Episode) {
	// Skip if there are no sources or files
	if len(episode.Sources) == 0 || len(episode.Sources[0].Files) == 0 || len(episode.Sources[0].Files[0].Links) == 0 {
		return
	}

	// Check the first link of the first file of the first source
	url := episode.Sources[0].Files[0].Links[0].URL
	resp, err := http.Head(url)
	if err != nil {
		log.Printf("Error checking episode link: %v", err)
		return
	}

	// If link is expired (status 410 Gone), update all sources
	if resp.StatusCode == http.StatusGone {
		for i := range episode.Sources {
			err = utils.UpdateEpisodeStream(&episode.Sources[i])
			if err != nil {
				log.Printf("Error updating episode source: %v", err)
			}
		}
	}
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
		movie.Files = nil
		movies = append(movies, movie)
	}
	return movies, nil
}

func (m *MongoRepo) CreateTV(ctx context.Context, tv *models.TV) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.tvcol.InsertOne(ctx, tv)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil // Ignore duplicates
		}
		return err
	}
	return nil
}

func (m *MongoRepo) GetTVById(ctx context.Context, id string) (*models.TV, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tv models.TV
	err := m.tvcol.FindOne(ctx, bson.M{"tv_id": id}).Decode(&tv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no TV series found with id %s", id)
		}
		return nil, err
	}

	if tv.TVID == "" {
		return nil, fmt.Errorf("no TV series found with id %s", id)
	}

	// Remove sources from episodes to save bandwidth when just getting show info
	for i := range tv.Seasons {
		for j := range tv.Seasons[i].Episodes {
			tv.Seasons[i].Episodes[j].Sources = nil
		}
	}

	return &tv, nil
}

// GetTVSeasonById retrieves a specific season of a TV show by ID and season number
func (m *MongoRepo) GetTVSeasonById(ctx context.Context, tvID string, seasonNum int) (*models.Season, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tv models.TV
	err := m.tvcol.FindOne(ctx, bson.M{"tv_id": tvID}).Decode(&tv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no TV series found with id %s", tvID)
		}
		return nil, err
	}

	for _, season := range tv.Seasons {
		if season.SeasonNumber == seasonNum {
			// Remove sources from episodes to save bandwidth when just getting season info
			for i := range season.Episodes {
				season.Episodes[i].Sources = nil
			}
			return &season, nil
		}
	}

	return nil, fmt.Errorf("no season %d found for TV series with id %s", seasonNum, tvID)
}

// GetTVEpisodeById retrieves a specific episode of a TV show by ID, season number, and episode number
func (m *MongoRepo) GetTVEpisodeById(ctx context.Context, tvID string, seasonNum int, episodeNum int) (*models.Episode, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tv models.TV
	err := m.tvcol.FindOne(ctx, bson.M{"tv_id": tvID}).Decode(&tv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no TV series found with id %s", tvID)
		}
		return nil, err
	}

	for _, season := range tv.Seasons {
		if season.SeasonNumber == seasonNum {
			for _, episode := range season.Episodes {
				if episode.EpisodeNo == episodeNum {
					// Create a copy to avoid modifying the stored document
					episodeCopy := episode

					// Check and update links if necessary
					getUpdatedEpisodeStream(&episodeCopy)

					return &episodeCopy, nil
				}
			}
			return nil, fmt.Errorf("no episode %d found in season %d for TV series with id %s", episodeNum, seasonNum, tvID)
		}
	}

	return nil, fmt.Errorf("no season %d found for TV series with id %s", seasonNum, tvID)
}

func (m *MongoRepo) SearchTVByQuery(ctx context.Context, query string) ([]models.TV, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tvShows []models.TV

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

	cursor, err := m.tvcol.Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var tv models.TV
		if err = cursor.Decode(&tv); err != nil {
			return nil, err
		}
		// Don't return episode details in search results to reduce payload size
		for i := range tv.Seasons {
			tv.Seasons[i].Episodes = nil
		}
		tvShows = append(tvShows, tv)
	}

	return tvShows, nil
}

func (m *MongoRepo) UpdateTV(ctx context.Context, tv *models.TV) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"tv_id": tv.TVID}
	update := bson.M{"$set": tv}

	_, err := m.tvcol.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update TV show: %w", err)
	}

	return nil
}

func (m *MongoRepo) GetAllTVShows(ctx context.Context, limit, skip int64) ([]models.TV, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	options := options.Find().SetLimit(limit).SetSkip(skip).SetSort(bson.M{"title": 1})
	cursor, err := m.tvcol.Find(ctx, bson.M{}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tvShows []models.TV
	for cursor.Next(ctx) {
		var tv models.TV
		if err = cursor.Decode(&tv); err != nil {
			return nil, err
		}
		// Don't return episode details in list results to reduce payload size
		for i := range tv.Seasons {
			tv.Seasons[i].Episodes = nil
		}
		tvShows = append(tvShows, tv)
	}

	return tvShows, nil
}

// UpdateMovie updates a movie in the database
func (m *MongoRepo) UpdateMovie(ctx context.Context, movie *models.Movie) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"movie_id": movie.MovieID}
	update := bson.M{"$set": movie}

	_, err := m.moviecol.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update movie: %w", err)
	}

	return nil
}

// GetAllMovies retrieves all movies from the database
func (m *MongoRepo) GetAllMovies(ctx context.Context) ([]models.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cursor, err := m.moviecol.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find movies: %w", err)
	}
	defer cursor.Close(ctx)

	var movies []models.Movie
	if err := cursor.All(ctx, &movies); err != nil {
		return nil, fmt.Errorf("failed to decode movies: %w", err)
	}

	return movies, nil
}

// GetMoviesWithLimitAndSkip retrieves a subset of movies from the database
func (m *MongoRepo) GetMoviesWithLimitAndSkip(ctx context.Context, limit, skip int64) ([]models.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	options := options.Find().SetLimit(limit).SetSkip(skip).SetSort(bson.M{"title": 1})
	cursor, err := m.moviecol.Find(ctx, bson.M{}, options)
	if err != nil {
		return nil, fmt.Errorf("failed to find movies: %w", err)
	}
	defer cursor.Close(ctx)

	var movies []models.Movie
	if err := cursor.All(ctx, &movies); err != nil {
		return nil, fmt.Errorf("failed to decode movies: %w", err)
	}

	log.Printf("Retrieved %d movies (limit: %d, skip: %d)", len(movies), limit, skip)
	return movies, nil
}
