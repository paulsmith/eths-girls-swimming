#!/usr/bin/env python3
"""
ETHS Girls Swimming Calendar Sync Script

Scrapes the official calendar from wildkitaquatics.com and generates
updated index.html and calendar.ics files.

Requirements:
- Python 3.6+
- requests, beautifulsoup4, icalendar libraries

Usage:
    python sync_calendar.py [--dry-run] [--verbose]
"""

import requests
from bs4 import BeautifulSoup
import re
from datetime import datetime, timedelta
import json
import argparse
import sys
import os
import time
import html
from urllib.parse import urljoin, parse_qs, urlparse
from icalendar import Calendar, Event
import pytz
from typing import List, Dict, Optional


class ETHSCalendarScraper:
    def __init__(self, verbose: bool = False):
        self.session = requests.Session()
        self.verbose = verbose
        self.base_url = "https://wildkitaquatics.com"
        self.events = []
        
        # Set headers to mimic a real browser
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
        })
    
    def log(self, message: str):
        """Print verbose logging messages."""
        if self.verbose:
            print(f"[INFO] {message}")
    
    def scrape_calendar(self) -> List[Dict]:
        """
        Main scraping method that handles the required navigation flow:
        1. Visit main page to establish session
        2. Navigate to competitions page
        3. Extract event list
        4. For each event, POST to get details
        """
        try:
            # Step 1: Visit main page to establish session/cookies
            self.log("Visiting main page to establish session...")
            main_response = self.session.get(f"{self.base_url}/main/EvanstonGirlsSwimming", timeout=30)
            main_response.raise_for_status()
            
            # Step 2: Navigate to competitions page
            self.log("Navigating to competitions page...")
            # Look for the competitions link in the main page
            soup = BeautifulSoup(main_response.text, 'html.parser')
            competitions_link = self._find_competitions_link(soup)
            
            if not competitions_link:
                raise Exception("Could not find competitions link on main page")
            
            competitions_url = urljoin(self.base_url, competitions_link)
            self.log(f"Found competitions URL: {competitions_url}")
            
            competitions_response = self.session.get(competitions_url, timeout=30)
            competitions_response.raise_for_status()
            
            # Step 3: Parse competitions page and extract event list
            self.log("Parsing competitions page...")
            competitions_soup = BeautifulSoup(competitions_response.text, 'html.parser')
            events_data = self._parse_competitions_page(competitions_soup)
            
            # Step 4: Get detailed info for each event via form POST
            self.log(f"Found {len(events_data)} events, fetching details...")
            for i, event_data in enumerate(events_data):
                self.log(f"Processing event {i+1}/{len(events_data)}: {event_data.get('name', 'Unknown')}")
                detailed_event = self._get_event_details(event_data)
                if detailed_event:
                    self.events.append(detailed_event)
                # Add rate limiting between requests
                time.sleep(1)
            
            self.log(f"Successfully scraped {len(self.events)} events")
            return self.events
            
        except Exception as e:
            print(f"Error scraping calendar: {str(e)}")
            return []
    
    def _find_competitions_link(self, soup: BeautifulSoup) -> Optional[str]:
        """Find the competitions link in the navigation."""
        # Look for various possible link patterns
        competitions_links = soup.find_all('a', string=re.compile(r'Competitions?', re.I))
        if competitions_links:
            return competitions_links[0].get('href')
        
        # Alternative: look in navigation menu
        nav_links = soup.find_all('a', href=re.compile(r'compet', re.I))
        if nav_links:
            return nav_links[0].get('href')
        
        return None
    
    def _parse_competitions_page(self, soup: BeautifulSoup) -> List[Dict]:
        """Parse the competitions page and extract basic event information."""
        events = []
        
        # Look for the competitions table
        table = soup.find('table', class_='table')
        if not table:
            self.log("Could not find competitions table")
            return events
        
        # Find all rows in the tbody (skip header)
        tbody = table.find('tbody')
        if not tbody:
            # If no tbody, look directly in table
            tbody = table
        
        event_rows = tbody.find_all('tr')
        self.log(f"Found {len(event_rows)} table rows")
        
        for row in event_rows:
            event_data = self._parse_event_row(row)
            if event_data:
                events.append(event_data)
        
        return events
    
    def _parse_event_row(self, row) -> Optional[Dict]:
        """Parse a single event row to extract basic information."""
        try:
            # Get all table cells
            cells = row.find_all('td')
            if len(cells) < 4:
                # Not a valid event row (probably header or incomplete)
                return None
            
            # Extract data from each column
            date_text = cells[0].get_text(strip=True)
            event_name = cells[1].get_text(strip=True)
            location = cells[2].get_text(strip=True)
            
            # Look for info form in the last cell
            info_form = cells[3].find('form')
            
            # Skip if no valid date or event name
            if not date_text or not event_name or len(event_name) < 2:
                return None
            
            # Normalize the date format
            normalized_date = self._extract_date(date_text)
            if not normalized_date:
                normalized_date = date_text  # Fallback to original
            
            return {
                'date_text': normalized_date,
                'name': event_name,
                'location': location,
                'info_element': info_form,
                'row_html': str(row)
            }
        except Exception as e:
            self.log(f"Error parsing event row: {str(e)}")
            return None
    
    def _extract_date(self, text: str) -> Optional[str]:
        """Extract and normalize date from text using regex patterns."""
        # Handle the specific format used on the website: "Wed, Aug 13"
        short_date_pattern = r'\b(\w+),?\s+(\w+)\s+(\d{1,2})\b'
        match = re.search(short_date_pattern, text)
        if match:
            day_name, month_abbr, day = match.groups()
            # Convert to full date with current year (will be 2025 season)
            year = 2025
            
            # Map month abbreviations to full names
            month_map = {
                'Jan': 'January', 'Feb': 'February', 'Mar': 'March',
                'Apr': 'April', 'May': 'May', 'Jun': 'June',
                'Jul': 'July', 'Aug': 'August', 'Sep': 'September',
                'Oct': 'October', 'Nov': 'November', 'Dec': 'December'
            }
            
            month_full = month_map.get(month_abbr, month_abbr)
            
            # Handle date ranges like "Nov 14 - Nov 15"
            if '-' in text:
                return f"{day_name}, {month_full} {day}, {year}"
            else:
                return f"{day_name}, {month_full} {day}, {year}"
        
        # Fallback patterns for other formats
        date_patterns = [
            r'\b(\w+day),?\s+(\w+)\s+(\d{1,2}),?\s+(\d{4})\b',  # "Friday, August 29, 2025"
            r'\b(\w+)\s+(\d{1,2}),?\s+(\d{4})\b',  # "August 29, 2025"
            r'\b(\d{1,2})/(\d{1,2})/(\d{4})\b',  # "8/29/2025"
        ]
        
        for pattern in date_patterns:
            match = re.search(pattern, text)
            if match:
                return match.group(0)
        
        return None
    
    def _extract_event_name(self, text: str) -> str:
        """Extract event name from text."""
        # Remove common prefixes and clean up
        lines = [line.strip() for line in text.split('\n') if line.strip()]
        
        # Look for opponent/event name (usually not a date or time)
        for line in lines:
            if not re.search(r'\d{1,2}:\d{2}|\d{1,2}/\d{1,2}/\d{4}|\w+day', line, re.I):
                if len(line) > 3 and len(line) < 50:
                    return line
        
        return "Unknown Event"
    
    def _get_event_details(self, event_data: Dict) -> Optional[Dict]:
        """Get detailed event information via form POST."""
        try:
            info_element = event_data.get('info_element')
            if not info_element:
                return self._create_basic_event(event_data)
            
            # Handle form POST
            if info_element.name == 'form':
                return self._handle_form_post(info_element, event_data)
            elif info_element.name == 'a':
                return self._handle_link_click(info_element, event_data)
            else:
                return self._create_basic_event(event_data)
                
        except Exception as e:
            self.log(f"Error getting event details: {str(e)}")
            return self._create_basic_event(event_data)
    
    def _handle_form_post(self, form_element, event_data: Dict) -> Optional[Dict]:
        """Handle form POST to get event details."""
        try:
            form_action = form_element.get('action', '')
            form_method = form_element.get('method', 'GET').upper()
            
            # Extract form data
            form_data = {}
            for input_elem in form_element.find_all(['input', 'select', 'textarea']):
                name = input_elem.get('name')
                value = input_elem.get('value', '')
                if name:
                    form_data[name] = value
            
            # Make the POST request
            post_url = urljoin(self.base_url, form_action)
            
            if form_method == 'POST':
                response = self.session.post(post_url, data=form_data, timeout=30)
            else:
                response = self.session.get(post_url, params=form_data, timeout=30)
            
            response.raise_for_status()
            
            # Parse the response for detailed event info
            detail_soup = BeautifulSoup(response.text, 'html.parser')
            return self._parse_event_details(detail_soup, event_data)
            
        except Exception as e:
            self.log(f"Error in form POST: {str(e)}")
            return self._create_basic_event(event_data)
    
    def _handle_link_click(self, link_element, event_data: Dict) -> Optional[Dict]:
        """Handle clicking an info link."""
        try:
            href = link_element.get('href', '')
            detail_url = urljoin(self.base_url, href)
            
            response = self.session.get(detail_url, timeout=30)
            response.raise_for_status()
            
            detail_soup = BeautifulSoup(response.text, 'html.parser')
            return self._parse_event_details(detail_soup, event_data)
            
        except Exception as e:
            self.log(f"Error clicking info link: {str(e)}")
            return self._create_basic_event(event_data)
    
    def _parse_event_details(self, soup: BeautifulSoup, basic_event: Dict) -> Dict:
        """Parse detailed event information from the detail page."""
        # Extract more detailed information
        event = self._create_basic_event(basic_event)
        
        # Look for time information
        time_text = self._extract_time(soup.get_text())
        if time_text:
            event['time'] = time_text
        
        # Look for location information
        location = self._extract_location(soup.get_text())
        if location:
            event['location'] = location
        
        # Look for event type
        event_type = self._extract_event_type(soup.get_text())
        if event_type:
            event['type'] = event_type
        
        return event
    
    def _create_basic_event(self, event_data: Dict) -> Dict:
        """Create a basic event structure."""
        location = event_data.get('location', 'TBD')
        event_name = event_data.get('name', 'Unknown Event')
        
        # Determine event type from name
        event_type = self._extract_event_type(event_name)
        
        return {
            'name': event_name,
            'date': event_data.get('date_text', ''),
            'time': '5:00 PM',  # Default time
            'location': location,
            'type': event_type,
            'home': location == 'ETHS',
            'raw_data': event_data
        }
    
    def _extract_time(self, text: str) -> Optional[str]:
        """Extract time from text."""
        time_pattern = r'\b(\d{1,2}):(\d{2})\s*(AM|PM)?\b'
        match = re.search(time_pattern, text, re.I)
        if match:
            return match.group(0)
        return None
    
    def _extract_location(self, text: str) -> Optional[str]:
        """Extract location from text."""
        # Look for common location patterns
        location_patterns = [
            r'at\s+([^,\n]+)',
            r'@\s+([^,\n]+)',
            r'Location:\s*([^,\n]+)',
        ]
        
        for pattern in location_patterns:
            match = re.search(pattern, text, re.I)
            if match:
                return match.group(1).strip()
        
        return None
    
    def _extract_event_type(self, text: str) -> str:
        """Determine event type from text."""
        text_lower = text.lower()
        
        if 'invitational' in text_lower or 'invite' in text_lower:
            return 'Invitational'
        elif 'relay' in text_lower:
            return 'Relay Meet'
        elif 'conference' in text_lower:
            return 'Conference'
        elif 'sectional' in text_lower:
            return 'Championship'
        elif 'state' in text_lower:
            return 'Championship'
        elif 'meeting' in text_lower:
            return 'Meeting'
        else:
            return 'Dual Meet'


def generate_html_calendar(events: List[Dict]) -> str:
    """Generate HTML calendar from events data."""
    # Read existing HTML as template using absolute path
    script_dir = os.path.dirname(os.path.abspath(__file__))
    html_path = os.path.join(script_dir, 'index.html')
    try:
        with open(html_path, 'r', encoding='utf-8') as f:
            current_html = f.read()
    except FileNotFoundError:
        print("Error: index.html not found")
        return ""
    
    # Parse current HTML to extract structure
    soup = BeautifulSoup(current_html, 'html.parser')
    competitions_list = soup.find('div', id='competitions-list')
    
    if not competitions_list:
        print("Error: Could not find competitions list in HTML")
        return current_html
    
    # Clear existing events
    competitions_list.clear()
    
    # Generate new event cards with sanitized content
    for i, event in enumerate(events):
        card_html = generate_event_card(event, i)
        # Parse with html.parser and sanitize
        card_soup = BeautifulSoup(card_html, 'html.parser')
        competitions_list.append(card_soup)
    
    # Update total count in JavaScript
    script_tag = soup.find('script')
    if script_tag:
        script_content = script_tag.string
        if script_content:
            updated_script = re.sub(
                r'totalCompetitions = \d+',
                f'totalCompetitions = {len(events)}',
                script_content
            )
            script_tag.string = updated_script
    
    return str(soup)


def generate_event_card(event: Dict, index: int) -> str:
    """Generate HTML for a single event card with sanitized content."""
    # Parse date
    try:
        date_obj = datetime.strptime(event['date'], '%A, %B %d, %Y')
        formatted_date = date_obj.strftime('%A, %B %d, %Y')
    except:
        formatted_date = html.escape(str(event['date']))
    
    # Sanitize all user content
    event_name = html.escape(str(event.get('name', 'Unknown Event')))
    event_time = html.escape(str(event.get('time', '5:00 PM')))
    location = html.escape(str(event.get('location', 'TBD')))
    event_type = html.escape(str(event.get('type', 'Dual Meet')))
    
    # Determine if home/away
    is_home = location == 'ETHS' or 'ETHS' in location
    home_indicator = '<span class="home-indicator">(Home)</span>' if is_home else ''
    
    # Map event type to CSS class (safe since we're using predefined values)
    type_class = {
        'Invitational': 'invitational',
        'Relay Meet': 'relay',
        'Conference': 'conference',
        'Championship': 'championship',
        'Meeting': 'meeting',
        'Special Event': 'special',
    }.get(event.get('type', 'Dual Meet'), '')
    
    card_html = f'''
    <div class="competition-card">
        <div class="comp-date">{formatted_date}</div>
        
        <div class="comp-time">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd"></path>
            </svg>
            {event_time}
        </div>
        <h3 class="comp-name">{event_name}</h3>
        <div class="comp-location">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M5.05 4.05a7 7 0 119.9 9.9L10 18.9l-4.95-4.95a7 7 0 010-9.9zM10 11a2 2 0 100-4 2 2 0 000 4z" clip-rule="evenodd"></path>
            </svg>
            <span title="{location}">{location}</span>
            {home_indicator}
        </div>
        <span class="comp-type {type_class}">{event_type}</span>
    </div>
    '''
    
    return card_html


def generate_ics_calendar(events: List[Dict]) -> str:
    """Generate ICS calendar file from events data."""
    cal = Calendar()
    cal.add('prodid', '-//paulsmith.github.io//ETHS Girls Swimming//EN')
    cal.add('version', '2.0')
    cal.add('calscale', 'GREGORIAN')
    cal.add('method', 'PUBLISH')
    cal.add('x-wr-calname', 'ETHS Girls Swimming')
    
    central = pytz.timezone('America/Chicago')
    
    for i, event_data in enumerate(events):
        event = Event()
        
        # Parse date and time
        try:
            date_str = event_data['date']
            time_str = event_data.get('time', '5:00 PM')
            
            # Combine date and time
            datetime_str = f"{date_str} {time_str}"
            
            # Try multiple date formats
            for date_format in ['%A, %B %d, %Y %I:%M %p', '%B %d, %Y %I:%M %p']:
                try:
                    dt = datetime.strptime(datetime_str, date_format)
                    break
                except ValueError:
                    continue
            else:
                # Fallback to a reasonable default date (August 1st of current year)
                current_year = datetime.now().year
                dt = datetime(current_year, 8, 1, 17, 0, 0, 0)
            
            # Localize to Central Time with DST handling
            try:
                dt_central = central.localize(dt)
            except pytz.AmbiguousTimeError:
                # Handle DST transitions by defaulting to standard time
                dt_central = central.localize(dt, is_dst=False)
            
        except Exception as e:
            print(f"Error parsing date for event {i}: {e}")
            dt_central = central.localize(datetime.now().replace(hour=17, minute=0))
        
        # Set event properties
        event.add('dtstart', dt_central)
        event.add('dtend', dt_central + timedelta(hours=2))  # 2-hour duration
        event.add('dtstamp', datetime.now())
        event.add('uid', f'eths-girls-swim-{i}@paulsmith.github.io')
        event.add('created', datetime.now())
        event.add('last-modified', datetime.now())
        event.add('sequence', 0)
        event.add('status', 'CONFIRMED')
        event.add('transp', 'TRANSPARENT')
        
        # Event details
        summary = f"ETHS Girls Swimming - {event_data['name']}"
        event.add('summary', summary)
        
        location = event_data.get('location', 'TBD')
        event.add('location', location)
        
        description = f"ETHS Girls Swimming {event_data.get('type', 'Dual Meet')}"
        event.add('description', description)
        
        cal.add_component(event)
    
    return cal.to_ical().decode('utf-8')


def main():
    parser = argparse.ArgumentParser(description='Sync ETHS Girls Swimming calendar')
    parser.add_argument('--dry-run', action='store_true', help='Print changes without writing files')
    parser.add_argument('--verbose', action='store_true', help='Verbose output')
    
    args = parser.parse_args()
    
    # Initialize scraper
    scraper = ETHSCalendarScraper(verbose=args.verbose)
    
    # Scrape events
    print("Scraping calendar from wildkitaquatics.com...")
    events = scraper.scrape_calendar()
    
    if not events:
        print("No events found. Exiting.")
        return 1
    
    print(f"Found {len(events)} events")
    
    if args.verbose:
        for event in events:
            print(f"  - {event['name']} on {event['date']}")
    
    # Generate new files
    print("Generating new calendar files...")
    
    try:
        # Generate HTML
        html_content = generate_html_calendar(events)
        
        # Generate ICS
        ics_content = generate_ics_calendar(events)
        
        if args.dry_run:
            print("DRY RUN - Would update the following files:")
            print("- index.html")
            print("- calendar.ics")
            print(f"\nHTML preview (first 500 chars):\n{html_content[:500]}...")
            print(f"\nICS preview (first 500 chars):\n{ics_content[:500]}...")
        else:
            # Write files using absolute paths
            script_dir = os.path.dirname(os.path.abspath(__file__))
            html_path = os.path.join(script_dir, 'index.html')
            ics_path = os.path.join(script_dir, 'calendar.ics')
            
            with open(html_path, 'w', encoding='utf-8') as f:
                f.write(html_content)
            
            with open(ics_path, 'w', encoding='utf-8') as f:
                f.write(ics_content)
            
            print("Calendar files updated successfully!")
    
    except Exception as e:
        print(f"Error generating calendar files: {str(e)}")
        return 1
    
    return 0


if __name__ == '__main__':
    sys.exit(main())