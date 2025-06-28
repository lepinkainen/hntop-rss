package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/feeds"
)

// Custom Atom structures to support multiple categories
type AtomCategory struct {
	XMLName xml.Name `xml:"category"`
	Term    string   `xml:"term,attr"`
	Label   string   `xml:"label,attr,omitempty"`
	Scheme  string   `xml:"scheme,attr,omitempty"`
}

type CustomAtomEntry struct {
	XMLName     xml.Name                `xml:"entry"`
	Xmlns       string                  `xml:"xmlns,attr,omitempty"`
	Title       string                  `xml:"title"`
	Updated     string                  `xml:"updated"`
	Id          string                  `xml:"id"`
	Categories  []AtomCategory          `xml:"category"`
	Content     *feeds.AtomContent      `xml:"content,omitempty"`
	Rights      string                  `xml:"rights,omitempty"`
	Source      string                  `xml:"source,omitempty"`
	Published   string                  `xml:"published,omitempty"`
	Links       []feeds.AtomLink        `xml:"link"`
	Summary     *feeds.AtomSummary      `xml:"summary,omitempty"`
	Author      *feeds.AtomAuthor       `xml:"author,omitempty"`
}

type CustomAtomFeed struct {
	XMLName     xml.Name           `xml:"feed"`
	Xmlns       string             `xml:"xmlns,attr"`
	Title       string             `xml:"title"`
	Id          string             `xml:"id"`
	Updated     string             `xml:"updated"`
	Link        *feeds.AtomLink    `xml:"link,omitempty"`
	Author      *feeds.AtomAuthor  `xml:"author,omitempty"`
	Subtitle    string             `xml:"subtitle,omitempty"`
	Rights      string             `xml:"rights,omitempty"`
	Entries     []*CustomAtomEntry `xml:"entry"`
}

// convertToCustomAtom converts a standard Feed to a CustomAtomFeed with proper categories
func convertToCustomAtom(feed *feeds.Feed, itemCategories map[string][]string) *CustomAtomFeed {
	atom := &feeds.Atom{Feed: feed}
	standardAtomFeed := atom.AtomFeed()
	
	customFeed := &CustomAtomFeed{
		Xmlns:    "http://www.w3.org/2005/Atom",
		Title:    standardAtomFeed.Title,
		Id:       standardAtomFeed.Id,
		Updated:  standardAtomFeed.Updated,
		Link:     standardAtomFeed.Link,
		Author:   standardAtomFeed.Author,
		Subtitle: standardAtomFeed.Subtitle,
		Rights:   standardAtomFeed.Rights,
	}
	
	// Convert entries with categories
	for _, entry := range standardAtomFeed.Entries {
		customEntry := &CustomAtomEntry{
			Title:     entry.Title,
			Updated:   entry.Updated,
			Id:        entry.Id,
			Content:   entry.Content,
			Rights:    entry.Rights,
			Source:    entry.Source,
			Published: entry.Published,
			Links:     entry.Links,
			Summary:   entry.Summary,
			Author:    entry.Author,
		}
		
		// Add categories for this entry
		if categories, exists := itemCategories[entry.Id]; exists {
			for _, cat := range categories {
				customEntry.Categories = append(customEntry.Categories, AtomCategory{
					Term:  cat,
					Label: cat,
				})
			}
		}
		
		customFeed.Entries = append(customFeed.Entries, customEntry)
	}
	
	return customFeed
}

// getOpenGraphWithFallback fetches OpenGraph data with caching and fallback
func getOpenGraphWithFallback(db *sql.DB, fetcher *OpenGraphFetcher, url string) *OpenGraphData {
	// Skip OpenGraph fetching if database is nil (for testing)
	if db == nil {
		return nil
	}
	
	// First check cache
	cached, err := getOpenGraphData(db, url)
	if err != nil {
		slog.Warn("Error getting cached OpenGraph data", "error", err, "url", url)
	}
	
	// Return cached data if available and successful
	if cached != nil && cached.FetchSuccess {
		return &OpenGraphData{
			URL:         cached.URL,
			Title:       cached.Title,
			Description: cached.Description,
			Image:       cached.Image,
			SiteName:    cached.SiteName,
		}
	}
	
	// Skip fetching if we have a recent failed attempt
	if cached != nil && !cached.FetchSuccess {
		slog.Debug("Skipping OpenGraph fetch due to recent failure", "url", url)
		return nil
	}
	
	// Fetch fresh data
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	ogData, err := fetcher.FetchOpenGraph(ctx, url)
	fetchSuccess := err == nil && ogData != nil
	
	if err != nil {
		slog.Debug("Failed to fetch OpenGraph data", "error", err, "url", url)
		// Cache the failure to avoid repeated attempts
		if ogData == nil {
			ogData = &OpenGraphData{URL: url}
		}
	} else if ogData != nil {
		cleanOpenGraphData(ogData)
		slog.Debug("Successfully fetched OpenGraph data", "url", url, "title", ogData.Title)
	}
	
	// Cache the result (success or failure)
	if ogData != nil {
		if err := cacheOpenGraphData(db, ogData, fetchSuccess); err != nil {
			slog.Warn("Failed to cache OpenGraph data", "error", err, "url", url)
		}
	}
	
	if fetchSuccess {
		return ogData
	}
	
	return nil
}

// generateRSSFeed creates an Atom RSS feed from the provided items with OpenGraph data
func generateRSSFeed(db *sql.DB, items []HackerNewsItem, minPoints int) string {
	slog.Debug("Generating RSS feed", "itemCount", len(items))
	now := time.Now()
	
	feed := &feeds.Feed{
		Title:       "Hacker News Top Stories",
		Description: "High-quality Hacker News stories, updated regularly",
		Link:        &feeds.Link{Href: "https://news.ycombinator.com/", Rel: "self", Type: "text/html"},
		Id:          "tag:news.ycombinator.com,2024:feed",
		Created:     now,
		Updated:     now,
	}

	// Track categories for each item (using CommentsLink as the ID)
	itemCategories := make(map[string][]string)

	domainRegex := regexp.MustCompile(`^https?://([^/]+)`)
	
	// Initialize OpenGraph fetcher
	ogFetcher := NewOpenGraphFetcher()
	slog.Debug("Initialized OpenGraph fetcher")

	for _, item := range items {
		// Extract domain from the article link
		domain := ""
		if matches := domainRegex.FindStringSubmatch(item.Link); len(matches) > 1 {
			domain = matches[1]
		}

		// Generate categories
		categories := categorizeContent(item.Title, domain, item.Link)
		pointCategory := categorizeByPoints(item.Points, minPoints)
		categories = append(categories, pointCategory)
		
		// Calculate post age
		postAge := calculatePostAge(item.CreatedAt)
		
		// Calculate engagement ratio
		engagementRatio := float64(item.CommentCount) / float64(item.Points)
		engagementText := ""
		if engagementRatio > 0.5 {
			engagementText = "🔥 High engagement"
		} else if engagementRatio > 0.3 {
			engagementText = "💬 Good discussion"
		}
		
		// Fetch OpenGraph data for the article
		var ogPreview string
		if item.Link != "" {
			slog.Debug("Fetching OpenGraph data for item", "hn_id", item.ItemID, "url", item.Link)
			ogData := getOpenGraphWithFallback(db, ogFetcher, item.Link)
			if ogData != nil && (ogData.Title != "" || ogData.Description != "") {
				ogPreview = fmt.Sprintf(`<div style="margin-bottom: 16px; padding: 12px; background: #f9f9f9; border-radius: 6px; border-left: 3px solid #007acc;">
					<h4 style="margin: 0 0 8px 0; color: #007acc; font-size: 14px;">📄 Article Preview</h4>
					%s
					%s
					%s
				</div>`,
					func() string {
						if ogData.Title != "" && ogData.Title != item.Title {
							return fmt.Sprintf(`<p style="margin: 0 0 6px 0; font-weight: bold; color: #333;">%s</p>`, ogData.Title)
						}
						return ""
					}(),
					func() string {
						if ogData.Description != "" {
							return fmt.Sprintf(`<p style="margin: 0 0 6px 0; color: #666; line-height: 1.4; font-size: 13px;">%s</p>`, ogData.Description)
						}
						return ""
					}(),
					func() string {
						if ogData.Image != "" {
							return fmt.Sprintf(`<img src="%s" alt="Article image" style="max-width: 100%%; height: auto; border-radius: 4px; margin-top: 8px;" loading="lazy">`, ogData.Image)
						}
						return ""
					}())
			}
		}

		// Enhanced HTML description with categories
		categoryTags := ""
		if len(categories) > 0 {
			categoryTags = "<div style=\"margin-bottom: 8px; line-height: 1.8;\">"
			for i, cat := range categories {
				// Add space between tags for better RSS reader compatibility
				if i > 0 {
					categoryTags += " "
				}
				categoryTags += fmt.Sprintf("<span style=\"display: inline-block; background: #e5e5e5; color: #666; padding: 3px 8px; border-radius: 12px; font-size: 12px; margin-right: 6px; margin-bottom: 2px; white-space: nowrap;\">%s</span>", cat)
			}
			categoryTags += "</div>"
		}
		
		description := fmt.Sprintf(`<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.5;">
			<div style="margin-bottom: 12px; padding: 8px; background-color: #f6f6ef; border-left: 4px solid #ff6600;">
				<strong style="color: #ff6600;">%d points</strong> • 
				<strong style="color: #666;">%d comments</strong> • 
				<span style="color: #828282;">%s</span>
				%s
			</div>
			
			%s
			
			%s
			
			<div style="margin-bottom: 8px;">
				<strong>Source:</strong> <code style="background: #f4f4f4; padding: 2px 4px; border-radius: 3px;">%s</code>
			</div>
			
			<div style="margin-bottom: 12px;">
				<strong>Author:</strong> <span style="color: #666;">%s</span>
			</div>
			
			<div style="margin-top: 16px; padding-top: 12px; border-top: 1px solid #e5e5e5;">
				<a href="%s" style="display: inline-block; padding: 6px 12px; background-color: #ff6600; color: white; text-decoration: none; border-radius: 4px; margin-right: 8px;">💬 HN Discussion</a>
				<a href="%s" style="display: inline-block; padding: 6px 12px; background-color: #666; color: white; text-decoration: none; border-radius: 4px;">📖 Read Article</a>
			</div>
		</div>`,
			item.Points,
			item.CommentCount,
			postAge,
			func() string {
				if engagementText != "" {
					return " • " + engagementText
				}
				return ""
			}(),
			categoryTags,
			ogPreview,
			domain,
			item.Author,
			item.CommentsLink,
			item.Link)

		rssItem := &feeds.Item{
			Title: item.Title,
			Link:  &feeds.Link{Href: item.CommentsLink, Rel: "alternate", Type: "text/html"},
			Id:    item.CommentsLink,
			Author: &feeds.Author{
				Name: item.Author,
			},
			Description: description,
			Created:     item.CreatedAt,
		}

		// Store categories for this item (using the same ID as the rssItem)
		itemCategories[item.CommentsLink] = categories

		feed.Items = append(feed.Items, rssItem)
	}

	// Generate custom Atom feed with proper categories
	customAtomFeed := convertToCustomAtom(feed, itemCategories)
	
	// Convert to XML
	xmlData, err := xml.MarshalIndent(customAtomFeed, "", "  ")
	if err != nil {
		slog.Error("Failed to generate RSS feed", "error", err)
		os.Exit(1)
	}

	// Add XML header
	rss := xml.Header + string(xmlData)

	slog.Debug("RSS feed generated successfully", "feedSize", len(rss))
	return rss
}