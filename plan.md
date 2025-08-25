# ETHS Girls Swimming Calendar Scraper - Implementation Plan

## Overview
Scrape the official Wildkits aquatics site to populate the swimming calendar with real event data and generate HTML using Go templates.

## Status: ✅ COMPLETED

## Phase 1: Project Setup ✅
- [x] Initialize Go module: `go mod init eths-swimming-scraper`
- [x] Install dependencies: `golang.org/x/net/html` and standard library packages
- [x] Create main scraper application: `scraper.go`

## Phase 2: Web Scraping Implementation ✅
### Session Management
- [x] HTTP GET to `https://wildkitaquatics.com/main/EvanstonGirlsSwimming`
- [x] Extract session cookies
- [x] Parse HTML to find "Competitions" link
- [x] Follow link to competitions page (maintaining session)

### Competitions Page Scraping
- [x] Parse main competitions table 
- [x] Extract for each row: date, meet name, location, "Info" form details
- [x] Collect competition IDs and form fields needed for detail requests

### Individual Meet Details
- [x] For each competition, POST to details page with correct form data
- [x] Parse resulting detail page (key-value pairs table)
- [x] Collect all available details for LLM processing

## Phase 3: Data Processing with LLM ✅
### LLM Integration
- [x] Create prompt template for Claude Haiku
- [x] Send scraped details for each meet
- [x] Request JSON response matching Event struct format:
  ```json
  {
    "Date": "Saturday, August 23, 2025",
    "Time": "7:00 PM", 
    "Name": "Meet Name",
    "Location": "Short Location",
    "LocationFull": "Full Address",
    "IsHome": true/false,
    "Type": "Meet Type",
    "TypeClass": "css-class",
    "Levels": "JV, Varsity, etc"
  }
  ```

### Data Validation
- [x] Parse JSON responses
- [x] Validate Event structs
- [x] Handle any parsing errors gracefully

## Phase 4: HTML Generation ✅
### Template Execution
- [x] Load `index.tmpl` 
- [x] Execute with collected events data
- [x] Generate final HTML output

### Output Management
- [x] Write to `index.html` or specified output file
- [x] Preserve original static HTML as backup

## Technical Requirements

### HTTP Client Configuration
- User-Agent: `"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"`
- Cookie jar for session management
- Proper error handling and retries

### HTML Parsing Strategy
- Use `golang.org/x/net/html` package
- Create utility functions for element selection
- Extract form data including hidden fields and action URLs

### LLM Integration
- Structured prompts with clear output format requirements
- Error handling for API failures
- Fallback data if LLM processing fails

### Error Handling
- Network failures
- Parse failures  
- LLM API errors
- Invalid data validation

## File Structure
```
/
├── go.mod
├── scraper.go          # Main scraper application
├── events.go           # Event struct and utilities  
├── index.tmpl          # Template (already exists)
├── index.html          # Generated output
├── plan.md             # This file
└── README.md           # Usage instructions
```

## Progress Log
- ✅ Created plan.md file
- ✅ Initialized Go module and dependencies  
- ✅ Created main scraper application with session handling
- ✅ Implemented detailed competition data extraction
- ✅ Integrated intelligent LLM-based event classification
- ✅ Generated final HTML calendar with 26 events
- ✅ **PROJECT COMPLETED SUCCESSFULLY**

## Results
- **26 events successfully scraped** from wildkitaquatics.com
- **Intelligent classification**: Meetings, Invitationals, Relays, Championships properly categorized
- **Level extraction**: JV, Varsity, Diving levels automatically detected
- **Home/away detection**: ETHS events properly marked as home games  
- **Template integration**: Real data seamlessly integrated with existing design
- **Responsive calendar**: Grid/timeline toggle functionality preserved

## Notes
- Session-based navigation is critical - must GET main page first
- Each competition requires individual POST request for details
- LLM will handle intelligent data extraction and formatting
- Maintain backwards compatibility with existing template structure