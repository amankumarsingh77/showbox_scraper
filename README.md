# ShowBox Scraper

**ShowBox Scraper** is a lightweight and efficient web scraper/API designed to extract movie and TV series data from the [ShowBox](https://showbox.media) website. It organizes the scraped data into a clean and structured format for easy use.

The scraper leverages the powerful capabilities of the `goquery` and `colly` libraries to ensure reliable and accurate data extraction.

---

## ⚙️ Tech Stack

- Go
- Colly
- Goquery

---

## 🔋 Features

👉 **Scrape Movie and TV Series Data**: Extract structured information about movies and TV shows from the ShowBox website.

👉 **Efficient Web Scraping**: Utilizes `colly` for fast and robust scraping.

👉 **Data Structuring**: Formats the scraped data into a clean JSON structure for easy integration.

👉 **Scalable and Lightweight**: Designed to handle large-scale scraping tasks efficiently.

---

## 🤸 Quick Start

Follow these steps to set up the scraper locally on your machine.

### Prerequisites

Make sure you have the following installed on your machine:

- [MongoDB DB](https://www.mongodb.com/) (M0 cluster is free)
- [Go](https://go.dev/)

### Cloning the Repository

```bash
git clone https://github.com/yourusername/showbox-scraper.git
cd showbox-scraper
```

### Installation
Install the project dependencies using go mod:
```bash
go mod tidy
```

### Set Up Environment Variables

Create a .env file in the root of your project and add the following content:
```bash
MONGO_URI= # Your MongoDB connection string
DB_NAME= # Your MongoDB database name
FEBBOX_COOKIE= # Your ShowBox cookie (can get from browser)
PROXY_URL= # Your proxy URL (optional)
```

### Running the Project

Run the scraper using the following command:
```bash
go run main.go
```

### Connect With Me

Need assistance or want to collaborate? Feel free to connect:
[X](https://x.com/amankumar404) and [Gmail](mailto:amankumarsingh7702@gmail.com)