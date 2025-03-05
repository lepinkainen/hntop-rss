package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

type HackerNewsItem struct {
	Title        string
	Link         string
	CommentsLink string
	Points       string
	CommentCount string
	Author       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
			comments_link TEXT,
			points TEXT,
			comment_count TEXT,
			author TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
		title := s.Find(".titleline > a:first-child").Text()
		link, _ := s.Find(".titleline > a:first-child").Attr("href")

		// Get the comments link from the subtext row
		itemId := s.AttrOr("id", "")
		commentsLink := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", itemId)

		subtext := s.Next()
		points := subtext.Find(".score").Text()
		commentCount := subtext.Find("a").Last().Text()
		author := subtext.Find(".hnuser").Text()

		// Parse the timestamp from the age span's title attribute
		updatedAt := now
		if ageSpan := subtext.Find(".age"); ageSpan.Length() > 0 {
			if timestamp, exists := ageSpan.Attr("title"); exists {
				// Parse the Unix timestamp from the title attribute
				if unixTime, err := strconv.ParseInt(strings.Split(timestamp, " ")[1], 10, 64); err == nil {
					updatedAt = time.Unix(unixTime, 0)
				}
			}
		}

		log.WithFields(log.Fields{
			"title":        title,
			"link":         link,
			"commentsLink": commentsLink,
			"points":       points,
			"comments":     commentCount,
			"author":       author,
			"updatedAt":    updatedAt,
		}).Debug("Found item")

		items = append(items, HackerNewsItem{
			Title:        title,
			Link:         link,
			CommentsLink: commentsLink,
			Points:       points,
			CommentCount: commentCount,
			Author:       author,
			CreatedAt:    now,
			UpdatedAt:    updatedAt,
		})
	})

	return items
}

func updateStoredItems(db *sql.DB, newItems []HackerNewsItem) {
	for _, item := range newItems {
		result, err := db.Exec(`
			INSERT INTO items (title, link, comments_link, points, comment_count, author, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(link) DO UPDATE SET
				title = excluded.title,
				comments_link = excluded.comments_link,
				points = excluded.points,
				comment_count = excluded.comment_count,
				author = excluded.author,
				updated_at = excluded.updated_at`,
			item.Title, item.Link, item.CommentsLink, item.Points, item.CommentCount, item.Author, item.CreatedAt, item.UpdatedAt)
		if err != nil {
			log.WithError(err).WithField("link", item.Link).Error("Error updating item")
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.WithField("title", item.Title).Info("Added/Updated item")
		}
	}
}

func getAllItems(db *sql.DB) []HackerNewsItem {
	rows, err := db.Query("SELECT title, link, comments_link, points, comment_count, author, created_at, updated_at FROM items ORDER BY created_at DESC LIMIT 30")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var items []HackerNewsItem
	for rows.Next() {
		var item HackerNewsItem
		err := rows.Scan(&item.Title, &item.Link, &item.CommentsLink, &item.Points, &item.CommentCount, &item.Author, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			log.WithError(err).Error("Error scanning row")
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
		Link:        &feeds.Link{Href: "https://news.ycombinator.com/", Rel: "self", Type: "text/html"},
		Created:     now,
		Updated:     now,
	}

	idRegex := regexp.MustCompile(`id=(\d+)`)

	for _, item := range items {
		// Extract item ID from the comments link
		itemID := ""
		if matches := idRegex.FindStringSubmatch(item.CommentsLink); len(matches) > 1 {
			itemID = matches[1]
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title: item.Title,
			Link:  &feeds.Link{Href: item.CommentsLink},
			Id:    fmt.Sprintf("tag:news.ycombinator.com:%s", itemID),
			Author: &feeds.Author{
				Name: item.Author,
			},
			Description: fmt.Sprintf("Points: %s | Comments: %s | Article Link: <a href=\"%s\">%s</a>",
				item.Points,
				item.CommentCount,
				item.Link,
				item.Link),
			Created: item.CreatedAt,
			Updated: item.UpdatedAt,
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
	log.WithFields(log.Fields{
		"count":    len(allItems),
		"filename": filename,
	}).Info("RSS feed saved")
}

func main() {
	// Configure log
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.WarnLevel) // Only show warnings and above by default

	// Define and parse the outdir flag
	outDir := flag.String("outdir", ".", "directory where the RSS feed file will be saved")
	flag.Parse()

	updateAndSaveFeed(*outDir)
}
