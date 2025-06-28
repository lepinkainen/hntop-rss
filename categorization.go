package main

import (
	"fmt"
	"strings"
	"time"
)

// categorizeContent analyzes content and returns applicable categories based on domain and title
func categorizeContent(title, domain, url string) []string {
	var categories []string
	
	// Domain-based categories
	switch {
	case strings.Contains(domain, "github.com"):
		categories = append(categories, "GitHub")
	case strings.Contains(domain, "arxiv.org"):
		categories = append(categories, "ArXiv")
	case strings.Contains(domain, "medium.com"):
		categories = append(categories, "Medium")
	case strings.Contains(domain, "youtube.com") || strings.Contains(domain, "youtu.be"):
		categories = append(categories, "YouTube")
	case strings.Contains(domain, "reddit.com"):
		categories = append(categories, "Reddit")
	case strings.Contains(domain, "twitter.com") || strings.Contains(domain, "x.com"):
		categories = append(categories, "Twitter")
	default:
		if domain != "" {
			// Extract main domain part for generic categorization
			parts := strings.Split(domain, ".")
			if len(parts) > 1 {
				mainDomain := parts[len(parts)-2]
				// Convert camelCase and concatenated words to proper spacing
				formattedDomain := formatDomainName(mainDomain)
				categories = append(categories, formattedDomain)
			}
		}
	}
	
	// Content type detection
	titleLower := strings.ToLower(title)
	switch {
	case strings.HasPrefix(titleLower, "show hn:"):
		categories = append(categories, "Show HN")
	case strings.HasPrefix(titleLower, "ask hn:"):
		categories = append(categories, "Ask HN")
	case strings.Contains(titleLower, "pdf") || strings.HasSuffix(url, ".pdf"):
		categories = append(categories, "PDF")
	case strings.Contains(titleLower, "video"):
		categories = append(categories, "Video")
	case strings.Contains(titleLower, "book") || strings.Contains(titleLower, "ebook"):
		categories = append(categories, "Book")
	}
	
	return categories
}

// categorizeByPoints returns a category label based on point count and threshold
func categorizeByPoints(points int, minPoints int) string {
	switch {
	case points >= 500:
		return "Viral 500+"
	case points >= 200:
		return "Hot 200+"
	case points >= 100:
		return "High Score 100+"
	case points >= minPoints * 2:
		return fmt.Sprintf("High Score %d+", minPoints * 2)
	case points >= minPoints:
		return fmt.Sprintf("Popular %d+", minPoints)
	default:
		return "Rising"
	}
}

// calculatePostAge returns a human-readable time difference from the given time to now
func calculatePostAge(createdAt time.Time) string {
	now := time.Now()
	diff := now.Sub(createdAt)
	
	switch {
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes < 1 {
			return "just now"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(diff.Hours() / (24 * 7))
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}

// formatDomainName converts domain names to more readable format
// Examples: "theverge" -> "The Verge", "arstechnica" -> "Ars Technica"
func formatDomainName(domain string) string {
	// Handle common domain patterns
	domainMappings := map[string]string{
		"theverge":     "The Verge",
		"arstechnica":  "Ars Technica",
		"techcrunch":   "TechCrunch",
		"stackoverflow": "Stack Overflow",
		"hbr":          "Harvard Business Review",
		"wsj":          "Wall Street Journal",
		"nytimes":      "New York Times",
		"washingtonpost": "Washington Post",
		"bloomberg":    "Bloomberg",
		"reuters":      "Reuters",
		"bbc":          "BBC",
		"cnn":          "CNN",
		"npr":          "NPR",
		"wired":        "Wired",
		"engadget":     "Engadget",
		"venturebeat":  "VentureBeat",
		"fastcompany":  "Fast Company",
		"hackernoon":   "Hacker Noon",
		"dev":          "Dev.to",
		"substack":     "Substack",
	}

	// Check if we have a specific mapping
	if mapped, exists := domainMappings[strings.ToLower(domain)]; exists {
		return mapped
	}

	// For unknown domains, just capitalize first letter
	if len(domain) > 0 {
		return strings.ToUpper(string(domain[0])) + domain[1:]
	}
	
	return domain
}