package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test helper functions

func createTestAlgoliaResponse() AlgoliaResponse {
	return AlgoliaResponse{
		Hits: []AlgoliaHit{
			{
				ObjectID:    "12345",
				Title:       "Test Article 1",
				URL:         "https://example.com/article1",
				Author:      "testuser1",
				Points:      75,
				NumComments: 25,
				CreatedAt:   "2024-01-01T12:00:00Z",
			},
			{
				ObjectID:    "67890",
				Title:       "Test Article 2",
				URL:         "https://github.com/test/repo",
				Author:      "testuser2",
				Points:      120,
				NumComments: 45,
				CreatedAt:   "2024-01-01T13:00:00Z",
			},
		},
	}
}

// Mock HTTP server for testing API calls
func createMockServer(response AlgoliaResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func TestAlgoliaResponse_JSONParsing(t *testing.T) {
	jsonData := `{
		"hits": [
			{
				"objectID": "12345",
				"title": "Test Article",
				"url": "https://example.com/test",
				"author": "testuser",
				"points": 75,
				"num_comments": 25,
				"created_at": "2024-01-01T12:00:00Z"
			},
			{
				"objectID": "67890",
				"title": "Another Article",
				"url": "https://github.com/test/repo",
				"author": "anotheruser",
				"points": 120,
				"num_comments": 45,
				"created_at": "2024-01-01T13:00:00Z"
			}
		]
	}`

	var response AlgoliaResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(response.Hits) != 2 {
		t.Errorf("Expected 2 hits, got %d", len(response.Hits))
	}

	// Check first hit
	hit1 := response.Hits[0]
	if hit1.ObjectID != "12345" {
		t.Errorf("Expected ObjectID '12345', got '%s'", hit1.ObjectID)
	}
	if hit1.Title != "Test Article" {
		t.Errorf("Expected Title 'Test Article', got '%s'", hit1.Title)
	}
	if hit1.Points != 75 {
		t.Errorf("Expected Points 75, got %d", hit1.Points)
	}
	if hit1.NumComments != 25 {
		t.Errorf("Expected NumComments 25, got %d", hit1.NumComments)
	}

	// Check second hit
	hit2 := response.Hits[1]
	if hit2.ObjectID != "67890" {
		t.Errorf("Expected ObjectID '67890', got '%s'", hit2.ObjectID)
	}
	if hit2.Author != "anotheruser" {
		t.Errorf("Expected Author 'anotheruser', got '%s'", hit2.Author)
	}
}

func TestFetchHackerNewsItems_WithMockServer(t *testing.T) {
	// Create mock response
	mockResponse := createTestAlgoliaResponse()
	server := createMockServer(mockResponse)
	defer server.Close()

	// This test would require modifying fetchHackerNewsItems to accept a URL parameter
	// For now, we'll test the transformation logic separately
	t.Skip("This test requires refactoring fetchHackerNewsItems to be testable")
}

func TestHackerNewsItemTransformation(t *testing.T) {
	// Test the transformation logic from AlgoliaHit to HackerNewsItem
	hit := AlgoliaHit{
		ObjectID:    "12345",
		Title:       "Test Article",
		URL:         "https://example.com/test",
		Author:      "testuser",
		Points:      75,
		NumComments: 25,
		CreatedAt:   "2024-01-01T12:00:00Z",
	}

	// Simulate the transformation logic from fetchHackerNewsItems
	now := time.Now()
	createdAt, err := time.Parse(time.RFC3339, hit.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	commentsLink := "https://news.ycombinator.com/item?id=" + hit.ObjectID

	item := HackerNewsItem{
		ItemID:       hit.ObjectID,
		Title:        hit.Title,
		Link:         hit.URL,
		CommentsLink: commentsLink,
		Points:       hit.Points,
		CommentCount: hit.NumComments,
		Author:       hit.Author,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
	}

	// Verify transformation
	if item.ItemID != "12345" {
		t.Errorf("Expected ItemID '12345', got '%s'", item.ItemID)
	}
	if item.Title != "Test Article" {
		t.Errorf("Expected Title 'Test Article', got '%s'", item.Title)
	}
	if item.Link != "https://example.com/test" {
		t.Errorf("Expected Link 'https://example.com/test', got '%s'", item.Link)
	}
	if item.CommentsLink != "https://news.ycombinator.com/item?id=12345" {
		t.Errorf("Expected CommentsLink 'https://news.ycombinator.com/item?id=12345', got '%s'", item.CommentsLink)
	}
	if item.Points != 75 {
		t.Errorf("Expected Points 75, got %d", item.Points)
	}
	if item.CommentCount != 25 {
		t.Errorf("Expected CommentCount 25, got %d", item.CommentCount)
	}
	if item.Author != "testuser" {
		t.Errorf("Expected Author 'testuser', got '%s'", item.Author)
	}
	if item.CreatedAt.Year() != 2024 {
		t.Errorf("Expected CreatedAt year 2024, got %d", item.CreatedAt.Year())
	}
}

func TestTimestampParsing_EdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp string
		shouldErr bool
	}{
		{"valid ISO 8601", "2024-01-01T12:00:00Z", false},
		{"valid with milliseconds", "2024-01-01T12:00:00.123Z", false},
		{"valid with timezone", "2024-01-01T12:00:00+02:00", false},
		{"invalid format", "2024-01-01 12:00:00", true},
		{"empty string", "", true},
		{"invalid date", "invalid-date", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := time.Parse(time.RFC3339, tc.timestamp)
			if tc.shouldErr && err == nil {
				t.Errorf("Expected error for timestamp '%s', but got none", tc.timestamp)
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error for timestamp '%s', but got: %v", tc.timestamp, err)
			}
		})
	}
}