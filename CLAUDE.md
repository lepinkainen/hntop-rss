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
- Features a flexible configuration system supporting local JSON files and remote config URLs
- Includes comprehensive test coverage across all major components

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
- `task validate-config` - Validate JSON configuration files
- `task release-check` - Check GoReleaser configuration
- `task release-snapshot` - Build snapshot release locally (for testing)
- `task release-dry-run` - Test release process without publishing

**Required**: Always run `task build` before finishing any task to ensure the project builds successfully.

### Testing

The project includes comprehensive test coverage with dedicated test files for each major component. Run tests with:

```bash
task test
```

This generates coverage reports and validates that all components work correctly. Test coverage is tracked in `coverage.out`.

## Architecture

### Core Components

The application is modularized across multiple files:

- **main.go** - Main entry point and orchestration logic
- **api.go** - Hacker News API integration and item fetching
- **database.go** - SQLite database operations and schema management
- **feed.go** - RSS/Atom feed generation
- **opengraph.go** - OpenGraph metadata extraction and caching
- **categorization.go** - Content categorization and filtering logic
- **config.go** - Configuration management with support for local and remote JSON configs
- **types.go** - Data structures and type definitions

### Test Files

The project includes comprehensive test coverage:

- **api_test.go** - Tests for API functionality
- **categorization_test.go** - Tests for categorization logic
- **database_test.go** - Tests for database operations
- **feed_test.go** - Tests for RSS feed generation
- **main_test.go** - Tests for main application logic
- **opengraph_test.go** - Tests for OpenGraph functionality

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

The project (module `github.com/lepinkainen/hntop-rss`) uses:

- `github.com/gorilla/feeds` v1.2.0 - RSS/Atom feed generation (extended with custom category support)
- `modernc.org/sqlite` v1.38.0 - Pure Go SQLite driver
- Standard library packages for HTTP, JSON, HTML parsing, XML, and concurrency

### Configuration

The application supports flexible configuration through:

- **Local JSON files**: Use `-config path/to/config.json` to specify local domain mapping configuration
- **Remote configuration**: Use `-config-url https://example.com/config.json` to fetch configuration from URLs
- **Default configuration**: Built-in domain mappings in `configs/domains.json`

The configuration system allows dynamic categorization of content based on domain mappings and can be updated without recompiling the application.

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
./build/hntop-rss -outdir /path/to/output -debug -min-points 50 -config configs/domains.json
```

- `-outdir` - Directory where RSS feed should be saved (default: current directory)
- `-debug` - Enable debug logging
- `-min-points` - Minimum points threshold for items (default: 50)
- `-config` - Path to local JSON configuration file for domain mappings
- `-config-url` - URL to remote JSON configuration file for domain mappings

The generated RSS feed is saved as `hackernews.xml` in the specified directory.

## Release Process

This project uses [GoReleaser](https://goreleaser.com/) for automated releases with GitHub Actions.

### Creating a Release

1. **Create and push a git tag**:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically**:
   - Runs tests and linting
   - Builds for all platforms (Linux, macOS, Windows)
   - Creates GitHub release with binaries and checksums
   - Generates changelog from commits

### Testing Releases Locally

Before creating a tag, test the release process:

```bash
# Check GoReleaser configuration
task release-check

# Build snapshot release locally
task release-snapshot

# Test release process without publishing
task release-dry-run
```

Built artifacts are placed in the `dist/` directory when using GoReleaser commands.
