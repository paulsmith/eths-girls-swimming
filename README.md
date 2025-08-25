# ETHS Girls Swimming Calendar Scraper

A Go application that scrapes competition data from the official Wildkits aquatics website, processes it using LLM, and generates an HTML calendar with responsive design.

## Features

- **3-Phase Pipeline**: Scrape → Process → HTML generation
- **Resumable Execution**: Run individual phases or full pipeline
- **Data Persistence**: Intermediate JSON files for development iteration
- **Real-time Scraping**: Extracts detailed competition info including times and bus schedules
- **LLM Processing**: Intelligent event classification and data formatting
- **Responsive Design**: Grid/timeline toggle with mobile support

## Quick Start

```bash
# Run full pipeline (scrape -> process -> html)
go run scraper.go events.go

# Or use compiled binary
go build -o scraper .
./scraper
```

## Pipeline Phases

### 1. Scrape (`scrape`)
Scrapes competition data from wildkitaquatics.com with session handling.
- Extracts basic info: date, name, location
- Fetches detailed info via form POSTs
- Saves raw data to `scraped_data.json`

### 2. Process (`process`) 
Processes raw data using Claude LLM for intelligent formatting.
- Formats dates and times
- Classifies event types (Meeting, Invitational, Relay, etc.)
- Detects home/away games
- Extracts level information (JV, Varsity, etc.)
- Saves processed data to `processed_events.json`

### 3. HTML (`html`)
Generates responsive HTML calendar from processed events.
- Uses Go html/template
- Maintains all CSS and JavaScript functionality
- Outputs to `index.html`

## CLI Usage

```bash
# Full pipeline
./scraper

# Individual phases
./scraper -from scrape -to scrape     # Only scrape data
./scraper -from process -to process   # Only process existing data  
./scraper -from html                  # Only generate HTML

# Resume from specific phase
./scraper -from process               # Skip scraping, process + HTML
./scraper -from html                  # Skip scraping + processing

# Verbose output
./scraper -v
```

## Options

- `-from <phase>`: Start from phase (scrape, process, html)
- `-to <phase>`: Stop at phase (scrape, process, html)
- `-v`: Verbose logging
- `-h`: Show help

## Files

- `scraped_data.json` - Raw scraped competition data
- `processed_events.json` - LLM-processed event data  
- `index.html` - Generated calendar HTML
- `index.tmpl` - Go template for HTML generation

## Environment

**Required for LLM processing:**
- `ANTHROPIC_API_KEY` - Claude API key for event processing

**Development workflow:**
1. `./scraper -from scrape -to scrape` - Scrape once during development
2. Iterate on LLM prompt: `./scraper -from process -to process`  
3. Iterate on template: `./scraper -from html`
4. Full pipeline for production: `./scraper`

## Event Data Structure

```go
type Event struct {
    Date         string `json:"date"`          // "Saturday, August 23, 2025"
    Time         string `json:"time"`          // "7:00 PM CT"  
    Name         string `json:"name"`          // "Lake Forest Relays"
    Location     string `json:"location"`      // "Lake Forest High School"
    LocationFull string `json:"location_full"` // Full address
    IsHome       bool   `json:"is_home"`       // true for ETHS events
    Type         string `json:"type"`          // Event type
    TypeClass    string `json:"type_class"`    // CSS class for styling
    Levels       string `json:"levels"`        // "JV, Varsity" etc
}
```

## Development Benefits

- **Cost Savings**: Avoid re-scraping during LLM prompt iteration
- **Speed**: Skip expensive phases when tweaking templates  
- **Debugging**: Inspect intermediate JSON files
- **Iteration**: Rapid development cycle for each phase

## Technical Details

- **Session Handling**: Maintains cookies for authenticated scraping
- **Form Processing**: Extracts and submits competition detail forms
- **Error Recovery**: Graceful fallbacks for LLM failures
- **Data Validation**: JSON schema validation between phases
- **Template Safety**: HTML escaping and XSS protection