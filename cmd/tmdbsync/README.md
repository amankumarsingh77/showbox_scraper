# TMDB Sync Tool

This tool synchronizes your ShowBox database with The Movie Database (TMDB) to enrich your movie and TV show data with accurate metadata.

## Features

- Synchronize your existing movies and TV shows with TMDB metadata
- Adds rich information like:
  - Accurate titles, descriptions, and release dates
  - High-quality poster and backdrop images
  - Cast and crew information
  - Genres
  - Ratings and popularity metrics
  - Trailers and video links
  - Network information for TV shows
  - Season and episode details for TV shows
- Intelligent matching algorithm that considers:
  - Title similarity (using Levenshtein distance)
  - Release year matching
  - Popularity scores
  - Original metadata

## Prerequisites

1. A TMDB API key (get one for free at https://www.themoviedb.org/settings/api)
2. Go 1.19 or higher
3. MongoDB database with your ShowBox data

## Installation

1. Clone the repository
2. Build the tool:
   ```
   go build -o tmdbsync cmd/tmdbsync/main.go
   ```
3. Set up your environment variables in `.env` file:
   ```
   MONGO_URI=your_mongodb_connection_string
   DB_NAME=showbox
   TMDB_API_KEY=your_tmdb_api_key
   ```

## Usage

### Basic Usage

Sync all movies:
```
./tmdbsync -movie
```

Sync all TV shows:
```
./tmdbsync -tv
```

Sync both movies and TV shows:
```
./tmdbsync -all
```

### Advanced Options

Sync a specific movie by ID:
```
./tmdbsync -movie -id "12345"
```

Sync a specific TV show by ID:
```
./tmdbsync -tv -id "67890"
```

Limit the number of items to sync:
```
./tmdbsync -movie -limit 100 -skip 200
```

Enable verbose logging:
```
./tmdbsync -movie -verbose
```

Provide TMDB API key via command line:
```
./tmdbsync -movie -tmdb-key "your_api_key_here"
```

## How It Works

1. The tool loads your existing movies/TV shows from MongoDB
2. For each item, it searches TMDB using intelligent matching
3. Once the best match is found, it fetches detailed information
4. The metadata is enriched with TMDB data and saved back to your database
5. For TV shows, it also syncs season and episode information

## Matching Algorithm

The matching algorithm uses a sophisticated scoring system:

- Title similarity (up to 50 points):
  - Exact match: 50 points
  - Similar titles: Up to 40 points based on Levenshtein distance
- Year match (up to 30 points):
  - **Automatically extracts release year from file names**
  - Exact year match: 30 points
  - Within 1 year: 20 points
  - Within 2 years: 10 points
- Popularity boost (up to 20 points):
  - Based on TMDB popularity score
  - Weighted by search result position

This ensures that even with slightly different titles or release years, the right content is matched accurately. The system is particularly effective because it extracts the year from your media file names (e.g., "Movie.Title.2007.1080p.mp4"), which provides a reliable source for release years.

### File Name Year Extraction

The tool automatically analyzes file names to extract the release year:

```
For example:
- "Tarzan.Goes.To.India.1962.1080p.BluRay.x264-[YTS.AM].mp4" → Year: 1962
- "The.Matrix.1999.UHD.BluRay.2160p.TrueHD.Atmos.7.1.HEVC.REMUX.mkv" → Year: 1999
```

This extracted year is used both to improve the initial search query and to prioritize matching with the correct year.

## Scheduled Sync

For regular updates, consider adding a scheduled task:

### Linux/macOS (Cron)
```
# Run daily at 2 AM
0 2 * * * /path/to/tmdbsync -all
```

### Windows (Task Scheduler)
Create a task that runs daily with the command:
```
C:\path\to\tmdbsync.exe -all
```

## Troubleshooting

- If you see "rate limit" errors, add delays between syncs using the `-skip` and `-limit` options
- Make sure your MongoDB connection string is correct
- Ensure your TMDB API key is valid and has not exceeded its rate limit 