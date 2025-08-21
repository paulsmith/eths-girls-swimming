#!/usr/bin/env python3
"""
Template Generator for ETHS Swimming Calendar

Extracts the HTML structure from index.html and creates a reusable template
that can be populated with scraped data.
"""

from bs4 import BeautifulSoup
import re
import os


def extract_template():
    """Extract HTML template from current index.html"""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    html_path = os.path.join(script_dir, 'index.html')
    try:
        with open(html_path, 'r', encoding='utf-8') as f:
            html_content = f.read()
    except FileNotFoundError:
        print("Error: index.html not found")
        return None
    
    soup = BeautifulSoup(html_content, 'html.parser')
    
    # Find the competitions list
    competitions_list = soup.find('div', id='competitions-list')
    if not competitions_list:
        print("Error: competitions-list not found")
        return None
    
    # Extract one event card as template
    first_card = competitions_list.find('div', class_='competition-card')
    if not first_card:
        print("Error: no competition cards found")
        return None
    
    # Clear all event cards
    competitions_list.clear()
    
    # Add placeholder comment
    placeholder = soup.new_string('\n        <!-- Events will be inserted here by sync script -->\n        ')
    competitions_list.append(placeholder)
    
    # Update JavaScript totalCompetitions to placeholder
    script_tag = soup.find('script')
    if script_tag and script_tag.string:
        script_content = script_tag.string
        updated_script = re.sub(
            r'totalCompetitions = \d+',
            'totalCompetitions = {{TOTAL_COMPETITIONS}}',
            script_content
        )
        script_tag.string = updated_script
    
    return str(soup), str(first_card)


def create_event_card_template(sample_card_html):
    """Create a template for event cards with placeholders"""
    # Replace specific values with template placeholders
    template = sample_card_html
    
    # Replace date
    template = re.sub(
        r'<div class="comp-date">[^<]+</div>',
        '<div class="comp-date">{{EVENT_DATE}}</div>',
        template
    )
    
    # Replace time
    template = re.sub(
        r'(<svg[^>]*>[^<]*</svg>\s*)([^<\n]+)',
        r'\1{{EVENT_TIME}}',
        template
    )
    
    # Replace event name
    template = re.sub(
        r'<h3 class="comp-name">[^<]+</h3>',
        '<h3 class="comp-name">{{EVENT_NAME}}</h3>',
        template
    )
    
    # Replace location
    template = re.sub(
        r'<span title="[^"]*">[^<]+</span>',
        '<span title="{{EVENT_LOCATION}}">{{EVENT_LOCATION_DISPLAY}}</span>',
        template
    )
    
    # Replace home indicator
    template = re.sub(
        r'<span class="home-indicator">\(Home\)</span>',
        '{{HOME_INDICATOR}}',
        template
    )
    
    # Replace event type
    template = re.sub(
        r'<span class="comp-type[^"]*">[^<]+</span>',
        '<span class="comp-type {{TYPE_CLASS}}">{{EVENT_TYPE}}</span>',
        template
    )
    
    # Replace levels if present
    template = re.sub(
        r'<div class="levels">[^<]+</div>',
        '{{LEVELS}}',
        template
    )
    
    return template


def save_templates():
    """Extract and save HTML templates"""
    html_template, sample_card = extract_template()
    
    if not html_template or not sample_card:
        return False
    
    # Create event card template
    card_template = create_event_card_template(sample_card)
    
    # Save main template using absolute paths
    script_dir = os.path.dirname(os.path.abspath(__file__))
    template_path = os.path.join(script_dir, 'template.html')
    card_template_path = os.path.join(script_dir, 'event_card_template.html')
    
    with open(template_path, 'w', encoding='utf-8') as f:
        f.write(html_template)
    
    # Save card template
    with open(card_template_path, 'w', encoding='utf-8') as f:
        f.write(card_template)
    
    print("Templates saved:")
    print("- template.html (main page structure)")
    print("- event_card_template.html (individual event card)")
    
    return True


if __name__ == '__main__':
    success = save_templates()
    if not success:
        exit(1)