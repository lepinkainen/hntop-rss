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
- Supports content categorization and filtering
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
- `generateRSSFeed()` - Creates Atom XML feed with OpenGraph metadata
- `categorizeContent()` - Categorizes content by domain and keywords
- `fetchOpenGraphData()` - Extracts and caches OpenGraph metadata

### Dependencies

- `github.com/gorilla/feeds` - RSS/Atom feed generation
- `modernc.org/sqlite` - Pure Go SQLite driver
- Standard library packages for HTTP, JSON, HTML parsing, and concurrency

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

