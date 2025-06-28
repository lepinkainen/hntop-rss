package main

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractOpenGraphTags_BasicTags(t *testing.T) {
	htmlContent := `
	<html>
	<head>
		<meta property="og:title" content="Test Article Title">
		<meta property="og:description" content="This is a test article description">
		<meta property="og:image" content="https://example.com/image.jpg">
		<meta property="og:site_name" content="Test Site">
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	if ogData.Title != "Test Article Title" {
		t.Errorf("Expected title 'Test Article Title', got '%s'", ogData.Title)
	}
	if ogData.Description != "This is a test article description" {
		t.Errorf("Expected description 'This is a test article description', got '%s'", ogData.Description)
	}
	if ogData.Image != "https://example.com/image.jpg" {
		t.Errorf("Expected image 'https://example.com/image.jpg', got '%s'", ogData.Image)
	}
	if ogData.SiteName != "Test Site" {
		t.Errorf("Expected site name 'Test Site', got '%s'", ogData.SiteName)
	}
}

func TestExtractOpenGraphTags_FallbackTitle(t *testing.T) {
	htmlContent := `
	<html>
	<head>
		<title>Fallback Title from Title Tag</title>
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	if ogData.Title != "Fallback Title from Title Tag" {
		t.Errorf("Expected fallback title 'Fallback Title from Title Tag', got '%s'", ogData.Title)
	}
}

func TestExtractOpenGraphTags_FallbackDescription(t *testing.T) {
	htmlContent := `
	<html>
	<head>
		<meta name="description" content="Fallback description from meta tag">
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	if ogData.Description != "Fallback description from meta tag" {
		t.Errorf("Expected fallback description 'Fallback description from meta tag', got '%s'", ogData.Description)
	}
}

func TestExtractOpenGraphTags_PriorityOrder(t *testing.T) {
	// OpenGraph tags should take priority over fallback tags
	htmlContent := `
	<html>
	<head>
		<meta property="og:title" content="OpenGraph Title">
		<meta property="og:description" content="OpenGraph description">
		<title>Fallback Title</title>
		<meta name="description" content="Fallback description">
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	if ogData.Title != "OpenGraph Title" {
		t.Errorf("Expected OpenGraph title to take priority, got '%s'", ogData.Title)
	}
	if ogData.Description != "OpenGraph description" {
		t.Errorf("Expected OpenGraph description to take priority, got '%s'", ogData.Description)
	}
}

func TestExtractOpenGraphTags_FirstValueWins(t *testing.T) {
	// If multiple OpenGraph tags exist, first one should win
	htmlContent := `
	<html>
	<head>
		<meta property="og:title" content="First Title">
		<meta property="og:title" content="Second Title">
		<meta property="og:description" content="First Description">
		<meta property="og:description" content="Second Description">
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	if ogData.Title != "First Title" {
		t.Errorf("Expected first title to win, got '%s'", ogData.Title)
	}
	if ogData.Description != "First Description" {
		t.Errorf("Expected first description to win, got '%s'", ogData.Description)
	}
}

func TestExtractOpenGraphTags_EmptyContent(t *testing.T) {
	htmlContent := `
	<html>
	<head>
		<meta property="og:title" content="">
		<meta property="og:description" content="">
		<title>Should Use This Title</title>
		<meta name="description" content="Should use this description">
	</head>
	<body></body>
	</html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	ogData := &OpenGraphData{
		URL: "https://example.com/test",
	}

	extractOpenGraphTags(doc, ogData)

	// Empty OG tags are treated as missing, so fallbacks are used
	if ogData.Title != "Should Use This Title" {
		t.Errorf("Expected fallback title, got '%s'", ogData.Title)
	}
	if ogData.Description != "Should use this description" {
		t.Errorf("Expected fallback description, got '%s'", ogData.Description)
	}
}

func TestTruncateString_WithinLimit(t *testing.T) {
	input := "Short string"
	result := truncateString(input, 50)
	if result != input {
		t.Errorf("Expected '%s', got '%s'", input, result)
	}
}

func TestTruncateString_ExceedsLimit(t *testing.T) {
	input := "This is a very long string that exceeds the limit"
	maxLen := 20
	result := truncateString(input, maxLen)
	expected := "This is a very lo..." // maxLen(20) - 3 = 17 chars + "..."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	if len(result) != maxLen {
		t.Errorf("Expected result length %d, got %d", maxLen, len(result))
	}
}

func TestTruncateString_ExactLimit(t *testing.T) {
	input := "Exactly twenty chars"
	maxLen := 20
	result := truncateString(input, maxLen)
	if result != input {
		t.Errorf("Expected '%s', got '%s'", input, result)
	}
}

func TestTruncateString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty string", "", 10, ""},
		{"very short limit", "Hello World", 3, "..."},
		{"limit equals ellipsis", "Hello", 3, "..."},
		{"just over ellipsis", "Hello", 4, "H..."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := truncateString(test.input, test.maxLen)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestCleanOpenGraphData_TrimAndTruncate(t *testing.T) {
	ogData := &OpenGraphData{
		URL:         "https://example.com/test",
		Title:       "  Title with leading and trailing spaces  ",
		Description: "  Description with spaces  ",
		SiteName:    "  Site Name  ",
		Image:       "https://example.com/image.jpg",
	}

	cleanOpenGraphData(ogData)

	if ogData.Title != "Title with leading and trailing spaces" {
		t.Errorf("Expected trimmed title, got '%s'", ogData.Title)
	}
	if ogData.Description != "Description with spaces" {
		t.Errorf("Expected trimmed description, got '%s'", ogData.Description)
	}
	if ogData.SiteName != "Site Name" {
		t.Errorf("Expected trimmed site name, got '%s'", ogData.SiteName)
	}
}

func TestCleanOpenGraphData_PreservesLongFields(t *testing.T) {
	longTitle := strings.Repeat("A", 250)       // Long title
	longDescription := strings.Repeat("B", 600) // Long description
	longSiteName := strings.Repeat("C", 150)    // Long site name

	ogData := &OpenGraphData{
		URL:         "https://example.com/test",
		Title:       longTitle,
		Description: longDescription,
		SiteName:    longSiteName,
		Image:       "https://example.com/image.jpg",
	}

	cleanOpenGraphData(ogData)

	// Fields should preserve their original length (no truncation)
	if len(ogData.Title) != 250 {
		t.Errorf("Expected title length 250, got %d", len(ogData.Title))
	}
	if ogData.Title != longTitle {
		t.Error("Expected title to be preserved without truncation")
	}

	if len(ogData.Description) != 600 {
		t.Errorf("Expected description length 600, got %d", len(ogData.Description))
	}
	if ogData.Description != longDescription {
		t.Error("Expected description to be preserved without truncation")
	}

	if len(ogData.SiteName) != 150 {
		t.Errorf("Expected site name length 150, got %d", len(ogData.SiteName))
	}
	if ogData.SiteName != longSiteName {
		t.Error("Expected site name to be preserved without truncation")
	}
}

func TestCleanOpenGraphData_InvalidImageURL(t *testing.T) {
	ogData := &OpenGraphData{
		URL:         "https://example.com/test",
		Title:       "Test Title",
		Description: "Test Description",
		SiteName:    "Test Site",
		Image:       "://invalid-url-scheme",
	}

	cleanOpenGraphData(ogData)

	if ogData.Image != "" {
		t.Errorf("Expected invalid image URL to be cleared, got '%s'", ogData.Image)
	}
}

func TestCleanOpenGraphData_ValidImageURL(t *testing.T) {
	validImageURL := "https://example.com/image.jpg"
	ogData := &OpenGraphData{
		URL:         "https://example.com/test",
		Title:       "Test Title",
		Description: "Test Description",
		SiteName:    "Test Site",
		Image:       validImageURL,
	}

	cleanOpenGraphData(ogData)

	if ogData.Image != validImageURL {
		t.Errorf("Expected valid image URL to be preserved, got '%s'", ogData.Image)
	}
}

func TestCleanOpenGraphData_EmptyImageURL(t *testing.T) {
	ogData := &OpenGraphData{
		URL:         "https://example.com/test",
		Title:       "Test Title",
		Description: "Test Description",
		SiteName:    "Test Site",
		Image:       "",
	}

	cleanOpenGraphData(ogData)

	if ogData.Image != "" {
		t.Errorf("Expected empty image URL to remain empty, got '%s'", ogData.Image)
	}
}
