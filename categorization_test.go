package main

import (
	"strings"
	"testing"
	"time"
)

// buildExpectedCategories builds the expected categories for a given domain and title
// based on the configuration file
func buildExpectedCategories(domain, title, url string, categoryMapper *CategoryMapper) []string {
	var expected []string

	// Add raw domain if not empty
	if domain != "" {
		expected = append(expected, domain)
	}

	// Add configured category if exists
	if domain != "" && categoryMapper != nil {
		if category := categoryMapper.GetCategoryForDomain(domain); category != "" {
			expected = append(expected, category)
		}
	}

	// Add content type categories (these are not configurable)
	titleLower := strings.ToLower(title)
	switch {
	case strings.HasPrefix(titleLower, "show hn:"):
		expected = append(expected, "Show HN")
	case strings.HasPrefix(titleLower, "ask hn:"):
		expected = append(expected, "Ask HN")
	case strings.Contains(titleLower, "pdf") || strings.HasSuffix(url, ".pdf"):
		expected = append(expected, "PDF")
	case strings.Contains(titleLower, "video"):
		expected = append(expected, "Video")
	case strings.Contains(titleLower, "book") || strings.Contains(titleLower, "ebook"):
		expected = append(expected, "Book")
	}

	return expected
}

func TestCategorizeContent(t *testing.T) {
	// Try to load config from the actual domains.json file
	config, err := loadConfigFromFile("configs/domains.json")
	var categoryMapper *CategoryMapper
	if err != nil {
		t.Logf("Could not load config from domains.json, testing with nil categoryMapper: %v", err)
		categoryMapper = nil
	} else {
		categoryMapper = NewCategoryMapper(config)
	}

	testCases := []struct {
		name   string
		title  string
		domain string
		url    string
	}{
		{
			name:   "GitHub repository",
			title:  "Awesome Go Library",
			domain: "github.com",
			url:    "https://github.com/user/repo",
		},
		{
			name:   "ArXiv paper",
			title:  "Machine Learning Research",
			domain: "arxiv.org",
			url:    "https://arxiv.org/abs/1234.5678",
		},
		{
			name:   "Show HN post",
			title:  "Show HN: My new project",
			domain: "example.com",
			url:    "https://example.com/project",
		},
		{
			name:   "Ask HN post",
			title:  "Ask HN: How do you learn programming?",
			domain: "news.ycombinator.com",
			url:    "https://news.ycombinator.com/item?id=123",
		},
		{
			name:   "PDF document",
			title:  "Research Paper (PDF)",
			domain: "university.edu",
			url:    "https://university.edu/paper.pdf",
		},
		{
			name:   "YouTube video",
			title:  "Tech Tutorial",
			domain: "youtube.com",
			url:    "https://youtube.com/watch?v=123",
		},
		{
			name:   "The Verge article",
			title:  "Tech News Article",
			domain: "theverge.com",
			url:    "https://theverge.com/tech-news",
		},
		{
			name:   "Unknown domain",
			title:  "Some Article",
			domain: "unknown-site.com",
			url:    "https://unknown-site.com/article",
		},
		{
			name:   "Empty domain",
			title:  "Some title",
			domain: "",
			url:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := categorizeContent(tc.title, tc.domain, tc.url, categoryMapper)
			expected := buildExpectedCategories(tc.domain, tc.title, tc.url, categoryMapper)

			if len(result) != len(expected) {
				t.Errorf("Expected %d categories, got %d: expected=%v, actual=%v", len(expected), len(result), expected, result)
				return
			}

			for i, expectedCat := range expected {
				if result[i] != expectedCat {
					t.Errorf("Expected category %d to be '%s', got '%s'", i, expectedCat, result[i])
				}
			}
		})
	}
}

func TestCategorizeByPoints(t *testing.T) {
	testCases := []struct {
		name      string
		points    int
		minPoints int
		expected  string
	}{
		{"viral post", 600, 50, "Viral 500+"},
		{"hot post", 250, 50, "Hot 200+"},
		{"high score post", 150, 50, "High Score 100+"},
		{"double threshold", 120, 50, "High Score 100+"},
		{"above threshold", 75, 50, "Popular 50+"},
		{"at threshold", 50, 50, "Popular 50+"},
		{"below threshold", 25, 50, "Rising"},
		{"custom threshold viral", 600, 100, "Viral 500+"},
		{"custom threshold high", 250, 100, "Hot 200+"},
		{"custom threshold popular", 150, 100, "High Score 100+"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := categorizeByPoints(tc.points, tc.minPoints)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestCalculatePostAge(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hours ago"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2 hours ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1 days ago"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
		{"4 weeks ago", now.Add(-28 * 24 * time.Hour), "4 weeks ago"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculatePostAge(tc.time)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
