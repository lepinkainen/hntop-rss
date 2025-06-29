# hntop-rss

A Go application that creates curated RSS feeds from Hacker News content with enhanced categorization and rich metadata.

## Features

- Fetches stories from the Hacker News Algolia API
- Generates Atom RSS feeds with the top 30 stories
- Smart content categorization with readable domain names
- OpenGraph metadata extraction for rich previews
- Configurable points threshold filtering
- Concurrent API calls for optimal performance
- SQLite storage with automatic cleanup

## Quick Start

### Prerequisites

- Go 1.24.0+
- [Task](https://taskfile.dev/) for build automation

### Build and Run

```bash
# Build the application
task build

# Run with default settings
./build/hntop-rss

# Run with custom output directory and debug logging
./build/hntop-rss -outdir /path/to/output -debug -minpoints 100
```

## Usage

```bash
./build/hntop-rss [options]
```

### Options

- `-outdir string` - Directory where RSS feed should be saved (default: current directory)
- `-debug` - Enable debug logging
- `-minpoints int` - Minimum points threshold for items (default: 50)
- `-config string` - Path to local configuration file (optional)
- `-config-url string` - URL to remote configuration file (optional)

## Configuration

### Domain Mappings

By default, the application fetches domain categorization mappings from a remote URL to convert raw domains (like "github.com") into readable category names (like "GitHub").

**Default behavior:** Fetches from [this repository](https://raw.githubusercontent.com/lepinkainen/hntop-rss/refs/heads/main/configs/domains.json)

**Configuration options:**

```bash
# Use only local configuration file (disables remote fetching)
./hntop-rss -config /path/to/local-domains.json

# Use custom remote URL for domain mappings
./hntop-rss -config-url https://example.com/custom-domains.json

# Use local file with custom remote fallback
./hntop-rss -config /path/to/local.json -config-url https://example.com/backup.json
```

**Configuration format:**

```json
{
  "category_domains": {
    "GitHub": ["github.com"],
    "YouTube": ["youtube.com", "youtu.be"],
    "The Verge": ["theverge.com"]
  }
}
```

If configuration loading fails, domain mapping is disabled and the application continues with basic categorization.

## Development

### Build Commands

- `task` or `task build` - Build the application (runs tests and lint first)
- `task test` - Run tests with coverage report
- `task lint` - Run golangci-lint
- `task clean` - Clean build artifacts
- `task build-linux` - Cross-compile for Linux AMD64
- `task upgrade-deps` - Upgrade all Go dependencies

### Output

The generated RSS feed is saved as `hackernews.xml` in the specified output directory, containing categorized items with OpenGraph metadata and rich previews.
