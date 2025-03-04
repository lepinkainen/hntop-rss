package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

type HackerNewsItem struct {
	Title     string
	Link      string
	Points    string
	CreatedAt time.Time
}

func initDB() *sql.DB {
	// Get the directory of the executable
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	dbPath := filepath.Join(exeDir, "hackernews.db")

	// Open database in the executable's directory
	db, err := sql.Open("sqlite", dbPath) // Use "sqlite" driver name
	if err != nil {
		log.Fatal(err)
	}

	// Create table if it doesn't exist
	createTable := `
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			link TEXT NOT NULL UNIQUE,
			points TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func fetchHackerNewsItems() []HackerNewsItem {
	res, err := http.Get("https://news.ycombinator.com")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var items []HackerNewsItem
	now := time.Now()

	doc.Find(".athing").Each(func(i int, s *goquery.Selection) {
		title := s.Find(".titleline a").Text()
		link, _ := s.Find(".titleline a").Attr("href")
		points := s.Next().Find(".score").Text()

		items = append(items, HackerNewsItem{
			Title:     title,
			Link:      link,
			Points:    points,
			CreatedAt: now,
		})
	})

	return items
}

func updateStoredItems(db *sql.DB, newItems []HackerNewsItem) {
	for _, item := range newItems {
		result, err := db.Exec(`
			INSERT INTO items (title, link, points, created_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(link) DO UPDATE SET
				title = excluded.title,
				points = excluded.points`,
			item.Title, item.Link, item.Points, item.CreatedAt)
		if err != nil {
			log.Printf("Error updating item %s: %v", item.Link, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			fmt.Printf("Added/Updated item: %s\n", item.Title)
		}
	}
}

func getAllItems(db *sql.DB) []HackerNewsItem {
	rows, err := db.Query("SELECT title, link, points, created_at FROM items ORDER BY created_at DESC LIMIT 30")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var items []HackerNewsItem
	for rows.Next() {
		var item HackerNewsItem
		err := rows.Scan(&item.Title, &item.Link, &item.Points, &item.CreatedAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		items = append(items, item)
	}

	return items
}

func generateRSSFeed(items []HackerNewsItem) string {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Hacker News RSS Feed",
		Description: "Latest stories from Hacker News",
		Link:        &feeds.Link{Href: "https://news.ycombinator.com"},
		Created:     now,
		Updated:     now,
	}

	for _, item := range items {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Link},
			Description: fmt.Sprintf("Points: %s", item.Points),
			Created:     item.CreatedAt,
		})
	}

	rss, err := feed.ToAtom()
	if err != nil {
		log.Fatal(err)
	}

	return rss
}

func updateAndSaveFeed(outDir string) {
	db := initDB()
	defer db.Close()

	// Fetch current front page items
	newItems := fetchHackerNewsItems()

	// Update database with new items
	updateStoredItems(db, newItems)

	// Get all items from database
	allItems := getAllItems(db)

	// Ensure output directory exists
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	// Generate and save the feed
	filename := filepath.Join(outDir, "hntop30.xml")
	rss := generateRSSFeed(allItems)
	err = os.WriteFile(filename, []byte(rss), 0644)
	if err != nil {
		log.Fatalf("Error writing RSS feed to file: %v", err)
	}
	fmt.Printf("RSS feed with %d items saved to %s\n", len(allItems), filename)
}

func main() {
	// Define and parse the outdir flag
	outDir := flag.String("outdir", ".", "directory where the RSS feed file will be saved")
	flag.Parse()

	updateAndSaveFeed(*outDir)
}
