package showbox

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Storage struct {
	TempDir   string
	FinalFile string
}

func NewStorage() *Storage {
	return &Storage{
		TempDir:   "temp",
		FinalFile: "movies_final.json",
	}
}

func (s *Storage) SaveProgress(movies []Movie) error {
	if err := os.MkdirAll(s.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	tempFile := filepath.Join(s.TempDir, fmt.Sprintf("movies_%s.json", timestamp))

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(movies); err != nil {
		return fmt.Errorf("failed to save progress: %v", err)
	}

	log.Printf("Progress saved: %d movies\n", len(movies))
	return nil
}

func (s *Storage) MergeFiles() error {
	allMovies := make(map[string]Movie)

	if _, err := os.Stat(s.FinalFile); err == nil {
		data, err := os.ReadFile(s.FinalFile)
		if err == nil {
			var existingMovies []Movie
			if err := json.Unmarshal(data, &existingMovies); err == nil {
				for _, movie := range existingMovies {
					allMovies[movie.ID] = movie
				}
			}
		}
	}

	files, err := filepath.Glob(filepath.Join(s.TempDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list temp files: %v", err)
	}

	for _, file := range files {
		var movies []Movie
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", file, err)
			continue
		}

		if err := json.Unmarshal(data, &movies); err != nil {
			log.Printf("Error parsing file %s: %v\n", file, err)
			continue
		}

		for _, movie := range movies {
			allMovies[movie.ID] = movie
		}

		if err := os.Remove(file); err != nil {
			log.Printf("Failed to remove file %s: %v\n", file, err)
		}
	}

	final := make([]Movie, 0, len(allMovies))
	for _, movie := range allMovies {
		final = append(final, movie)
	}

	finalFile, err := os.Create(s.FinalFile)
	if err != nil {
		return fmt.Errorf("failed to create final file: %v", err)
	}
	defer finalFile.Close()

	encoder := json.NewEncoder(finalFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(final); err != nil {
		return fmt.Errorf("failed to save final file: %v", err)
	}

	log.Printf("Final merge complete: %d unique movies saved\n", len(final))
	return nil
}
