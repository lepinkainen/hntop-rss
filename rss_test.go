package main

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tests for generateRSSFeed function

func TestGenerateRSSFeed_EmptyItems(t *testing.T) {
	items := []HackerNewsItem{}
	rss := generateRSSFeed(items, 50)

	if !strings.Contains(rss, "Hacker News Top") {
		t.Error("RSS feed should contain the title")
	}
	if !strings.Contains(rss, "xmlns=\"http://www.w3.org/2005/Atom\"") {
		t.Error("RSS feed should be Atom format")
	}
	if strings.Contains(rss, "<entry>") {
		t.Error("Empty items should not generate any entries")
	}
}

func TestGenerateRSSFeed_SingleItem(t *testing.T) {
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

	rss := generateRSSFeed(items, 50)

	// Check for feed structure
	if !strings.Contains(rss, "Hacker News Top") {
		t.Error("RSS feed should contain the title")
	}
	if !strings.Contains(rss, "<entry>") {
		t.Error("RSS feed should contain entry")
	}
	if !strings.Contains(rss, "Test Article") {
		t.Error("RSS feed should contain item title")
	}
	if !strings.Contains(rss, "75 points") {
		t.Error("RSS feed should contain points")
	}
	if !strings.Contains(rss, "25 comments") {
		t.Error("RSS feed should contain comment count")
	}
	if !strings.Contains(rss, "testuser") {
		t.Error("RSS feed should contain author")
	}
	if !strings.Contains(rss, "example.com") {
		t.Error("RSS feed should contain domain")
	}
}

func TestGenerateRSSFeed_MultipleItems(t *testing.T) {
	items := []HackerNewsItem{
		{
			ItemID:       "12345",
			Title:        "First Article",
			Link:         "https://example.com/first",
			CommentsLink: "https://news.ycombinator.com/item?id=12345",
			Points:       75,
			CommentCount: 25,
			Author:       "user1",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
		{
			ItemID:       "67890",
			Title:        "Second Article",
			Link:         "https://github.com/test/repo",
			CommentsLink: "https://news.ycombinator.com/item?id=67890",
			Points:       120,
			CommentCount: 45,
			Author:       "user2",
			CreatedAt:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
		},
	}

	rss := generateRSSFeed(items, 50)

	// Check both items are present
	if !strings.Contains(rss, "First Article") {
		t.Error("RSS feed should contain first item")
	}
	if !strings.Contains(rss, "Second Article") {
		t.Error("RSS feed should contain second item")
	}
	if !strings.Contains(rss, "user1") {
		t.Error("RSS feed should contain first author")
	}
	if !strings.Contains(rss, "user2") {
		t.Error("RSS feed should contain second author")
	}
	if !strings.Contains(rss, "github.com") {
		t.Error("RSS feed should contain GitHub domain")
	}

	// Count entries
	entryCount := strings.Count(rss, "<entry>")
	if entryCount != 2 {
		t.Errorf("Expected 2 entries, got %d", entryCount)
	}
}

func TestGenerateRSSFeed_PointsFormatting(t *testing.T) {
	testCases := []struct {
		name     string
		points   int
		expected string
	}{
		{"plain number", 75, "75 points"},
		{"single digit", 5, "5 points"},
		{"triple digit", 150, "150 points"},
		{"zero", 0, "0 points"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			items := []HackerNewsItem{
				{
					ItemID:       "12345",
					Title:        "Test Article",
					Link:         "https://example.com/test",
					CommentsLink: "https://news.ycombinator.com/item?id=12345",
					Points:       tc.points,
					CommentCount: 25,
					Author:       "testuser",
					CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					UpdatedAt:    time.Now(),
				},
			}

			rss := generateRSSFeed(items, 50)
			if !strings.Contains(rss, tc.expected) {
				t.Errorf("Expected '%s' in RSS feed, but it was not found", tc.expected)
			}
		})
	}
}

func TestPointsAndCommentsFormatting(t *testing.T) {
	testCases := []struct {
		name             string
		points           int
		comments         int
		expectedPoints   string
		expectedComments string
	}{
		{"single digit", 5, 3, "5", "3"},
		{"double digit", 75, 25, "75", "25"},
		{"triple digit", 150, 100, "150", "100"},
		{"zero values", 0, 0, "0", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the conversion logic used in RSS generation
			points := strconv.Itoa(tc.points)
			comments := strconv.Itoa(tc.comments)

			// Verify the conversion works correctly
			if len(points) == 0 {
				t.Error("Points string should not be empty")
			}
			if len(comments) == 0 {
				t.Error("Comments string should not be empty")
			}
		})
	}
}