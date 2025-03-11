package febox

const (
	ProxyURL    = "https://simple-proxy.ak7702401082.workers.dev?destination="
	ShowboxBase = "https://showbox.media/"
	FebboxBase  = "https://www.febbox.com"
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

type Config struct {
	MaxConcurrency  int
	RequestInterval int
	MaxRetries      int
	RetryDelay      int
	HTTPTimeout     int
	isMovie         bool
}
