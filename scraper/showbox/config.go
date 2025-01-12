package showbox

type Config struct {
	BaseURL     string
	ProxyURL    string
	StartPage   int
	EndPage     int
	UserAgent   string `default:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"`
	Timeout     int    `default:"120"`
	Parallelism int    `default:"5"`
	RandomDelay int    `default:"3"`
}

func DefaultConfig() *Config {
	return &Config{
		BaseURL:     "http://156.242.65.27",
		ProxyURL:    "https://simple-proxy.xartpvt.workers.dev?destination=",
		StartPage:   1,
		EndPage:     10,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Timeout:     120,
		Parallelism: 5,
		RandomDelay: 3,
	}
}
