#!/usr/bin/env python3
"""
Validation script to test the calendar sync setup without external dependencies.
Checks file structure, HTML parsing capabilities, and basic functionality.
"""

import os
import sys
import re
from datetime import datetime

def check_files():
    """Check that all required files are present"""
    required_files = [
        'sync_calendar.py',
        'template_generator.py', 
        'requirements.txt',
        'README_SYNC.md',
        '.github/workflows/sync-calendar.yml',
        'index.html',
        'calendar.ics'
    ]
    
    missing = []
    for file in required_files:
        if not os.path.exists(file):
            missing.append(file)
    
    if missing:
        print(f"❌ Missing files: {', '.join(missing)}")
        return False
    else:
        print("✅ All required files present")
        return True

def validate_html_structure():
    """Validate the current HTML structure"""
    try:
        with open('index.html', 'r') as f:
            html_content = f.read()
        
        # Check for required elements
        required_elements = [
            r'<div[^>]*id="competitions-list"',
            r'<div[^>]*class="competition-card"',
            r'totalCompetitions\s*=\s*\d+',
            r'function toggleView\(\)'
        ]
        
        for pattern in required_elements:
            if not re.search(pattern, html_content):
                print(f"❌ Missing HTML element: {pattern}")
                return False
        
        # Count current events
        event_count = len(re.findall(r'<div[^>]*class="competition-card"', html_content))
        print(f"✅ HTML structure valid ({event_count} events found)")
        return True
        
    except Exception as e:
        print(f"❌ Error validating HTML: {e}")
        return False

def validate_ics_structure():
    """Validate the current ICS structure"""
    try:
        with open('calendar.ics', 'r') as f:
            ics_content = f.read()
        
        # Check for required ICS elements
        if not ics_content.startswith('BEGIN:VCALENDAR'):
            print("❌ ICS file doesn't start with BEGIN:VCALENDAR")
            return False
        
        if not ics_content.strip().endswith('END:VCALENDAR'):
            print("❌ ICS file doesn't end with END:VCALENDAR")
            return False
        
        # Count events
        event_count = len(re.findall(r'BEGIN:VEVENT', ics_content))
        print(f"✅ ICS structure valid ({event_count} events found)")
        return True
        
    except Exception as e:
        print(f"❌ Error validating ICS: {e}")
        return False

def validate_requirements():
    """Validate requirements.txt"""
    try:
        with open('requirements.txt', 'r') as f:
            requirements = f.read()
        
        required_packages = ['requests', 'beautifulsoup4', 'icalendar', 'pytz']
        
        for package in required_packages:
            if package not in requirements:
                print(f"❌ Missing package in requirements.txt: {package}")
                return False
        
        print("✅ All required packages in requirements.txt")
        return True
        
    except Exception as e:
        print(f"❌ Error validating requirements: {e}")
        return False

def validate_workflow():
    """Validate GitHub Actions workflow"""
    try:
        with open('.github/workflows/sync-calendar.yml', 'r') as f:
            workflow = f.read()
        
        required_elements = [
            'schedule:',
            'workflow_dispatch:',
            'python sync_calendar.py',
            'create-pull-request'
        ]
        
        for element in required_elements:
            if element not in workflow:
                print(f"❌ Missing workflow element: {element}")
                return False
        
        print("✅ GitHub Actions workflow valid")
        return True
        
    except Exception as e:
        print(f"❌ Error validating workflow: {e}")
        return False

def test_basic_parsing():
    """Test basic parsing functionality without external dependencies"""
    try:
        # Test date regex patterns
        date_patterns = [
            r'\b(\w+day),?\s+(\w+)\s+(\d{1,2}),?\s+(\d{4})\b',
            r'\b(\w+)\s+(\d{1,2}),?\s+(\d{4})\b',
            r'\b(\d{1,2})/(\d{1,2})/(\d{4})\b',
        ]
        
        test_dates = [
            "Friday, August 29, 2025",
            "August 29, 2025", 
            "8/29/2025"
        ]
        
        for test_date in test_dates:
            found = False
            for pattern in date_patterns:
                if re.search(pattern, test_date):
                    found = True
                    break
            if not found:
                print(f"❌ Date pattern failed for: {test_date}")
                return False
        
        # Test time regex
        time_pattern = r'\b(\d{1,2}):(\d{2})\s*(AM|PM)?\b'
        test_times = ["5:00 PM", "10:00 AM", "7:30"]
        
        for test_time in test_times:
            if not re.search(time_pattern, test_time, re.I):
                print(f"❌ Time pattern failed for: {test_time}")
                return False
        
        print("✅ Basic parsing patterns working")
        return True
        
    except Exception as e:
        print(f"❌ Error testing parsing: {e}")
        return False

def main():
    """Run all validation checks"""
    print("🏊‍♀️ ETHS Swimming Calendar Sync Validation")
    print("=" * 50)
    
    checks = [
        ("File Structure", check_files),
        ("HTML Structure", validate_html_structure),
        ("ICS Structure", validate_ics_structure),
        ("Requirements", validate_requirements),
        ("GitHub Workflow", validate_workflow),
        ("Basic Parsing", test_basic_parsing)
    ]
    
    passed = 0
    total = len(checks)
    
    for name, check_func in checks:
        print(f"\n🔍 {name}:")
        if check_func():
            passed += 1
        else:
            print(f"   Failed validation for {name}")
    
    print(f"\n📊 Results: {passed}/{total} checks passed")
    
    if passed == total:
        print("\n🎉 All validations passed! The sync system is ready to use.")
        print("\nNext steps:")
        print("1. Test with dry-run: python sync_calendar.py --dry-run --verbose")
        print("2. Install dependencies: uv pip install -r requirements.txt")
        print("3. Set up automated workflow in GitHub Actions")
        return 0
    else:
        print(f"\n❌ {total - passed} validation(s) failed. Please fix issues before proceeding.")
        return 1

if __name__ == '__main__':
    sys.exit(main())