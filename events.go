// ABOUTME: Event struct and related utilities for swimming competition data
// ABOUTME: Handles serialization and validation of competition event information
package main

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
