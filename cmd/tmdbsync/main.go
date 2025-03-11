package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/amankumarsingh77/go-showbox-api/db"
	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"github.com/amankumarsingh77/go-showbox-api/pkg/tmdb"
	"github.com/joho/godotenv"
)

func main() {
	// Parse command line arguments
	moviePtr := flag.Bool("movie", false, "Sync movie database with TMDB")
	tvPtr := flag.Bool("tv", false, "Sync TV database with TMDB")
	allPtr := flag.Bool("all", false, "Sync both movie and TV databases with TMDB")
	idPtr := flag.String("id", "", "Sync specific movie or TV show by ID")
	limitPtr := flag.Int("limit", 0, "Limit the number of items to sync (0 for all)")
	skipPtr := flag.Int("skip", 0, "Skip the first N items when syncing")
	verbosePtr := flag.Bool("verbose", false, "Show detailed matching information")
	tmdbApiKeyPtr := flag.String("tmdb-key", "", "TMDB API key (overrides environment variable)")

	flag.Parse()

	// Setup logger
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// If no flags are set, show usage
	if !*moviePtr && !*tvPtr && !*allPtr && *idPtr == "" {
		fmt.Println("TMDBSync - Synchronize your ShowBox database with TMDB")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  Sync all movies:   tmdbsync -movie")
		fmt.Println("  Sync all TV shows: tmdbsync -tv")
		fmt.Println("  Sync everything:   tmdbsync -all")
		fmt.Println("  Sync one movie:    tmdbsync -movie -id \"12345\"")
		fmt.Println("  Sync with limits:  tmdbsync -movie -limit 100 -skip 200")
		return
	}

	// Set verbose mode if requested
	if *verbosePtr {
		// We're already seeing this output from our enhanced logging
		fmt.Println("Verbose mode enabled - showing detailed matching information")
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using existing environment variables")
	}

	// Override TMDB API key if provided via command line
	if *tmdbApiKeyPtr != "" {
		os.Setenv("TMDB_API_KEY", *tmdbApiKeyPtr)
	}

	// Ensure TMDB API key is set
	if os.Getenv("TMDB_API_KEY") == "" {
		log.Fatal("TMDB_API_KEY environment variable is not set. Please set it in your .env file or use -tmdb-key flag.")
	}

	// Connect to MongoDB
	log.Println("Connecting to MongoDB...")
	dbConn, err := db.NewMongoConn()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "showbox" // Default DB name
	}

	// Initialize repository
	log.Println("Initializing repository...")
	dbRepo := repository.NewMongoRepo(
		dbConn.Database(dbName).Collection("movies"),
		dbConn.Database(dbName).Collection("tv"),
	)

	// Initialize TMDB sync service
	log.Println("Initializing TMDB sync service...")
	syncService, err := tmdb.NewSyncService(dbRepo)
	if err != nil {
		log.Fatalf("Failed to initialize TMDB sync service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	startTime := time.Now()

	// Sync based on flags
	if *idPtr != "" {
		// Sync specific item by ID
		if *moviePtr {
			log.Printf("Syncing specific movie with ID: %s", *idPtr)
			movie, err := dbRepo.GetMovieById(ctx, *idPtr)
			if err != nil {
				log.Fatalf("Error retrieving movie: %v", err)
			}
			if err := syncService.SyncMovie(ctx, movie); err != nil {
				log.Fatalf("Error syncing movie: %v", err)
			}
			log.Printf("Movie sync completed successfully")
		} else if *tvPtr {
			log.Printf("Syncing specific TV show with ID: %s", *idPtr)
			tv, err := dbRepo.GetTVById(ctx, *idPtr)
			if err != nil {
				log.Fatalf("Error retrieving TV show: %v", err)
			}
			if err := syncService.SyncTV(ctx, tv); err != nil {
				log.Fatalf("Error syncing TV show: %v", err)
			}
			log.Printf("TV show sync completed successfully")
		} else {
			log.Fatalf("Please specify -movie or -tv when using -id")
		}
	} else if *moviePtr || *allPtr {
		log.Println("Starting movie database sync with TMDB...")
		if *limitPtr > 0 {
			log.Printf("Limiting to %d movies, skipping first %d", *limitPtr, *skipPtr)

			// Get limited set of movies
			movies, err := dbRepo.GetMoviesWithLimitAndSkip(ctx, int64(*limitPtr), int64(*skipPtr))
			if err != nil {
				log.Printf("Error getting movies: %v", err)
			} else {
				syncMovieBatch(ctx, syncService, movies)
			}
		} else {
			// Sync all movies
			if err := syncService.SyncAllMovies(ctx); err != nil {
				log.Printf("Error syncing movies: %v", err)
			} else {
				log.Println("Movie database sync completed successfully")
			}
		}
	}

	if *tvPtr || *allPtr {
		log.Println("Starting TV database sync with TMDB...")
		if err := syncService.SyncAllTV(ctx); err != nil {
			log.Printf("Error syncing TV shows: %v", err)
		} else {
			log.Println("TV database sync completed successfully")
		}
	}

	duration := time.Since(startTime)
	log.Printf("Sync process completed in %s", duration)
}

func syncMovieBatch(ctx context.Context, syncService *tmdb.SyncService, movies []models.Movie) {
	total := len(movies)
	var syncedCount, errorCount int

	log.Printf("Starting batch sync of %d movies", total)

	for i, movie := range movies {
		log.Printf("[%d/%d] Syncing movie: %s", i+1, total, movie.Title)

		if err := syncService.SyncMovie(ctx, &movie); err != nil {
			log.Printf("Error syncing movie '%s': %v", movie.Title, err)
			errorCount++
		} else {
			syncedCount++
		}

		// Sleep briefly to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("Batch sync completed: %d/%d movies synced successfully, %d errors",
		syncedCount, total, errorCount)
}
