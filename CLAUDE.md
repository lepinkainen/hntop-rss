# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**IMPORTANT**: Before planning new tasks or making changes, always check `llm-shared/project_tech_stack.md` for project-specific guidelines, preferred libraries, and development standards.

## Project Overview

This is a Go application that fetches Hacker News stories from the Algolia API and generates an RSS feed from high-scoring items. The application:

- Fetches items from the Hacker News Algolia API
- Stores items in a SQLite database (hackernews.db) 
- Updates item statistics using concurrent API calls
- Generates an Atom RSS feed with the top 30 stories
- Includes OpenGraph metadata extraction and caching
- Supports content categorization and filtering with proper Atom `<category>` elements
- Enhanced visual tag spacing for better RSS reader compatibility
- Uses pure Go SQLite driver for database operations

## Build and Development Commands

This project uses [Task](https://taskfile.dev/) for build automation. All commands are defined in `Taskfile.yml`:

- `task` or `task build` - Build the application (runs tests and lint first)
- `task test` - Run tests with coverage report
- `task lint` - Run golangci-lint
- `task clean` - Clean build artifacts
- `task build-linux` - Cross-compile for Linux AMD64
- `task build-ci` - Build for CI environments
- `task test-ci` - Run tests for CI
- `task upgrade-deps` - Upgrade all Go dependencies

**Required**: Always run `task build` before finishing any task to ensure the project builds successfully.

## Architecture

### Core Components

The application is modularized across multiple files:

- **main.go** - Main entry point and orchestration logic
- **api.go** - Hacker News API integration and item fetching
- **database.go** - SQLite database operations and schema management
- **rss.go** - RSS/Atom feed generation
- **opengraph.go** - OpenGraph metadata extraction and caching
- **categorization.go** - Content categorization and filtering logic
- **types.go** - Data structures and type definitions

### Key Functions

- `fetchHackerNewsItems()` - Fetches items from HN Algolia API
- `updateStoredItems()` - Upserts items to SQLite with conflict resolution
- `updateItemStats()` - Updates item statistics with concurrent API calls
- `getAllItems()` - Queries top items filtered by points threshold
- `generateRSSFeed()` - Creates Atom XML feed with OpenGraph metadata and proper categories
- `categorizeContent()` - Categorizes content by domain and keywords with enhanced domain mapping
- `formatDomainName()` - Converts domain names to readable format (e.g., "theverge" → "The Verge")
- `convertToCustomAtom()` - Converts standard feeds to custom Atom format with multiple categories
- `fetchOpenGraphData()` - Extracts and caches OpenGraph metadata

### Dependencies

- `github.com/gorilla/feeds` - RSS/Atom feed generation (extended with custom category support)
- `modernc.org/sqlite` - Pure Go SQLite driver
- Standard library packages for HTTP, JSON, HTML parsing, XML, and concurrency

### RSS/Atom Feed Features

The application generates enhanced Atom feeds with:
- **Proper Atom categories**: Each item includes standards-compliant `<category term="..." label="...">` elements
- **Enhanced visual tags**: Improved CSS styling with better spacing for RSS reader compatibility
- **Smart domain categorization**: Maps common domains to readable names (e.g., "theverge" → "The Verge")
- **Multiple category types**: Domain-based, content-type, and points-based categories
- **OpenGraph integration**: Rich previews with titles, descriptions, and images
- **Custom Atom structures**: Extended gorilla/feeds with proper multi-category support

### Database Schema

The SQLite database includes:
- `items` table - Hacker News item data with points, comments, metadata
- `opengraph_cache` table - Cached OpenGraph metadata with expiration
- Uses UPSERT operations for conflict resolution
- Supports concurrent access with proper locking

## Running the Application

The built binary accepts command-line flags:

```bash
./build/hntop-rss -outdir /path/to/output -debug -minpoints 50
```

- `-outdir` - Directory where RSS feed should be saved (default: current directory)
- `-debug` - Enable debug logging
- `-minpoints` - Minimum points threshold for items (default: 50)

The generated RSS feed is saved as `hntop30.xml` in the specified directory.

