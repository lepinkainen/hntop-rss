package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
)

var Version string

// updateAndSaveFeed orchestrates the entire process of fetching, updating, and generating the RSS feed
func updateAndSaveFeed(outDir string, minPoints int, categoryMapper *CategoryMapper) {
	db := initDB()
	defer func() { _ = db.Close() }()

	// Clean up expired OpenGraph cache entries
	if err := cleanupExpiredOpenGraphCache(db); err != nil {
		slog.Warn("Failed to cleanup expired OpenGraph cache", "error", err)
	}

	// Fetch current front page items
	newItems := fetchHackerNewsItems()

	// Update database with new items and get list of updated item IDs
	recentlyUpdated := updateStoredItems(db, newItems)

	// Get all items from database
	allItems := getAllItems(db, 30, minPoints)

	// Update item stats with current data from Algolia, skipping recently updated items
	updateItemStats(db, allItems, recentlyUpdated)

	// Re-fetch items to get updated stats for RSS generation
	allItems = getAllItems(db, 30, minPoints)

	// Ensure output directory exists
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		slog.Error("Error creating output directory", "error", err)
		os.Exit(1)
	}

	// Generate and save the feed
	filename := filepath.Join(outDir, "hntop30.xml")
	rss := generateRSSFeed(db, allItems, minPoints, categoryMapper)
	err = os.WriteFile(filename, []byte(rss), 0644)
	if err != nil {
		slog.Error("Error writing RSS feed to file", "error", err)
		os.Exit(1)
	}
	slog.Info("RSS feed saved", "count", len(allItems), "filename", filename)
}

// main is the application entry point that parses flags and starts the RSS generation
func main() {
	// Define and parse command line flags
	outDir := flag.String("outdir", ".", "directory where the RSS feed file will be saved")
	debug := flag.Bool("debug", false, "enable debug logging")
	minPoints := flag.Int("min-points", 50, "minimum points threshold for items to include in RSS feed")
	configPath := flag.String("config", "", "path to local configuration file (optional)")
	configURL := flag.String("config-url", "", "URL to remote configuration file (defaults to GitHub)")
	flag.Parse()

	// Configure log level based on debug flag
	logLevel := slog.LevelWarn
	if *debug {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// Load configuration
	categoryMapper := LoadConfig(*configPath, *configURL)

	slog.Debug("Starting application", "outDir", *outDir, "debugMode", *debug, "minPoints", *minPoints)
	updateAndSaveFeed(*outDir, *minPoints, categoryMapper)
}
