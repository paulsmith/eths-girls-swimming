#!/usr/bin/env python3
from urllib.parse import quote

# Read the current HTML
with open('index.html', 'r') as f:
    html = f.read()

# The calendar URL
calendar_url = "https://paulsmith.github.io/eths-girls-swimming/calendar.ics"

# Google Calendar expects the URL to be properly encoded
# Format: https://calendar.google.com/calendar/u/0/r?cid=ENCODED_URL
encoded_url = quote(calendar_url, safe='')
google_cal_url = f"https://calendar.google.com/calendar/u/0/r?cid={encoded_url}"

print(f"Calendar URL: {calendar_url}")
print(f"Encoded URL: {encoded_url}")
print(f"Google Calendar URL: {google_cal_url}")

# Replace the existing href
old_href = 'href="https://calendar.google.com/calendar/u/0/r?cid=https://paulsmith.github.io/eths-girls-swimming/calendar.ics"'
new_href = f'href="{google_cal_url}"'

html = html.replace(old_href, new_href)

# Write the updated HTML
with open('index.html', 'w') as f:
    f.write(html)

print("Updated Google Calendar URL with proper encoding")