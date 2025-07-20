#!/usr/bin/env python3

# Read the current HTML
with open('index.html', 'r') as f:
    html = f.read()

# Add helpful instructions below the buttons
instructions = '''
        <div style="text-align: center; margin-bottom: 2rem; padding: 1rem; background: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0;">
            <p style="font-size: 0.9rem; color: #475569; margin-bottom: 0.5rem;">
                <strong>To subscribe to this calendar:</strong>
            </p>
            <p style="font-size: 0.85rem; color: #64748b; margin-bottom: 0.25rem;">
                <strong>Google Calendar:</strong> Click "Subscribe in Google Calendar" above, or manually add this URL:
            </p>
            <p style="font-size: 0.8rem; color: #64748b; background: #f1f5f9; padding: 0.5rem; border-radius: 4px; font-family: monospace; word-break: break-all; margin-bottom: 0.5rem;">
                https://paulsmith.github.io/eths-girls-swimming/calendar.ics
            </p>
            <p style="font-size: 0.85rem; color: #64748b;">
                <strong>Other calendars:</strong> Download the .ics file and import it into your calendar app
            </p>
        </div>'''

# Insert instructions after the calendar-actions div
insertion_point = '</div>\n        \n        <div id="competitions-container">'
replacement = '</div>\n        ' + instructions + '\n        <div id="competitions-container">'

html = html.replace(insertion_point, replacement)

# Write the updated HTML
with open('index.html', 'w') as f:
    f.write(html)

print("Added subscription instructions")