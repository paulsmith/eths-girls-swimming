# Calendar Sync Implementation

This document explains the automated calendar sync system for ETHS Girls Swimming.

## Overview

The sync system automatically updates the swimming calendar from the official source at wildkitaquatics.com. It handles the complex navigation requirements and form POST operations needed to extract event details.

## Components

### 1. Core Sync Script (`sync_calendar.py`)
- **Purpose**: Main scraping and generation script
- **Features**:
  - Handles session-based navigation (main page → competitions)
  - Processes form POST requests for event details
  - Generates updated HTML and ICS calendar files
  - Supports dry-run mode for testing
  - Comprehensive error handling

### 2. Template Generator (`template_generator.py`)
- **Purpose**: Extracts HTML structure for reusability
- **Features**:
  - Creates reusable templates from existing HTML
  - Preserves styling and JavaScript functionality
  - Generates event card templates

### 3. GitHub Actions Workflow (`.github/workflows/sync-calendar.yml`)
- **Schedule**: Weekly on Sundays at 6 AM UTC (1 AM CDT)
- **Manual Trigger**: Available with dry-run option
- **Features**:
  - Automatic dependency installation
  - Change detection
  - Pull request creation when updates found
  - Comprehensive validation steps

## Usage

### Manual Sync (Local)
```bash
# Install dependencies
uv pip install -r requirements.txt

# Dry run (no changes)
python sync_calendar.py --dry-run --verbose

# Full sync
python sync_calendar.py --verbose
```

### Manual Sync (GitHub Actions)
1. Go to Actions tab in GitHub repository
2. Select "Sync Swimming Calendar" workflow
3. Click "Run workflow"
4. Choose dry-run option if desired
5. Click "Run workflow"

### Automated Sync
- Runs automatically every Sunday at 6 AM UTC
- Creates PR if changes detected
- Requires manual review and merge

## Technical Details

### Navigation Flow
1. **Session Establishment**: Visit main page to get cookies/session state
2. **Navigation**: Click "Competitions" link to access calendar
3. **Event Extraction**: Parse competition listing page
4. **Detail Fetching**: POST to info links for each event
5. **Data Processing**: Clean and structure event data
6. **Generation**: Create new HTML and ICS files

### Form POST Handling
The script automatically detects and handles form POST requests:
- Extracts form action and method
- Collects form data (inputs, selects, textareas)
- Submits POST request with session cookies
- Parses response for detailed event information

### Error Handling
- Network timeouts and connection issues
- Missing or changed page structure
- Form processing failures
- Date/time parsing errors
- Fallback to basic event information

### Data Validation
- Date format standardization
- Event type classification
- Location parsing and cleanup
- Home/away game detection
- Time format consistency

## File Structure
```
/
├── sync_calendar.py           # Main sync script
├── template_generator.py      # Template extraction utility
├── requirements.txt           # Python dependencies
├── README_SYNC.md            # This documentation
├── .github/workflows/
│   └── sync-calendar.yml     # GitHub Actions workflow
├── index.html                # Generated HTML calendar
└── calendar.ics              # Generated ICS calendar
```

## Dependencies
- `requests`: HTTP client for web scraping
- `beautifulsoup4`: HTML parsing and manipulation
- `icalendar`: ICS calendar file generation
- `pytz`: Timezone handling
- `lxml`: Fast XML/HTML parsing

## Configuration

### Event Type Mapping
- "invitational" or "invite" → Invitational
- "relay" → Relay Meet
- "conference" → Conference
- "sectional" or "state" → Championship
- "meeting" → Meeting
- Default → Dual Meet

### CSS Class Mapping
- Invitational → `invitational`
- Relay Meet → `relay`
- Conference → `conference`
- Championship → `championship`
- Meeting → `meeting`
- Special Event → `special`

### Time Zones
- All times assumed to be Central Time (America/Chicago)
- ICS calendar uses UTC with timezone information
- Default event duration: 2 hours

## Troubleshooting

### Common Issues
1. **Website Structure Changes**: Update selectors in `_parse_competitions_page()`
2. **Form Changes**: Update form handling in `_handle_form_post()`
3. **Date Format Changes**: Update patterns in `_extract_date()`

### Debug Mode
Use `--verbose` flag for detailed logging:
```bash
python sync_calendar.py --verbose --dry-run
```

### Manual Testing
1. Check website accessibility: Visit https://wildkitaquatics.com/main/EvanstonGirlsSwimming
2. Navigate to competitions page manually
3. Test form POST operations
4. Verify event data structure

## Security Considerations
- User-Agent headers set to avoid blocking
- Session handling for authentication
- No sensitive data stored
- Safe HTML parsing with BeautifulSoup
- Input validation for all scraped data

## Maintenance
- Review weekly for website changes
- Monitor GitHub Actions for failures
- Update selectors if page structure changes
- Validate generated calendars periodically
- Keep dependencies updated for security