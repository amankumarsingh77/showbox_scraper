package febox

const (
	ProxyURL    = "https://simple-proxy-2.xartpvt.workers.dev?destination="
	ShowboxBase = "http://156.242.65.27/"
	FebboxBase  = "https://www.febbox.com"
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

type Config struct {
	MaxConcurrency  int
	RequestInterval int
	MaxRetries      int
	RetryDelay      int
	HTTPTimeout     int
}
