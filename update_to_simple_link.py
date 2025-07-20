#!/usr/bin/env python3
import re

# Read the current HTML
with open('index.html', 'r') as f:
    html = f.read()

# Replace the button with a simple anchor link
# The Google Calendar URL format is: https://calendar.google.com/calendar/r?cid=URL_ENCODED_ICS_URL
calendar_url = "https://paulsmith.github.io/eths-girls-swimming/calendar.ics"
google_cal_url = f"https://calendar.google.com/calendar/r?cid={calendar_url.replace('https://', '')}"

# Replace the button HTML
old_button = '''<button class="btn btn-google" onclick="subscribeToGoogleCalendar()">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM7 10h5v5H7z"/>
                </svg>
                Subscribe in Google Calendar
            </button>'''

new_link = f'''<a href="{google_cal_url}" class="btn btn-google" target="_blank">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM7 10h5v5H7z"/>
                </svg>
                Subscribe in Google Calendar
            </a>'''

html = html.replace(old_button, new_link)

# Remove the subscribe note div since we don't need it anymore
html = re.sub(r'<div class="subscribe-note".*?</div>\s*', '', html, flags=re.DOTALL)

# Remove the JavaScript function
html = re.sub(r'\s*function subscribeToGoogleCalendar\(\) \{[^}]+\}\s*', '', html, flags=re.DOTALL)

# Write the updated HTML
with open('index.html', 'w') as f:
    f.write(html)

print("Updated to use simple anchor link for Google Calendar subscription")