package main

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/feeds"
)

// generateRSSFeed creates an Atom RSS feed from the provided items
func generateRSSFeed(items []HackerNewsItem, minPoints int) string {
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

	domainRegex := regexp.MustCompile(`^https?://([^/]+)`)

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

		// Enhanced HTML description with categories
		categoryTags := ""
		if len(categories) > 0 {
			categoryTags = "<div style=\"margin-bottom: 8px;\">"
			for _, cat := range categories {
				categoryTags += fmt.Sprintf("<span style=\"display: inline-block; background: #e5e5e5; color: #666; padding: 2px 6px; border-radius: 12px; font-size: 12px; margin-right: 4px;\">%s</span>", cat)
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

		feed.Items = append(feed.Items, rssItem)
	}

	rss, err := feed.ToAtom()
	if err != nil {
		slog.Error("Failed to generate RSS feed", "error", err)
		os.Exit(1)
	}

	slog.Debug("RSS feed generated successfully", "feedSize", len(rss))
	return rss
}