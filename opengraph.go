package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// OpenGraph fetcher with rate limiting and domain-based delays
type OpenGraphFetcher struct {
	client      *http.Client
	domainMutex sync.Mutex
	lastFetch   map[string]time.Time
	semaphore   chan struct{}
	urlMutexes  sync.Map // URL -> *sync.Mutex for preventing concurrent fetches of same URL
}

// NewOpenGraphFetcher creates a new OpenGraph fetcher with rate limiting
func NewOpenGraphFetcher() *OpenGraphFetcher {
	return &OpenGraphFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Limit redirects to 10
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		lastFetch: make(map[string]time.Time),
		semaphore: make(chan struct{}, 5), // Max 5 concurrent fetches
	}
}

// FetchOpenGraph fetches OpenGraph data from a URL with rate limiting
func (f *OpenGraphFetcher) FetchOpenGraph(ctx context.Context, targetURL string) (*OpenGraphData, error) {
	// Get or create a mutex for this URL to prevent concurrent fetches
	urlMutexInterface, _ := f.urlMutexes.LoadOrStore(targetURL, &sync.Mutex{})
	urlMutex := urlMutexInterface.(*sync.Mutex)

	// Lock for this specific URL
	urlMutex.Lock()
	defer urlMutex.Unlock()

	// Acquire semaphore slot
	select {
	case f.semaphore <- struct{}{}:
		defer func() { <-f.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Parse URL to get domain
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	domain := parsedURL.Host

	// Apply domain-based rate limiting
	f.domainMutex.Lock()
	if lastFetch, exists := f.lastFetch[domain]; exists {
		timeSinceLastFetch := time.Since(lastFetch)
		if timeSinceLastFetch < time.Second {
			sleepTime := time.Second - timeSinceLastFetch
			f.domainMutex.Unlock()
			slog.Debug("Rate limiting domain", "domain", domain, "sleep", sleepTime)
			select {
			case <-time.After(sleepTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			f.domainMutex.Lock()
		}
	}
	f.lastFetch[domain] = time.Now()
	f.domainMutex.Unlock()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set proper User-Agent
	req.Header.Set("User-Agent", "HNTop-RSS/1.0 (OpenGraph fetcher)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	slog.Debug("Fetching OpenGraph data", "url", targetURL)

	// Make the request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return nil, fmt.Errorf("not an HTML page: %s", contentType)
	}

	// Limit response body size to 1MB
	limitedReader := io.LimitReader(resp.Body, 1024*1024)

	// Parse HTML
	doc, err := html.Parse(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract OpenGraph data
	ogData := &OpenGraphData{
		URL: targetURL,
	}

	extractOpenGraphTags(doc, ogData)

	slog.Debug("Extracted OpenGraph data", "url", targetURL, "title", ogData.Title, "hasDescription", ogData.Description != "")

	return ogData, nil
}

// extractOpenGraphTags recursively extracts OpenGraph meta tags from HTML
func extractOpenGraphTags(n *html.Node, ogData *OpenGraphData) {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var property, content string
		for _, attr := range n.Attr {
			switch attr.Key {
			case "property":
				property = attr.Val
			case "content":
				content = attr.Val
			}
		}

		// Extract OpenGraph properties
		switch property {
		case "og:title":
			if ogData.Title == "" {
				ogData.Title = content
			}
		case "og:description":
			if ogData.Description == "" {
				ogData.Description = content
			}
		case "og:image":
			if ogData.Image == "" {
				ogData.Image = content
			}
		case "og:site_name":
			if ogData.SiteName == "" {
				ogData.SiteName = content
			}
		}
	}

	// Also check for fallback title in <title> tag
	if n.Type == html.ElementNode && n.Data == "title" && ogData.Title == "" {
		if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
			ogData.Title = strings.TrimSpace(n.FirstChild.Data)
		}
	}

	// Also check for meta description fallback
	if n.Type == html.ElementNode && n.Data == "meta" && ogData.Description == "" {
		var name, content string
		for _, attr := range n.Attr {
			switch attr.Key {
			case "name":
				name = attr.Val
			case "content":
				content = attr.Val
			}
		}
		if name == "description" {
			ogData.Description = content
		}
	}

	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractOpenGraphTags(c, ogData)
	}
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// cleanOpenGraphData cleans and validates OpenGraph data
func cleanOpenGraphData(ogData *OpenGraphData) {
	// Clean up whitespace from fields
	ogData.Title = strings.TrimSpace(ogData.Title)
	ogData.Description = strings.TrimSpace(ogData.Description)
	ogData.SiteName = strings.TrimSpace(ogData.SiteName)

	// Validate image URL
	if ogData.Image != "" {
		if _, err := url.Parse(ogData.Image); err != nil {
			ogData.Image = ""
		}
	}
}
