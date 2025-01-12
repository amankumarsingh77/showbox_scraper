package showbox

import "log"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := DefaultConfig()
	storage := NewStorage()

	scraper, err := NewScraper(config, storage)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}

	if err := scraper.Run(); err != nil {
		log.Fatalf("Scraper error: %v", err)
	}
}
