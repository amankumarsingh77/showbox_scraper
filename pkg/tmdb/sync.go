package tmdb

import (
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/amankumarsingh77/go-showbox-api/db/models"
	"github.com/amankumarsingh77/go-showbox-api/db/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SyncService handles the synchronization between the local database and TMDB
type SyncService struct {
	tmdbClient *Client
	repo       *repository.MongoRepo
}

// NewSyncService creates a new sync service
func NewSyncService(repo *repository.MongoRepo) (*SyncService, error) {
	tmdbClient, err := NewClient()
	if err != nil {
		return nil, err
	}

	return &SyncService{
		tmdbClient: tmdbClient,
		repo:       repo,
	}, nil
}

// SyncMovie synchronizes a single movie with TMDB
func (s *SyncService) SyncMovie(ctx context.Context, movie *models.Movie) error {
	log.Printf("Starting sync for movie: %s (ID: %s)", movie.Title, movie.MovieID)

	// First, try to find by TMDB ID if it exists
	if movie.TMDBID != 0 {
		log.Printf("Movie already has TMDB ID: %d, fetching updated details", movie.TMDBID)
		details, err := s.tmdbClient.GetMovieDetails(movie.TMDBID)
		if err == nil {
			s.updateMovieFromTMDB(movie, details)
			return s.repo.UpdateMovie(ctx, movie)
		}
		log.Printf("Error fetching existing TMDB details: %v, will try searching by title", err)
		// If error, continue to search by title
	}

	// Extract year from file name if available
	yearFromFile := ""
	if len(movie.Files) > 0 && movie.Files[0].FileName != "" {
		yearFromFile = extractYearFromFileName(movie.Files[0].FileName)
		if yearFromFile != "" {
			log.Printf("Extracted year from file name: %s", yearFromFile)
		}
	}

	// Search query - if we have a year from the file name, add it to improve search
	searchQuery := movie.Title
	if yearFromFile != "" {
		searchQuery = fmt.Sprintf("%s %s", movie.Title, yearFromFile)
		log.Printf("Searching TMDB for movie with title and year: %s", searchQuery)
	} else {
		log.Printf("Searching TMDB for movie: %s", movie.Title)
	}

	searchResp, err := s.tmdbClient.SearchMovie(searchQuery)
	if err != nil {
		return fmt.Errorf("failed to search for movie: %w", err)
	}

	log.Printf("Found %d potential matches on TMDB", len(searchResp.Results))

	if len(searchResp.Results) == 0 {
		// If no results with year, try just the title
		if yearFromFile != "" && searchQuery != movie.Title {
			log.Printf("No results found with year. Trying with title only: %s", movie.Title)
			searchResp, err = s.tmdbClient.SearchMovie(movie.Title)
			if err != nil {
				return fmt.Errorf("failed to search for movie: %w", err)
			}

			if len(searchResp.Results) == 0 {
				return fmt.Errorf("no matches found for movie: %s", movie.Title)
			}
			log.Printf("Found %d potential matches using title only", len(searchResp.Results))
		} else {
			return fmt.Errorf("no matches found for movie: %s", movie.Title)
		}
	}

	// Find the best match, passing the year from file name for better matching
	bestMatch := findBestMovieMatch(movie.Title, searchResp.Results, yearFromFile)
	if bestMatch == nil {
		return fmt.Errorf("couldn't find a good match for movie: %s", movie.Title)
	}

	log.Printf("Best match: %s (%s) - TMDB ID: %d",
		bestMatch.Title,
		getYearFromDate(bestMatch.ReleaseDate),
		bestMatch.ID)

	// Get detailed information
	details, err := s.tmdbClient.GetMovieDetails(bestMatch.ID)
	if err != nil {
		return fmt.Errorf("failed to get movie details: %w", err)
	}

	// Update the movie with TMDB data
	s.updateMovieFromTMDB(movie, details)
	log.Printf("Updated movie metadata from TMDB: %s", movie.Title)

	// Save the updated movie
	return s.repo.UpdateMovie(ctx, movie)
}

// SyncTV synchronizes a single TV show with TMDB
func (s *SyncService) SyncTV(ctx context.Context, tv *models.TV) error {
	log.Printf("Starting sync for TV show: %s (ID: %s)", tv.Title, tv.TVID)

	// First, try to find by TMDB ID if it exists
	if tv.TMDBID != 0 {
		log.Printf("TV show already has TMDB ID: %d, fetching updated details", tv.TMDBID)
		details, err := s.tmdbClient.GetTVDetails(tv.TMDBID)
		if err == nil {
			s.updateTVFromTMDB(tv, details)
			if err := s.syncTVSeasons(ctx, tv, details); err != nil {
				log.Printf("Warning: error syncing seasons for TV %s: %v", tv.Title, err)
			}
			return s.repo.UpdateTV(ctx, tv)
		}
		log.Printf("Error fetching existing TMDB details: %v, will try searching by title", err)
		// If error, continue to search by title
	}

	// Extract year from file names if available
	yearFromFile := ""
	if len(tv.Seasons) > 0 && len(tv.Seasons[0].Episodes) > 0 &&
		len(tv.Seasons[0].Episodes[0].Sources) > 0 &&
		len(tv.Seasons[0].Episodes[0].Sources[0].Files) > 0 {
		// Try to get the year from the first episode file name
		fileName := tv.Seasons[0].Episodes[0].Sources[0].Files[0].FileName
		if fileName != "" {
			yearFromFile = extractYearFromFileName(fileName)
			if yearFromFile != "" {
				log.Printf("Extracted year from file name: %s", yearFromFile)
			}
		}
	}

	// Search query - if we have a year from the file name, add it to improve search
	searchQuery := tv.Title
	if yearFromFile != "" {
		searchQuery = fmt.Sprintf("%s %s", tv.Title, yearFromFile)
		log.Printf("Searching TMDB for TV show with title and year: %s", searchQuery)
	} else {
		log.Printf("Searching TMDB for TV show: %s", tv.Title)
	}

	searchResp, err := s.tmdbClient.SearchTV(searchQuery)
	if err != nil {
		return fmt.Errorf("failed to search for TV show: %w", err)
	}

	log.Printf("Found %d potential matches on TMDB", len(searchResp.Results))

	if len(searchResp.Results) == 0 {
		// If no results with year, try just the title
		if yearFromFile != "" && searchQuery != tv.Title {
			log.Printf("No results found with year. Trying with title only: %s", tv.Title)
			searchResp, err = s.tmdbClient.SearchTV(tv.Title)
			if err != nil {
				return fmt.Errorf("failed to search for TV show: %w", err)
			}

			if len(searchResp.Results) == 0 {
				return fmt.Errorf("no matches found for TV show: %s", tv.Title)
			}
			log.Printf("Found %d potential matches using title only", len(searchResp.Results))
		} else {
			return fmt.Errorf("no matches found for TV show: %s", tv.Title)
		}
	}

	// Find the best match, passing the year from file name for better matching
	bestMatch := findBestTVMatch(tv.Title, searchResp.Results, yearFromFile)
	if bestMatch == nil {
		return fmt.Errorf("couldn't find a good match for TV show: %s", tv.Title)
	}

	log.Printf("Best match: %s (%s) - TMDB ID: %d",
		bestMatch.Name,
		getYearFromDate(bestMatch.FirstAirDate),
		bestMatch.ID)

	// Get detailed information
	details, err := s.tmdbClient.GetTVDetails(bestMatch.ID)
	if err != nil {
		return fmt.Errorf("failed to get TV details: %w", err)
	}

	// Update the TV show with TMDB data
	s.updateTVFromTMDB(tv, details)
	log.Printf("Updated TV show metadata from TMDB: %s", tv.Title)

	// Sync seasons and episodes
	log.Printf("Syncing seasons and episodes for %s", tv.Title)
	if err := s.syncTVSeasons(ctx, tv, details); err != nil {
		log.Printf("Warning: error syncing seasons for TV %s: %v", tv.Title, err)
	}

	// Save the updated TV show
	return s.repo.UpdateTV(ctx, tv)
}

// SyncAllMovies synchronizes all movies in the database with TMDB
func (s *SyncService) SyncAllMovies(ctx context.Context) error {
	movies, err := s.repo.GetAllMovies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all movies: %w", err)
	}

	for i, movie := range movies {
		log.Printf("Syncing movie %d/%d: %s", i+1, len(movies), movie.Title)
		if err := s.SyncMovie(ctx, &movie); err != nil {
			log.Printf("Error syncing movie '%s': %v", movie.Title, err)
		}
		// Sleep to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// SyncAllTV synchronizes all TV shows in the database with TMDB
func (s *SyncService) SyncAllTV(ctx context.Context) error {
	tvShows, err := s.repo.GetAllTVShows(ctx, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get all TV shows: %w", err)
	}

	for i, tv := range tvShows {
		log.Printf("Syncing TV show %d/%d: %s", i+1, len(tvShows), tv.Title)
		if err := s.SyncTV(ctx, &tv); err != nil {
			log.Printf("Error syncing TV show '%s': %v", tv.Title, err)
		}
		// Sleep to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// Helper functions

func (s *SyncService) updateMovieFromTMDB(movie *models.Movie, details *MovieDetails) {
	movie.TMDBID = details.ID
	movie.Description = details.Overview
	movie.IMDbID = details.IMDbID
	movie.PosterPath = details.PosterPath
	movie.BackdropPath = details.BackdropPath
	movie.ReleaseDate = details.ReleaseDate
	movie.Runtime = details.Runtime
	movie.VoteAverage = details.VoteAverage
	movie.VoteCount = details.VoteCount
	movie.Popularity = details.Popularity

	// Convert genres
	movie.Genres = make([]models.Genre, len(details.Genres))
	for i, g := range details.Genres {
		movie.Genres[i] = models.Genre{
			ID:   g.ID,
			Name: g.Name,
		}
	}

	// Convert cast
	if details.Credits.Cast != nil {
		movie.Cast = make([]models.Cast, 0, min(10, len(details.Credits.Cast)))
		for i, c := range details.Credits.Cast {
			if i >= 10 {
				break // Limit to top 10 cast members
			}
			movie.Cast = append(movie.Cast, models.Cast{
				ID:          c.ID,
				Name:        c.Name,
				Character:   c.Character,
				ProfilePath: c.ProfilePath,
			})
		}
	}

	// Convert crew (directors, writers, etc.)
	if details.Credits.Crew != nil {
		movie.Crew = make([]models.Crew, 0)
		for _, c := range details.Credits.Crew {
			// Only include important crew roles
			if c.Job == "Director" || c.Job == "Writer" || c.Job == "Producer" || c.Job == "Screenplay" {
				movie.Crew = append(movie.Crew, models.Crew{
					ID:          c.ID,
					Name:        c.Name,
					Department:  c.Department,
					Job:         c.Job,
					ProfilePath: c.ProfilePath,
				})
			}
		}
	}

	// Convert videos (trailers, teasers, etc.)
	if details.Videos.Results != nil {
		movie.Videos = make([]models.Video, 0)
		for _, v := range details.Videos.Results {
			// Only include YouTube videos that are trailers or teasers
			if v.Site == "YouTube" && (v.Type == "Trailer" || v.Type == "Teaser") {
				movie.Videos = append(movie.Videos, models.Video{
					ID:       v.ID,
					Key:      v.Key,
					Name:     v.Name,
					Site:     v.Site,
					Type:     v.Type,
					Official: v.Official,
				})
			}
		}
	}

	// Update timestamp
	movie.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
}

func (s *SyncService) updateTVFromTMDB(tv *models.TV, details *TVDetails) {
	tv.TMDBID = details.ID
	tv.Description = details.Overview
	tv.PosterPath = details.PosterPath
	tv.BackdropPath = details.BackdropPath
	tv.FirstAirDate = details.FirstAirDate
	tv.LastAirDate = details.LastAirDate
	tv.Status = details.Status
	tv.VoteAverage = details.VoteAverage
	tv.VoteCount = details.VoteCount
	tv.Popularity = details.Popularity
	tv.NumberOfSeasons = details.NumberOfSeasons
	tv.NumberOfEpisodes = details.NumberOfEpisodes

	// Convert genres
	tv.Genres = make([]models.Genre, len(details.Genres))
	for i, g := range details.Genres {
		tv.Genres[i] = models.Genre{
			ID:   g.ID,
			Name: g.Name,
		}
	}

	// Convert networks
	tv.Networks = make([]models.Network, len(details.Networks))
	for i, n := range details.Networks {
		tv.Networks[i] = models.Network{
			ID:            n.ID,
			Name:          n.Name,
			LogoPath:      n.LogoPath,
			OriginCountry: n.OriginCountry,
		}
	}

	// Convert cast
	if details.Credits.Cast != nil {
		tv.Cast = make([]models.Cast, 0, min(10, len(details.Credits.Cast)))
		for i, c := range details.Credits.Cast {
			if i >= 10 {
				break // Limit to top 10 cast members
			}
			tv.Cast = append(tv.Cast, models.Cast{
				ID:          c.ID,
				Name:        c.Name,
				Character:   c.Character,
				ProfilePath: c.ProfilePath,
			})
		}
	}

	// Convert crew (creators, etc.)
	if details.Credits.Crew != nil {
		tv.Crew = make([]models.Crew, 0)
		for _, c := range details.Credits.Crew {
			// Only include important crew roles
			if c.Job == "Creator" || c.Job == "Executive Producer" || c.Job == "Director" {
				tv.Crew = append(tv.Crew, models.Crew{
					ID:          c.ID,
					Name:        c.Name,
					Department:  c.Department,
					Job:         c.Job,
					ProfilePath: c.ProfilePath,
				})
			}
		}
	}

	// Convert videos (trailers, teasers, etc.)
	if details.Videos.Results != nil {
		tv.Videos = make([]models.Video, 0)
		for _, v := range details.Videos.Results {
			// Only include YouTube videos that are trailers or teasers
			if v.Site == "YouTube" && (v.Type == "Trailer" || v.Type == "Teaser") {
				tv.Videos = append(tv.Videos, models.Video{
					ID:       v.ID,
					Key:      v.Key,
					Name:     v.Name,
					Site:     v.Site,
					Type:     v.Type,
					Official: v.Official,
				})
			}
		}
	}

	// Update timestamp
	tv.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
}

func (s *SyncService) syncTVSeasons(ctx context.Context, tv *models.TV, details *TVDetails) error {
	// Create a map of our existing seasons for easier lookup
	seasonMap := make(map[int]*models.Season)
	for i := range tv.Seasons {
		seasonMap[tv.Seasons[i].SeasonNumber] = &tv.Seasons[i]
	}

	// Process each season from TMDB
	for _, tmdbSeason := range details.Seasons {
		// Skip season 0 (usually specials)
		if tmdbSeason.SeasonNumber == 0 {
			continue
		}

		// Find or create the season in our data
		season, exists := seasonMap[tmdbSeason.SeasonNumber]
		if !exists {
			// This is a new season that doesn't exist in our data yet
			// We'll skip it as we don't have files for it
			continue
		}

		// Update season with TMDB data
		season.TMDBID = tmdbSeason.ID
		season.SeasonName = tmdbSeason.Name
		season.AirDate = tmdbSeason.AirDate
		season.PosterPath = tmdbSeason.PosterPath

		// Get detailed season information
		seasonDetails, err := s.tmdbClient.GetTVSeasonDetails(tv.TMDBID, tmdbSeason.SeasonNumber)
		if err != nil {
			return fmt.Errorf("failed to get season details: %w", err)
		}

		// Create a map of our existing episodes for easier lookup
		episodeMap := make(map[int]*models.Episode)
		for i := range season.Episodes {
			episodeMap[season.Episodes[i].EpisodeNo] = &season.Episodes[i]
		}

		// Process each episode from TMDB
		for _, tmdbEpisode := range seasonDetails.Episodes {
			// Find the episode in our data
			episode, exists := episodeMap[tmdbEpisode.EpisodeNumber]
			if !exists {
				// This episode doesn't exist in our data yet
				// We'll skip it as we don't have files for it
				continue
			}

			// Update episode with TMDB data
			episode.TMDBID = tmdbEpisode.ID
			episode.EpisodeName = tmdbEpisode.Name
			episode.AirDate = tmdbEpisode.AirDate
			episode.StillPath = tmdbEpisode.StillPath
			episode.Overview = tmdbEpisode.Overview
			episode.VoteAverage = tmdbEpisode.VoteAverage
			episode.VoteCount = tmdbEpisode.VoteCount
		}
	}

	return nil
}

// findBestMovieMatch finds the best match from TMDB results for a movie title
func findBestMovieMatch(title string, results []MovieResult, yearFromFile string) *MovieResult {
	if len(results) == 0 {
		return nil
	}

	// Extract year from title if present (e.g., "Movie Title (2020)")
	titleLower := strings.ToLower(title)
	titleYear := ""
	titleWithoutYear := titleLower

	yearRegex := regexp.MustCompile(`\((\d{4})\)`)
	yearMatches := yearRegex.FindStringSubmatch(titleLower)
	if len(yearMatches) > 1 {
		titleYear = yearMatches[1]
		titleWithoutYear = strings.TrimSpace(strings.Replace(titleLower, yearMatches[0], "", 1))
	}

	// Use year from file name if available and no year in title
	if titleYear == "" && yearFromFile != "" {
		titleYear = yearFromFile
	}

	type scoredResult struct {
		result *MovieResult
		score  float64
	}

	// Score each result
	var scoredResults []scoredResult
	for i, result := range results {
		resultCopy := result // Create a copy to avoid issues with loop variable capture
		score := 0.0

		// 1. Title similarity score (0-50 points)
		resultTitle := strings.ToLower(result.Title)
		if resultTitle == titleWithoutYear {
			score += 50 // Exact match is best
		} else {
			// Calculate string similarity using Levenshtein distance
			similarity := calculateStringSimilarity(titleWithoutYear, resultTitle)
			score += similarity * 40 // Up to 40 points for similar titles
		}

		// 2. Year match score (0-30 points)
		resultYear := ""
		if len(result.ReleaseDate) >= 4 {
			resultYear = result.ReleaseDate[:4]
		}

		if titleYear != "" && resultYear == titleYear {
			score += 30 // Exact year match with high weight because we extracted it from filename
		} else if titleYear != "" && resultYear != "" {
			// Partial points for close years
			yearDiff := math.Abs(float64(parseYear(resultYear) - parseYear(titleYear)))
			if yearDiff <= 1 {
				score += 20 // Only 1 year off
			} else if yearDiff <= 2 {
				score += 10 // 2 years off
			}
		}

		// 3. Popularity boost (0-20 points)
		// More popular movies get a boost, scaled by position in results
		popScore := math.Min(20, result.Popularity)
		popBoost := popScore * (1.0 - float64(i)*0.1) // Decrease boost for later results
		score += popBoost

		scoredResults = append(scoredResults, scoredResult{&resultCopy, score})
	}

	// Sort by score (highest first)
	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	// Debug logging for top matches
	for i, sr := range scoredResults {
		if i < 3 { // Log top 3 matches
			log.Printf("Match %d: '%s' (%s) - Score: %.2f",
				i+1,
				sr.result.Title,
				getYearFromDate(sr.result.ReleaseDate),
				sr.score)
		}
	}

	// If best match has a very low score, it might not be a good match at all
	if len(scoredResults) > 0 && scoredResults[0].score >= 30 {
		return scoredResults[0].result
	}

	return nil
}

// findBestTVMatch finds the best match from TMDB results for a TV show title
func findBestTVMatch(title string, results []TVResult, yearFromFile string) *TVResult {
	if len(results) == 0 {
		return nil
	}

	titleLower := strings.ToLower(title)

	// Extract year if present (e.g., "TV Show (2020)")
	titleYear := ""
	titleWithoutYear := titleLower

	yearRegex := regexp.MustCompile(`\((\d{4})\)`)
	yearMatches := yearRegex.FindStringSubmatch(titleLower)
	if len(yearMatches) > 1 {
		titleYear = yearMatches[1]
		titleWithoutYear = strings.TrimSpace(strings.Replace(titleLower, yearMatches[0], "", 1))
	}

	// Use year from file name if available and no year in title
	if titleYear == "" && yearFromFile != "" {
		titleYear = yearFromFile
	}

	type scoredResult struct {
		result *TVResult
		score  float64
	}

	// Score each result
	var scoredResults []scoredResult
	for i, result := range results {
		resultCopy := result // Create a copy to avoid issues with loop variable capture
		score := 0.0

		// 1. Title similarity score (0-50 points)
		resultTitle := strings.ToLower(result.Name)
		if resultTitle == titleWithoutYear {
			score += 50 // Exact match is best
		} else {
			// Calculate string similarity
			similarity := calculateStringSimilarity(titleWithoutYear, resultTitle)
			score += similarity * 40 // Up to 40 points for similar titles
		}

		// 2. Year match score (0-30 points)
		resultYear := ""
		if len(result.FirstAirDate) >= 4 {
			resultYear = result.FirstAirDate[:4]
		}

		if titleYear != "" && resultYear == titleYear {
			score += 30 // Exact year match with higher weight because we extracted it from filename
		} else if titleYear != "" && resultYear != "" {
			// Partial points for close years
			yearDiff := math.Abs(float64(parseYear(resultYear) - parseYear(titleYear)))
			if yearDiff <= 1 {
				score += 20 // Only 1 year off
			} else if yearDiff <= 2 {
				score += 10 // 2 years off
			}
		}

		// 3. Popularity boost (0-20 points)
		popScore := math.Min(20, result.Popularity)
		popBoost := popScore * (1.0 - float64(i)*0.1) // Decrease boost for later results
		score += popBoost

		scoredResults = append(scoredResults, scoredResult{&resultCopy, score})
	}

	// Sort by score (highest first)
	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	// Debug logging for top matches
	for i, sr := range scoredResults {
		if i < 3 { // Log top 3 matches
			log.Printf("Match %d: '%s' (%s) - Score: %.2f",
				i+1,
				sr.result.Name,
				getYearFromDate(sr.result.FirstAirDate),
				sr.score)
		}
	}

	// If best match has a very low score, it might not be a good match at all
	if len(scoredResults) > 0 && scoredResults[0].score >= 30 {
		return scoredResults[0].result
	}

	return nil
}

// Helper functions for the matching algorithms

// calculateStringSimilarity returns a normalized similarity score between 0 and 1
// Uses Levenshtein distance normalized by the length of the longer string
func calculateStringSimilarity(s1, s2 string) float64 {
	// Simple case - exact match
	if s1 == s2 {
		return 1.0
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	// Return normalized similarity (1 - normalized distance)
	return 1.0 - float64(distance)/maxLen
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	// Convert strings to rune slices to handle Unicode properly
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	// Create a matrix to store the distances
	rows := len(runes1) + 1
	cols := len(runes2) + 1
	distance := make([][]int, rows)

	// Initialize the matrix
	for i := 0; i < rows; i++ {
		distance[i] = make([]int, cols)
		distance[i][0] = i
	}

	for j := 0; j < cols; j++ {
		distance[0][j] = j
	}

	// Fill the matrix
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if runes1[i-1] == runes2[j-1] {
				cost = 0
			}

			distance[i][j] = min3(
				distance[i-1][j]+1,      // deletion
				distance[i][j-1]+1,      // insertion
				distance[i-1][j-1]+cost, // substitution
			)
		}
	}

	return distance[rows-1][cols-1]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// parseYear safely parses a year string to int
func parseYear(year string) int {
	y, err := strconv.Atoi(year)
	if err != nil {
		return 0
	}
	return y
}

// getYearFromDate extracts the year from a date string
func getYearFromDate(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

// min returns the smaller of a and b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractYearFromFileName extracts the year from a movie file name
// e.g. "Tarzan.Goes.To.India.1962.1080p.BluRay.x264-[YTS.AM].mp4" -> "1962"
func extractYearFromFileName(fileName string) string {
	// Look for a 4-digit year in the file name
	yearRegex := regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	matches := yearRegex.FindStringSubmatch(fileName)

	if len(matches) > 1 {
		// Validate that it's a reasonable movie year (1900-current year)
		year := matches[1]
		yearNum, err := strconv.Atoi(year)
		if err == nil {
			currentYear := time.Now().Year()
			if yearNum >= 1900 && yearNum <= currentYear {
				return year
			}
		}
	}

	return ""
}
