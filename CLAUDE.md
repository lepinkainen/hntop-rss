# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**IMPORTANT**: Before planning new tasks or making changes, always check `llm-shared/project_tech_stack.md` for project-specific guidelines, preferred libraries, and development standards.

## Project Overview

This is a Go application that scrapes Hacker News stories and generates an RSS feed from high-scoring items (>50 points). The application:

- Scrapes the Hacker News front page
- Stores items in a SQLite database (hackernews.db)
- Generates an Atom RSS feed with the top 30 stories
- Uses pure Go SQLite driver for database operations

## Build and Development Commands

This project uses [Task](https://taskfile.dev/) for build automation. All commands are defined in `Taskfile.yml`:

- `task` or `task build` - Build the application (runs tests and lint first)
- `task test` - Run tests with coverage report
- `task lint` - Run golangci-lint
- `task clean` - Clean build artifacts
- `task build-linux` - Cross-compile for Linux AMD64
- `task upgrade-deps` - Upgrade all Go dependencies

**Required**: Always run `task build` before finishing any task to ensure the project builds successfully.

## Architecture

### Core Components

- **main.go** - Single-file application containing all functionality
- **Database Schema** - SQLite table `items` with Hacker News item data
- **RSS Generation** - Uses gorilla/feeds to generate Atom XML

### Key Functions

- `fetchHackerNewsItems()` - Scrapes HN front page using goquery
- `updateStoredItems()` - Upserts items to SQLite with conflict resolution
- `getAllItems()` - Queries top 30 items with >50 points
- `generateRSSFeed()` - Creates Atom XML feed

### Dependencies

- `github.com/PuerkitoBio/goquery` - HTML parsing and scraping
- `github.com/gorilla/feeds` - RSS/Atom feed generation
- `github.com/sirupsen/logrus` - Structured logging
- `modernc.org/sqlite` - Pure Go SQLite driver

## Database

The application creates `hackernews.db` in the executable's directory. The database uses UPSERT operations to handle duplicate items based on Hacker News item ID.

## Running the Application

The built binary accepts an `-outdir` flag to specify where the RSS feed should be saved:

```bash
./build/hntop-rss -outdir /path/to/output
```

The generated RSS feed is saved as `hntop30.xml` in the specified directory.

