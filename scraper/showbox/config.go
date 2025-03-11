package showbox

type Config struct {
	DirectBaseURL string
	BaseURL       string
	ProxyURL      string
	StartPage     int
	EndPage       int
	UserAgent     string `default:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"`
	Timeout       int    `default:"120"`
	Parallelism   int    `default:"5"`
	RandomDelay   int    `default:"3"`
	StreamProxy   string `default:"https://simple-proxy.ak7702401082.workers.dev?destination="`
	isMovie       bool
}

func DefaultConfig() *Config {
	return &Config{
		DirectBaseURL: "http://156.242.65.27",
		BaseURL:       "https://www.showbox.media",
		ProxyURL:      "https://simple-proxy.ak7702401082.workers.dev?destination=",
		StartPage:     1,
		EndPage:       258,
		UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Timeout:       120,
		Parallelism:   2,
		RandomDelay:   3,
		StreamProxy:   "https://simple-proxy.ak7702401082.workers.dev?destination=",
		isMovie:       false,
	}
}
