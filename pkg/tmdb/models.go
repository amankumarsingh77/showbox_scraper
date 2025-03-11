package tmdb

// SearchMovieResponse represents the response from the TMDB search/movie endpoint
type SearchMovieResponse struct {
	Page         int           `json:"page"`
	Results      []MovieResult `json:"results"`
	TotalResults int           `json:"total_results"`
	TotalPages   int           `json:"total_pages"`
}

// MovieResult represents a movie result from the TMDB search
type MovieResult struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Adult            bool    `json:"adult"`
	GenreIDs         []int   `json:"genre_ids"`
	OriginalLanguage string  `json:"original_language"`
}

// SearchTVResponse represents the response from the TMDB search/tv endpoint
type SearchTVResponse struct {
	Page         int        `json:"page"`
	Results      []TVResult `json:"results"`
	TotalResults int        `json:"total_results"`
	TotalPages   int        `json:"total_pages"`
}

// TVResult represents a TV show result from the TMDB search
type TVResult struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	OriginalName     string  `json:"original_name"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	Overview         string  `json:"overview"`
	FirstAirDate     string  `json:"first_air_date"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	GenreIDs         []int   `json:"genre_ids"`
	OriginalLanguage string  `json:"original_language"`
}

// MovieDetails represents the detailed information about a movie from TMDB
type MovieDetails struct {
	ID                  int            `json:"id"`
	IMDbID              string         `json:"imdb_id"`
	Title               string         `json:"title"`
	OriginalTitle       string         `json:"original_title"`
	Overview            string         `json:"overview"`
	PosterPath          string         `json:"poster_path"`
	BackdropPath        string         `json:"backdrop_path"`
	ReleaseDate         string         `json:"release_date"`
	Runtime             int            `json:"runtime"`
	Budget              int            `json:"budget"`
	Revenue             int            `json:"revenue"`
	Status              string         `json:"status"`
	Genres              []Genre        `json:"genres"`
	ProductionCompanies []Company      `json:"production_companies"`
	ProductionCountries []Country      `json:"production_countries"`
	VoteAverage         float64        `json:"vote_average"`
	VoteCount           int            `json:"vote_count"`
	Popularity          float64        `json:"popularity"`
	SpokenLanguages     []Language     `json:"spoken_languages"`
	Credits             Credits        `json:"credits,omitempty"`
	Videos              VideosResponse `json:"videos,omitempty"`
	Images              ImagesResponse `json:"images,omitempty"`
}

// TVDetails represents the detailed information about a TV show from TMDB
type TVDetails struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	OriginalName     string         `json:"original_name"`
	Overview         string         `json:"overview"`
	PosterPath       string         `json:"poster_path"`
	BackdropPath     string         `json:"backdrop_path"`
	FirstAirDate     string         `json:"first_air_date"`
	LastAirDate      string         `json:"last_air_date"`
	Status           string         `json:"status"`
	Genres           []Genre        `json:"genres"`
	Networks         []Network      `json:"networks"`
	NumberOfSeasons  int            `json:"number_of_seasons"`
	NumberOfEpisodes int            `json:"number_of_episodes"`
	EpisodeRunTime   []int          `json:"episode_run_time"`
	VoteAverage      float64        `json:"vote_average"`
	VoteCount        int            `json:"vote_count"`
	Popularity       float64        `json:"popularity"`
	Seasons          []Season       `json:"seasons"`
	Credits          Credits        `json:"credits,omitempty"`
	Videos           VideosResponse `json:"videos,omitempty"`
	Images           ImagesResponse `json:"images,omitempty"`
}

// SeasonDetails represents the detailed information about a TV season from TMDB
type SeasonDetails struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	SeasonNumber int       `json:"season_number"`
	AirDate      string    `json:"air_date"`
	Overview     string    `json:"overview"`
	PosterPath   string    `json:"poster_path"`
	Episodes     []Episode `json:"episodes"`
}

// EpisodeDetails represents the detailed information about a TV episode from TMDB
type EpisodeDetails struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	AirDate       string  `json:"air_date"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	StillPath     string  `json:"still_path"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
}

// Supporting types

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Company struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Country struct {
	ISO31661 string `json:"iso_3166_1"`
	Name     string `json:"name"`
}

type Language struct {
	ISO6391 string `json:"iso_639_1"`
	Name    string `json:"name"`
}

type Network struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Season struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	AirDate      string `json:"air_date"`
	PosterPath   string `json:"poster_path"`
}

type Episode struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	AirDate       string  `json:"air_date"`
	StillPath     string  `json:"still_path"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
}

type Credits struct {
	Cast []Cast `json:"cast"`
	Crew []Crew `json:"crew"`
}

type Cast struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

type Crew struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Department  string `json:"department"`
	Job         string `json:"job"`
	ProfilePath string `json:"profile_path"`
}

type VideosResponse struct {
	Results []Video `json:"results"`
}

type Video struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Site     string `json:"site"`
	Size     int    `json:"size"`
	Type     string `json:"type"`
	ISO6391  string `json:"iso_639_1"`
	ISO31661 string `json:"iso_3166_1"`
	Official bool   `json:"official"`
}

type ImagesResponse struct {
	Backdrops []Image `json:"backdrops"`
	Posters   []Image `json:"posters"`
}

type Image struct {
	AspectRatio float64 `json:"aspect_ratio"`
	FilePath    string  `json:"file_path"`
	Height      int     `json:"height"`
	Width       int     `json:"width"`
	ISO6391     string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int     `json:"vote_count"`
}
