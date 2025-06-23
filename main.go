package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

type HackerNewsItem struct {
	ItemID       string
	Title        string
	Link         string
	CommentsLink string
	Points       string
	CommentCount string
	Author       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AlgoliaResponse struct {
	Hits []AlgoliaHit `json:"hits"`
}

type AlgoliaHit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Author      string `json:"author"`
	Points      int    `json:"points"`
	NumComments int    `json:"num_comments"`
	CreatedAt   string `json:"created_at"`
}

func initDB() *sql.DB {
	// Get the directory of the executable
	exePath, err := os.Executable()
	if err != nil {
		slog.Error("Error getting executable path", "error", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)
	dbPath := filepath.Join(exeDir, "hackernews.db")
	slog.Debug("Initializing database", "path", dbPath)

	// Open database in the executable's directory
	db, err := sql.Open("sqlite", dbPath) // Use "sqlite" driver name
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	// Create table if it doesn't exist
	createTable := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,      -- Internal row ID (optional, but common)
		item_hn_id TEXT NOT NULL UNIQUE,        -- Hacker News Item ID, for deduplication
		title TEXT NOT NULL,
		link TEXT NOT NULL,                     -- The actual article URL
		comments_link TEXT,
		points TEXT,
		comment_count TEXT,
		author TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db.Exec(createTable)
	if err != nil {
		slog.Error("Failed to create table", "error", err)
		os.Exit(1)
	}
	slog.Debug("Database initialized successfully")

	return db
}

func fetchHackerNewsItems() []HackerNewsItem {
	slog.Debug("Fetching Hacker News items from Algolia API")
	res, err := http.Get("https://hn.algolia.com/api/v1/search_by_date?tags=front_page&hitsPerPage=100")
	if err != nil {
		slog.Error("Failed to fetch Hacker News items", "error", err)
		os.Exit(1)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		slog.Error("HTTP status code error", "code", res.StatusCode, "status", res.Status)
		os.Exit(1)
	}

	var algoliaResp AlgoliaResponse
	if err := json.NewDecoder(res.Body).Decode(&algoliaResp); err != nil {
		slog.Error("Failed to decode JSON response", "error", err)
		os.Exit(1)
	}

	var items []HackerNewsItem
	now := time.Now()
	slog.Debug("Processing Algolia response", "hitCount", len(algoliaResp.Hits))

	for _, hit := range algoliaResp.Hits {
		// Parse the ISO 8601 timestamp
		createdAt, err := time.Parse(time.RFC3339, hit.CreatedAt)
		if err != nil {
			slog.Warn("Failed to parse timestamp, using current time", "error", err, "timestamp", hit.CreatedAt)
			createdAt = now
		}

		// Generate comments link from object ID
		commentsLink := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)

		// Convert numeric fields to strings to match existing schema
		points := strconv.Itoa(hit.Points)
		commentCount := strconv.Itoa(hit.NumComments)

		slog.Debug("Found item",
			"title", hit.Title,
			"link", hit.URL,
			"commentsLink", commentsLink,
			"points", points,
			"comments", commentCount,
			"author", hit.Author,
			"createdAt", createdAt)

		items = append(items, HackerNewsItem{
			ItemID:       hit.ObjectID,
			Title:        hit.Title,
			Link:         hit.URL,
			CommentsLink: commentsLink,
			Points:       points,
			CommentCount: commentCount,
			Author:       hit.Author,
			CreatedAt:    createdAt,
			UpdatedAt:    now,
		})
	}

	slog.Debug("Finished processing items", "totalItems", len(items))
	return items
}

func updateStoredItems(db *sql.DB, newItems []HackerNewsItem) map[string]bool {
	slog.Debug("Updating stored items", "itemCount", len(newItems))
	updatedItems := make(map[string]bool)
	
	for _, item := range newItems {
		// The 'item.CreatedAt' should be the original submission time of the HN post.
		// The 'item.UpdatedAt' should be when it was last seen/modified by your scraper.
		result, err := db.Exec(`
			INSERT INTO items (item_hn_id, title, link, comments_link, points, comment_count, author, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(item_hn_id) DO UPDATE SET
				title = excluded.title,
				link = excluded.link, 
				comments_link = excluded.comments_link,
				points = excluded.points,
				comment_count = excluded.comment_count,
				author = excluded.author,
				updated_at = excluded.updated_at`, // Note: created_at is not updated on conflict
			item.ItemID, item.Title, item.Link, item.CommentsLink, item.Points, item.CommentCount, item.Author, item.CreatedAt, item.UpdatedAt)

		if err != nil {
			slog.Error("Error updating item", "error", err, "hn_id", item.ItemID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			slog.Info("Processed item (added/updated in DB)", "title", item.Title, "hn_id", item.ItemID)
			updatedItems[item.ItemID] = true
		}
	}
	
	return updatedItems
}

func getAllItems(db *sql.DB, limit int) []HackerNewsItem {
	slog.Debug("Querying database for items", "limit", limit)
	rows, err := db.Query("SELECT item_hn_id, title, link, comments_link, points, comment_count, author, created_at, updated_at FROM items WHERE points > 50 ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		slog.Error("Failed to query database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = rows.Close() }()

	var items []HackerNewsItem
	for rows.Next() {
		var item HackerNewsItem
		err := rows.Scan(&item.ItemID, &item.Title, &item.Link, &item.CommentsLink, &item.Points, &item.CommentCount, &item.Author, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			slog.Error("Error scanning row", "error", err)
			continue
		}
		items = append(items, item)
	}

	slog.Debug("Retrieved items from database", "count", len(items))
	return items
}

func updateItemStats(db *sql.DB, items []HackerNewsItem, recentlyUpdated map[string]bool) {
	slog.Debug("Updating item stats", "itemCount", len(items))
	skippedCount := 0
	
	for _, item := range items {
		// Skip items with empty ItemID
		if item.ItemID == "" {
			slog.Warn("Skipping item with empty ItemID", "title", item.Title)
			continue
		}

		// Skip items that were just updated in updateStoredItems
		if recentlyUpdated[item.ItemID] {
			slog.Debug("Skipping recently updated item", "hn_id", item.ItemID)
			skippedCount++
			continue
		}

		// Fetch current stats from Algolia API
		url := fmt.Sprintf("https://hn.algolia.com/api/v1/items/%s", item.ItemID)
		res, err := http.Get(url)
		if err != nil {
			slog.Warn("Failed to fetch item stats from Algolia", "error", err, "hn_id", item.ItemID)
			continue
		}

		if res.StatusCode != 200 {
			slog.Warn("HTTP error fetching item stats", "code", res.StatusCode, "hn_id", item.ItemID)
			_ = res.Body.Close()
			continue
		}

		var algoliaItem AlgoliaHit
		if err := json.NewDecoder(res.Body).Decode(&algoliaItem); err != nil {
			slog.Warn("Failed to decode Algolia response", "error", err, "hn_id", item.ItemID)
			_ = res.Body.Close()
			continue
		}
		_ = res.Body.Close()

		// Update database with current stats
		points := strconv.Itoa(algoliaItem.Points)
		commentCount := strconv.Itoa(algoliaItem.NumComments)
		
		_, err = db.Exec(`
			UPDATE items SET 
				points = ?, 
				comment_count = ?, 
				updated_at = ?
			WHERE item_hn_id = ?`,
			points, commentCount, time.Now(), item.ItemID)
		
		if err != nil {
			slog.Warn("Failed to update item stats in database", "error", err, "hn_id", item.ItemID)
			continue
		}

		slog.Debug("Updated item stats", "hn_id", item.ItemID, "points", points, "comments", commentCount)
	}
	
	if skippedCount > 0 {
		slog.Debug("Skipped recently updated items", "count", skippedCount)
	}
}

func generateRSSFeed(items []HackerNewsItem) string {
	slog.Debug("Generating RSS feed", "itemCount", len(items))
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Hacker News RSS Feed",
		Description: "Latest stories from Hacker News",
		Link:        &feeds.Link{Href: "https://news.ycombinator.com/", Rel: "self", Type: "text/html"},
		Id:          "tag:news.ycombinator.com,2024:feed",
		Created:     now,
		Updated:     now,
	}

	// idRegex := regexp.MustCompile(`id=(\d+)`)
	domainRegex := regexp.MustCompile(`^https?://([^/]+)`)
	commentRegex := regexp.MustCompile(`(\d+)`)

	for _, item := range items {
		// Extract domain from the article link
		domain := ""
		if matches := domainRegex.FindStringSubmatch(item.Link); len(matches) > 1 {
			domain = matches[1]
		}

		// Format points (remove "points" suffix if present)
		points := strings.TrimSuffix(item.Points, " points")
		points = strings.TrimSuffix(points, " point")

		// Format comment count - just extract the digits portion
		comments := item.CommentCount
		if matches := commentRegex.FindStringSubmatch(item.CommentCount); len(matches) > 1 {
			comments = matches[1]
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title: item.Title,
			Link:  &feeds.Link{Href: item.CommentsLink, Rel: "alternate", Type: "text/html"},
			Id:    item.CommentsLink,
			Author: &feeds.Author{
				Name: item.Author,
			},
			Description: fmt.Sprintf(`<p>
				<strong>%s points</strong> | 
				<strong>%s comments</strong><br/>
				Source: %s<br/>
				<a href="%s">View Comments</a> | 
				<a href="%s">Read Article</a>
				</p>`,
				points,
				comments,
				domain,
				item.CommentsLink,
				item.Link),
			Created: item.CreatedAt,
		})
	}

	rss, err := feed.ToAtom()
	if err != nil {
		slog.Error("Failed to generate RSS feed", "error", err)
		os.Exit(1)
	}

	slog.Debug("RSS feed generated successfully", "feedSize", len(rss))
	return rss
}

func updateAndSaveFeed(outDir string) {
	db := initDB()
	defer func() { _ = db.Close() }()

	// Fetch current front page items
	newItems := fetchHackerNewsItems()

	// Update database with new items and get list of updated item IDs
	recentlyUpdated := updateStoredItems(db, newItems)

	// Get all items from database
	allItems := getAllItems(db, 30)

	// Update item stats with current data from Algolia, skipping recently updated items
	updateItemStats(db, allItems, recentlyUpdated)

	// Re-fetch items to get updated stats for RSS generation
	allItems = getAllItems(db, 30)

	// Ensure output directory exists
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		slog.Error("Error creating output directory", "error", err)
		os.Exit(1)
	}

	// Generate and save the feed
	filename := filepath.Join(outDir, "hntop30.xml")
	rss := generateRSSFeed(allItems)
	err = os.WriteFile(filename, []byte(rss), 0644)
	if err != nil {
		slog.Error("Error writing RSS feed to file", "error", err)
		os.Exit(1)
	}
	slog.Info("RSS feed saved", "count", len(allItems), "filename", filename)
}

func main() {
	// Define and parse command line flags
	outDir := flag.String("outdir", ".", "directory where the RSS feed file will be saved")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Configure log level based on debug flag
	logLevel := slog.LevelWarn
	if *debug {
		logLevel = slog.LevelDebug
	}
	
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Debug("Starting application", "outDir", *outDir, "debugMode", *debug)
	updateAndSaveFeed(*outDir)
}
