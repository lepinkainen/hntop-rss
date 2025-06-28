package main

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB() *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}

	// Create items table schema
	createItemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_hn_id TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		link TEXT NOT NULL,
		comments_link TEXT,
		points INTEGER DEFAULT 0,
		comment_count INTEGER DEFAULT 0,
		author TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = db.Exec(createItemsTable)
	if err != nil {
		panic(err)
	}

	// Create OpenGraph cache table schema
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
		panic(err)
	}

	// Create indexes for opengraph_cache table
	createOGIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_opengraph_url ON opengraph_cache(url)",
		"CREATE INDEX IF NOT EXISTS idx_opengraph_expires ON opengraph_cache(expires_at)",
	}
	for _, indexSQL := range createOGIndexes {
		_, err = db.Exec(indexSQL)
		if err != nil {
			panic(err)
		}
	}

	return db
}

func TestInitDB(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Test that the table was created
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table 'items' was not created: %v", err)
	}
	if tableName != "items" {
		t.Errorf("Expected table name 'items', got '%s'", tableName)
	}
}

func TestUpdateStoredItems_NewItem(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	items := []HackerNewsItem{
		{
			ItemID:       "12345",
			Title:        "Test Article",
			Link:         "https://example.com/test",
			CommentsLink: "https://news.ycombinator.com/item?id=12345",
			Points:       75,
			CommentCount: 25,
			Author:       "testuser",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
	}

	updateStoredItems(db, items)

	// Check that the item was inserted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if err != nil {
		t.Fatalf("Error counting items: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item, got %d", count)
	}

	// Check the item details
	var title, link, author string
	err = db.QueryRow("SELECT title, link, author FROM items WHERE item_hn_id = ?", "12345").Scan(&title, &link, &author)
	if err != nil {
		t.Fatalf("Error retrieving item: %v", err)
	}
	if title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", title)
	}
	if link != "https://example.com/test" {
		t.Errorf("Expected link 'https://example.com/test', got '%s'", link)
	}
	if author != "testuser" {
		t.Errorf("Expected author 'testuser', got '%s'", author)
	}
}

func TestUpdateStoredItems_UpdateExisting(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert initial item
	initialItem := []HackerNewsItem{
		{
			ItemID:       "12345",
			Title:        "Original Title",
			Link:         "https://example.com/original",
			CommentsLink: "https://news.ycombinator.com/item?id=12345",
			Points:       50,
			CommentCount: 10,
			Author:       "original_user",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	updateStoredItems(db, initialItem)

	// Update with new information
	updatedItem := []HackerNewsItem{
		{
			ItemID:       "12345", // Same ID
			Title:        "Updated Title",
			Link:         "https://example.com/updated",
			CommentsLink: "https://news.ycombinator.com/item?id=12345",
			Points:       75,
			CommentCount: 25,
			Author:       "updated_user",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), // Original creation time
			UpdatedAt:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC), // New update time
		},
	}
	updateStoredItems(db, updatedItem)

	// Check that there's still only one item
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if err != nil {
		t.Fatalf("Error counting items: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item after update, got %d", count)
	}

	// Check that the item was updated
	var title string
	var points int
	err = db.QueryRow("SELECT title, points FROM items WHERE item_hn_id = ?", "12345").Scan(&title, &points)
	if err != nil {
		t.Fatalf("Error retrieving updated item: %v", err)
	}
	if title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", title)
	}
	if points != 75 {
		t.Errorf("Expected updated points 75, got %d", points)
	}
}

func TestGetAllItems_FilterByPoints(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert items with different point counts
	items := []HackerNewsItem{
		{
			ItemID:       "1",
			Title:        "High Points Article",
			Link:         "https://example.com/high",
			CommentsLink: "https://news.ycombinator.com/item?id=1",
			Points:       75, // Above 50
			CommentCount: 25,
			Author:       "user1",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
		{
			ItemID:       "2",
			Title:        "Low Points Article",
			Link:         "https://example.com/low",
			CommentsLink: "https://news.ycombinator.com/item?id=2",
			Points:       25, // Below 50
			CommentCount: 5,
			Author:       "user2",
			CreatedAt:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
		{
			ItemID:       "3",
			Title:        "Medium Points Article",
			Link:         "https://example.com/medium",
			CommentsLink: "https://news.ycombinator.com/item?id=3",
			Points:       60, // Above 50
			CommentCount: 15,
			Author:       "user3",
			CreatedAt:    time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
	}

	updateStoredItems(db, items)

	// Get items (should only return those with >50 points)
	retrievedItems := getAllItems(db, 30, 50)

	if len(retrievedItems) != 2 {
		t.Errorf("Expected 2 items with >50 points, got %d", len(retrievedItems))
	}

	// Check that only high-point items are returned
	foundTitles := make(map[string]bool)
	for _, item := range retrievedItems {
		foundTitles[item.Title] = true
	}

	if !foundTitles["High Points Article"] {
		t.Error("Expected 'High Points Article' to be returned")
	}
	if !foundTitles["Medium Points Article"] {
		t.Error("Expected 'Medium Points Article' to be returned")
	}
	if foundTitles["Low Points Article"] {
		t.Error("'Low Points Article' should not be returned (points < 50)")
	}
}

func TestGetAllItems_OrderByCreatedAt(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert items with different creation times
	items := []HackerNewsItem{
		{
			ItemID:       "1",
			Title:        "Older Article",
			Link:         "https://example.com/old",
			CommentsLink: "https://news.ycombinator.com/item?id=1",
			Points:       75,
			CommentCount: 25,
			Author:       "user1",
			CreatedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), // Earlier
			UpdatedAt:    time.Now(),
		},
		{
			ItemID:       "2",
			Title:        "Newer Article",
			Link:         "https://example.com/new",
			CommentsLink: "https://news.ycombinator.com/item?id=2",
			Points:       60,
			CommentCount: 15,
			Author:       "user2",
			CreatedAt:    time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC), // Later
			UpdatedAt:    time.Now(),
		},
	}

	updateStoredItems(db, items)
	retrievedItems := getAllItems(db, 30, 50)

	if len(retrievedItems) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(retrievedItems))
	}

	// Check ordering (should be DESC by created_at, so newer first)
	if retrievedItems[0].Title != "Newer Article" {
		t.Errorf("Expected 'Newer Article' first, got '%s'", retrievedItems[0].Title)
	}
	if retrievedItems[1].Title != "Older Article" {
		t.Errorf("Expected 'Older Article' second, got '%s'", retrievedItems[1].Title)
	}
}

func TestGetOpenGraphData_NotFound(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Try to get non-existent OpenGraph data
	cache, err := getOpenGraphData(db, "https://example.com/nonexistent")
	if err != nil {
		t.Fatalf("Expected no error for non-existent URL, got: %v", err)
	}
	if cache != nil {
		t.Error("Expected nil cache for non-existent URL")
	}
}

func TestGetOpenGraphData_Found(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert test OpenGraph data that hasn't expired
	testURL := "https://example.com/test"
	futureTime := time.Now().Add(24 * time.Hour)
	
	_, err := db.Exec(`
		INSERT INTO opengraph_cache (url, title, description, image, site_name, expires_at, fetch_success)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		testURL, "Test Title", "Test Description", "https://example.com/image.jpg", "Example Site", futureTime, true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Retrieve the data
	cache, err := getOpenGraphData(db, testURL)
	if err != nil {
		t.Fatalf("Error retrieving OpenGraph data: %v", err)
	}
	if cache == nil {
		t.Fatal("Expected cache data, got nil")
	}

	// Verify the data
	if cache.URL != testURL {
		t.Errorf("Expected URL '%s', got '%s'", testURL, cache.URL)
	}
	if cache.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", cache.Title)
	}
	if cache.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", cache.Description)
	}
	if cache.Image != "https://example.com/image.jpg" {
		t.Errorf("Expected image URL 'https://example.com/image.jpg', got '%s'", cache.Image)
	}
	if cache.SiteName != "Example Site" {
		t.Errorf("Expected site name 'Example Site', got '%s'", cache.SiteName)
	}
	if !cache.FetchSuccess {
		t.Error("Expected fetch_success to be true")
	}
}

func TestGetOpenGraphData_Expired(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert expired OpenGraph data
	testURL := "https://example.com/expired"
	pastTime := time.Now().Add(-24 * time.Hour)
	
	_, err := db.Exec(`
		INSERT INTO opengraph_cache (url, title, description, expires_at, fetch_success)
		VALUES (?, ?, ?, ?, ?)`,
		testURL, "Expired Title", "Expired Description", pastTime, true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Try to retrieve expired data
	cache, err := getOpenGraphData(db, testURL)
	if err != nil {
		t.Fatalf("Expected no error for expired data, got: %v", err)
	}
	if cache != nil {
		t.Error("Expected nil cache for expired URL")
	}
}

func TestCacheOpenGraphData_NewEntry(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Create test OpenGraph data
	ogData := &OpenGraphData{
		URL:         "https://example.com/new",
		Title:       "New Title",
		Description: "New Description",
		Image:       "https://example.com/new.jpg",
		SiteName:    "New Site",
	}

	// Cache the data
	err := cacheOpenGraphData(db, ogData, true)
	if err != nil {
		t.Fatalf("Error caching OpenGraph data: %v", err)
	}

	// Verify it was stored
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM opengraph_cache WHERE url = ?", ogData.URL).Scan(&count)
	if err != nil {
		t.Fatalf("Error counting cached entries: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 cached entry, got %d", count)
	}

	// Verify the data
	var title, description, image, siteName string
	var fetchSuccess bool
	err = db.QueryRow("SELECT title, description, image, site_name, fetch_success FROM opengraph_cache WHERE url = ?", 
		ogData.URL).Scan(&title, &description, &image, &siteName, &fetchSuccess)
	if err != nil {
		t.Fatalf("Error retrieving cached data: %v", err)
	}

	if title != ogData.Title {
		t.Errorf("Expected title '%s', got '%s'", ogData.Title, title)
	}
	if description != ogData.Description {
		t.Errorf("Expected description '%s', got '%s'", ogData.Description, description)
	}
	if image != ogData.Image {
		t.Errorf("Expected image '%s', got '%s'", ogData.Image, image)
	}
	if siteName != ogData.SiteName {
		t.Errorf("Expected site name '%s', got '%s'", ogData.SiteName, siteName)
	}
	if !fetchSuccess {
		t.Error("Expected fetch_success to be true")
	}
}

func TestCacheOpenGraphData_UpdateExisting(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	testURL := "https://example.com/update"

	// Insert initial data
	initialData := &OpenGraphData{
		URL:         testURL,
		Title:       "Initial Title",
		Description: "Initial Description",
		Image:       "https://example.com/initial.jpg",
		SiteName:    "Initial Site",
	}
	err := cacheOpenGraphData(db, initialData, true)
	if err != nil {
		t.Fatalf("Error caching initial data: %v", err)
	}

	// Update with new data
	updatedData := &OpenGraphData{
		URL:         testURL,
		Title:       "Updated Title",
		Description: "Updated Description",
		Image:       "https://example.com/updated.jpg",
		SiteName:    "Updated Site",
	}
	err = cacheOpenGraphData(db, updatedData, false) // Test with fetchSuccess=false
	if err != nil {
		t.Fatalf("Error updating cached data: %v", err)
	}

	// Verify there's still only one entry
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM opengraph_cache WHERE url = ?", testURL).Scan(&count)
	if err != nil {
		t.Fatalf("Error counting cached entries: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 cached entry after update, got %d", count)
	}

	// Verify the data was updated
	var title string
	var fetchSuccess bool
	err = db.QueryRow("SELECT title, fetch_success FROM opengraph_cache WHERE url = ?", 
		testURL).Scan(&title, &fetchSuccess)
	if err != nil {
		t.Fatalf("Error retrieving updated data: %v", err)
	}

	if title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", title)
	}
	if fetchSuccess {
		t.Error("Expected fetch_success to be false after update")
	}
}

func TestCleanupExpiredOpenGraphCache(t *testing.T) {
	db := setupTestDB()
	defer func() { _ = db.Close() }()

	// Insert some test data - mix of expired and non-expired
	testData := []struct {
		url       string
		expiresAt time.Time
	}{
		{"https://example.com/expired1", time.Now().Add(-24 * time.Hour)},
		{"https://example.com/expired2", time.Now().Add(-1 * time.Hour)},
		{"https://example.com/valid1", time.Now().Add(24 * time.Hour)},
		{"https://example.com/valid2", time.Now().Add(1 * time.Hour)},
	}

	for _, data := range testData {
		_, err := db.Exec(`
			INSERT INTO opengraph_cache (url, title, expires_at, fetch_success)
			VALUES (?, ?, ?, ?)`,
			data.url, "Test Title", data.expiresAt, true)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Verify we have 4 entries before cleanup
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM opengraph_cache").Scan(&count)
	if err != nil {
		t.Fatalf("Error counting entries before cleanup: %v", err)
	}
	if count != 4 {
		t.Errorf("Expected 4 entries before cleanup, got %d", count)
	}

	// Run cleanup
	err = cleanupExpiredOpenGraphCache(db)
	if err != nil {
		t.Fatalf("Error during cleanup: %v", err)
	}

	// Verify we have 2 entries after cleanup (the non-expired ones)
	err = db.QueryRow("SELECT COUNT(*) FROM opengraph_cache").Scan(&count)
	if err != nil {
		t.Fatalf("Error counting entries after cleanup: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entries after cleanup, got %d", count)
	}

	// Verify the remaining entries are the non-expired ones
	rows, err := db.Query("SELECT url FROM opengraph_cache ORDER BY url")
	if err != nil {
		t.Fatalf("Error querying remaining entries: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var remainingURLs []string
	for rows.Next() {
		var url string
		err := rows.Scan(&url)
		if err != nil {
			t.Fatalf("Error scanning URL: %v", err)
		}
		remainingURLs = append(remainingURLs, url)
	}

	expectedURLs := []string{"https://example.com/valid1", "https://example.com/valid2"}
	if len(remainingURLs) != len(expectedURLs) {
		t.Errorf("Expected %d remaining URLs, got %d", len(expectedURLs), len(remainingURLs))
	}
	for i, expectedURL := range expectedURLs {
		if i >= len(remainingURLs) || remainingURLs[i] != expectedURL {
			t.Errorf("Expected URL '%s' at position %d, got '%s'", expectedURL, i, 
				func() string { if i < len(remainingURLs) { return remainingURLs[i] }; return "none" }())
		}
	}
}