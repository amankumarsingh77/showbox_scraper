package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client represents a TMDB API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new TMDB API client
func NewClient() (*Client, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY environment variable is not set")
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// SearchMovie searches for a movie by title
func (c *Client) SearchMovie(title string) (*SearchMovieResponse, error) {
	endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", c.baseURL, c.apiKey, url.QueryEscape(title))

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to search movie: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result SearchMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SearchTV searches for a TV show by title
func (c *Client) SearchTV(title string) (*SearchTVResponse, error) {
	endpoint := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", c.baseURL, c.apiKey, url.QueryEscape(title))

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to search TV show: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result SearchTVResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetMovieDetails gets detailed information about a movie by its TMDB ID
func (c *Client) GetMovieDetails(tmdbID int) (*MovieDetails, error) {
	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s&append_to_response=credits,images,videos",
		c.baseURL, tmdbID, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTVDetails gets detailed information about a TV show by its TMDB ID
func (c *Client) GetTVDetails(tmdbID int) (*TVDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d?api_key=%s&append_to_response=credits,images,videos",
		c.baseURL, tmdbID, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get TV details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTVSeasonDetails gets detailed information about a TV season
func (c *Client) GetTVSeasonDetails(tmdbID, seasonNumber int) (*SeasonDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d/season/%d?api_key=%s",
		c.baseURL, tmdbID, seasonNumber, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result SeasonDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTVEpisodeDetails gets detailed information about a TV episode
func (c *Client) GetTVEpisodeDetails(tmdbID, seasonNumber, episodeNumber int) (*EpisodeDetails, error) {
	endpoint := fmt.Sprintf("%s/tv/%d/season/%d/episode/%d?api_key=%s",
		c.baseURL, tmdbID, seasonNumber, episodeNumber, c.apiKey)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var result EpisodeDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
