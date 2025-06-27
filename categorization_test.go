package main

import (
	"testing"
	"time"
)

func TestCategorizeContent(t *testing.T) {
	testCases := []struct {
		name     string
		title    string
		domain   string
		url      string
		expected []string
	}{
		{
			name:     "GitHub repository",
			title:    "Awesome Go Library",
			domain:   "github.com",
			url:      "https://github.com/user/repo",
			expected: []string{"GitHub"},
		},
		{
			name:     "ArXiv paper",
			title:    "Machine Learning Research",
			domain:   "arxiv.org",
			url:      "https://arxiv.org/abs/1234.5678",
			expected: []string{"ArXiv"},
		},
		{
			name:     "Show HN post",
			title:    "Show HN: My new project",
			domain:   "example.com",
			url:      "https://example.com/project",
			expected: []string{"Example", "Show HN"},
		},
		{
			name:     "Ask HN post",
			title:    "Ask HN: How do you learn programming?",
			domain:   "news.ycombinator.com",
			url:      "https://news.ycombinator.com/item?id=123",
			expected: []string{"Ycombinator", "Ask HN"},
		},
		{
			name:     "PDF document",
			title:    "Research Paper (PDF)",
			domain:   "university.edu",
			url:      "https://university.edu/paper.pdf",
			expected: []string{"University", "PDF"},
		},
		{
			name:     "YouTube video",
			title:    "Tech Tutorial",
			domain:   "youtube.com",
			url:      "https://youtube.com/watch?v=123",
			expected: []string{"YouTube"},
		},
		{
			name:     "Empty domain",
			title:    "Some title",
			domain:   "",
			url:      "",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := categorizeContent(tc.title, tc.domain, tc.url)
			
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d categories, got %d: %v", len(tc.expected), len(result), result)
				return
			}
			
			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Expected category %d to be '%s', got '%s'", i, expected, result[i])
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