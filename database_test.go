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

	// Create table schema
	createTable := `
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

	_, err = db.Exec(createTable)
	if err != nil {
		panic(err)
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