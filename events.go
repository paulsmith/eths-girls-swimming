// ABOUTME: Event struct and related utilities for swimming competition data
// ABOUTME: Handles serialization and validation of competition event information
package main

import (
	"fmt"
	"strings"
	"time"
)

// Event represents a swimming competition or meet
type Event struct {
	Date         string `json:"date"`          // "Saturday, August 23, 2025"
	Time         string `json:"time"`          // "7:00 PM"
	Name         string `json:"name"`          // "Meet Name"
	Location     string `json:"location"`      // "Short Location"
	LocationFull string `json:"location_full"` // "Full Address"
	IsHome       bool   `json:"is_home"`       // true for home events
	Type         string `json:"type"`          // "meeting", "meet", "relay", etc.
	Levels       string `json:"levels"`        // "JV, Varsity, etc" (optional)
	Details      string `json:"details"`       // Additional details or important info
}

// PageData represents the template data structure
type PageData struct {
	Events []Event `json:"events"`
}

// CompetitionInfo holds scraped data before LLM processing
type CompetitionInfo struct {
	RawDate     string
	RawName     string
	RawLocation string
	Details     map[string]string
}

// CalendarData represents the template data structure for ICS generation
type CalendarData struct {
	Events      []Event `json:"events"`
	CreatedTime string  `json:"created_time"` // UTC timestamp for DTSTAMP
}

// FormatStartTime converts event date/time to ICS format (YYYYMMDDTHHMMSSZ)
func (e *Event) FormatStartTime() string {
	return e.formatEventTime(false)
}

// FormatEndTime converts event date/time to ICS format, adding 1.5 hours to start time
func (e *Event) FormatEndTime() string {
	return e.formatEventTime(true)
}

// formatEventTime handles the common logic for formatting start/end times
func (e *Event) formatEventTime(isEndTime bool) string {
	// Parse the date
	parsedDate, err := time.Parse("Monday, January 2, 2006", e.Date)
	if err != nil {
		// Fallback parsing for different formats
		parsedDate, err = time.Parse("January 2, 2006", e.Date)
		if err != nil {
			// Use current year if parsing fails
			parsedDate = time.Now()
		}
	}

	// Parse the time
	eventTime := parsedDate
	if e.Time != "" && e.Time != "TBD" {
		timeStr := strings.Replace(e.Time, " CT", "", 1) // Remove CT timezone
		
		// Try different time formats
		for _, format := range []string{"3:04 PM", "15:04", "3:04PM", "15:04:05"} {
			if parsed, err := time.Parse(format, timeStr); err == nil {
				eventTime = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
					parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC)
				break
			}
		}
	} else {
		// Default to 9:30 PM UTC (2:30 PM CT) if no time specified
		eventTime = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
			21, 30, 0, 0, time.UTC)
	}

	// Add 1.5 hours for end time
	if isEndTime {
		eventTime = eventTime.Add(90 * time.Minute)
	}

	return eventTime.Format("20060102T150405Z")
}

// FormatUID generates a unique identifier for the event
func (e *Event) FormatUID(index int) string {
	// Extract date components for UID
	parsedDate, err := time.Parse("Monday, January 2, 2006", e.Date)
	if err != nil {
		parsedDate, _ = time.Parse("January 2, 2006", e.Date)
	}
	
	return fmt.Sprintf("%04d-%02d-%02d-%d", parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), index)
}
