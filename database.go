package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// initDB initializes and returns a SQLite database connection
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

	// Create items table if it doesn't exist
	createItemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,      -- Internal row ID (optional, but common)
		item_hn_id TEXT NOT NULL UNIQUE,        -- Hacker News Item ID, for deduplication
		title TEXT NOT NULL,
		link TEXT NOT NULL,                     -- The actual article URL
		comments_link TEXT,
		points INTEGER DEFAULT 0,
		comment_count INTEGER DEFAULT 0,
		author TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db.Exec(createItemsTable)
	if err != nil {
		slog.Error("Failed to create items table", "error", err)
		os.Exit(1)
	}

	// Create OpenGraph cache table if it doesn't exist
	createOGCacheTable := `
	CREATE TABLE IF NOT EXISTS opengraph_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		title TEXT,
		description TEXT,
		image TEXT,
		site_name TEXT,
		fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP,
		fetch_success BOOLEAN DEFAULT TRUE
	)`
	_, err = db.Exec(createOGCacheTable)
	if err != nil {
		slog.Error("Failed to create opengraph_cache table", "error", err)
		os.Exit(1)
	}

	// Create indexes for opengraph_cache table
	createOGIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_opengraph_url ON opengraph_cache(url)",
		"CREATE INDEX IF NOT EXISTS idx_opengraph_expires ON opengraph_cache(expires_at)",
	}
	for _, indexSQL := range createOGIndexes {
		_, err = db.Exec(indexSQL)
		if err != nil {
			slog.Error("Failed to create opengraph_cache index", "error", err)
			os.Exit(1)
		}
	}
	slog.Debug("Database initialized successfully")

	return db
}

// updateStoredItems updates the database with new items, returns map of updated item IDs
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

// getAllItems retrieves items from database with minimum points threshold
func getAllItems(db *sql.DB, limit int, minPoints int) []HackerNewsItem {
	slog.Debug("Querying database for items", "limit", limit, "minPoints", minPoints)
	rows, err := db.Query("SELECT item_hn_id, title, link, comments_link, points, comment_count, author, created_at, updated_at FROM items WHERE points > ? ORDER BY created_at DESC LIMIT ?", minPoints, limit)
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

// getOpenGraphData retrieves cached OpenGraph data for a URL
func getOpenGraphData(db *sql.DB, url string) (*OpenGraphCache, error) {
	slog.Debug("Getting cached OpenGraph data", "url", url)

	query := `
		SELECT id, url, title, description, image, site_name, fetched_at, expires_at, fetch_success 
		FROM opengraph_cache 
		WHERE url = ? AND expires_at > ?`

	var cache OpenGraphCache
	err := db.QueryRow(query, url, time.Now()).Scan(
		&cache.ID,
		&cache.URL,
		&cache.Title,
		&cache.Description,
		&cache.Image,
		&cache.SiteName,
		&cache.FetchedAt,
		&cache.ExpiresAt,
		&cache.FetchSuccess,
	)

	if err == sql.ErrNoRows {
		slog.Debug("No cached OpenGraph data found", "url", url)
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query OpenGraph cache: %w", err)
	}

	slog.Debug("Found cached OpenGraph data", "url", url, "title", cache.Title)
	return &cache, nil
}

// cacheOpenGraphData stores OpenGraph data in the cache
func cacheOpenGraphData(db *sql.DB, ogData *OpenGraphData, fetchSuccess bool) error {
	slog.Debug("Caching OpenGraph data", "url", ogData.URL, "success", fetchSuccess)

	// Calculate expiry time: 7 days for successful fetches, 1 day for failures
	var expiresAt time.Time
	if fetchSuccess {
		expiresAt = time.Now().Add(7 * 24 * time.Hour)
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	query := `
		INSERT INTO opengraph_cache (url, title, description, image, site_name, fetched_at, expires_at, fetch_success)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			image = excluded.image,
			site_name = excluded.site_name,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			fetch_success = excluded.fetch_success`

	_, err := db.Exec(query,
		ogData.URL,
		ogData.Title,
		ogData.Description,
		ogData.Image,
		ogData.SiteName,
		time.Now(),
		expiresAt,
		fetchSuccess,
	)

	if err != nil {
		return fmt.Errorf("failed to cache OpenGraph data: %w", err)
	}

	slog.Debug("Successfully cached OpenGraph data", "url", ogData.URL)
	return nil
}

// cleanupExpiredOpenGraphCache removes expired OpenGraph cache entries
func cleanupExpiredOpenGraphCache(db *sql.DB) error {
	slog.Debug("Cleaning up expired OpenGraph cache entries")

	result, err := db.Exec("DELETE FROM opengraph_cache WHERE expires_at < ?", time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired cache: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		slog.Debug("Cleaned up expired OpenGraph cache entries", "count", rowsAffected)
	}

	return nil
}
