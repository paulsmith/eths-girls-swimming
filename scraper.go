// ABOUTME: Web scraper for Wildkits aquatics competition data
// ABOUTME: Scrapes competition details from wildkitaquatics.com and generates HTML calendar
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"golang.org/x/net/html"
)

const (
	baseURL       = "https://wildkitaquatics.com/main/EvanstonGirlsSwimming"
	userAgent     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
	currentSeason = "2025-2026"
	currentYear   = "2025"

	// File paths for intermediate data
	rawDataFile       = "scraped_data.json"
	processedDataFile = "processed_events.json"
)

// Pipeline controls execution phases
type Pipeline struct {
	StartFromPhase string
	StopAtPhase    string
}

// Config holds CLI configuration
type Config struct {
	Pipeline Pipeline
	Verbose  bool
}

type Scraper struct {
	client *http.Client
	config Config
}

func NewScraper(config Config) *Scraper {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &Scraper{client: client, config: config}
}

// saveRawData saves scraped competition data to JSON file
func (s *Scraper) saveRawData(competitions []CompetitionInfo) error {
	data, err := json.MarshalIndent(competitions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling raw data: %w", err)
	}

	if err := os.WriteFile(rawDataFile, data, 0644); err != nil {
		return fmt.Errorf("writing raw data file: %w", err)
	}

	log.Printf("Saved raw data to %s (%d competitions)", rawDataFile, len(competitions))
	return nil
}

// loadRawData loads scraped competition data from JSON file
func (s *Scraper) loadRawData() ([]CompetitionInfo, error) {
	data, err := os.ReadFile(rawDataFile)
	if err != nil {
		return nil, fmt.Errorf("reading raw data file: %w", err)
	}

	var competitions []CompetitionInfo
	if err := json.Unmarshal(data, &competitions); err != nil {
		return nil, fmt.Errorf("unmarshaling raw data: %w", err)
	}

	log.Printf("Loaded raw data from %s (%d competitions)", rawDataFile, len(competitions))
	return competitions, nil
}

// saveProcessedData saves processed events to JSON file
func (s *Scraper) saveProcessedData(events []Event) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling processed data: %w", err)
	}

	if err := os.WriteFile(processedDataFile, data, 0644); err != nil {
		return fmt.Errorf("writing processed data file: %w", err)
	}

	log.Printf("Saved processed data to %s (%d events)", processedDataFile, len(events))
	return nil
}

// loadProcessedData loads processed events from JSON file
func (s *Scraper) loadProcessedData() ([]Event, error) {
	data, err := os.ReadFile(processedDataFile)
	if err != nil {
		return nil, fmt.Errorf("reading processed data file: %w", err)
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("unmarshaling processed data: %w", err)
	}

	log.Printf("Loaded processed data from %s (%d events)", processedDataFile, len(events))
	return events, nil
}

func (s *Scraper) request(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return s.client.Do(req)
}

func (s *Scraper) establishSession(ctx context.Context) error {
	log.Printf("Establishing session with %s", baseURL)

	resp, err := s.request("GET", baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to GET main page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Printf("Session established, cookies: %d", len(s.client.Jar.Cookies(resp.Request.URL)))
	return nil
}

func (s *Scraper) findCompetitionsLink(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var competitionsURL string
	var findLink func(*html.Node)
	findLink = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.Contains(strings.ToLower(attr.Val), "competition") {
					competitionsURL = attr.Val
					return
				}
			}
			// Check link text
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				text := strings.ToLower(strings.TrimSpace(n.FirstChild.Data))
				if strings.Contains(text, "competition") {
					for _, attr := range n.Attr {
						if attr.Key == "href" {
							competitionsURL = attr.Val
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLink(c)
			if competitionsURL != "" {
				return
			}
		}
	}

	findLink(doc)
	if competitionsURL == "" {
		return "", fmt.Errorf("competitions link not found")
	}

	// Make URL absolute if needed
	if !strings.HasPrefix(competitionsURL, "http") {
		baseU, _ := url.Parse(baseURL)
		competitionsURL = baseU.ResolveReference(&url.URL{Path: competitionsURL}).String()
	}

	return competitionsURL, nil
}

func (s *Scraper) scrapeCompetitionsPage(ctx context.Context) ([]CompetitionInfo, error) {
	log.Printf("Scraping competitions page")

	// First get the main page to establish session
	resp, err := s.request("GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get main page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Find competitions link
	competitionsURL, err := s.findCompetitionsLink(string(body))
	if err != nil {
		log.Printf("Could not find competitions link, trying direct path")
		// Fallback to common paths
		baseU, _ := url.Parse(baseURL)
		competitionsURL = baseU.ResolveReference(&url.URL{Path: "/competitions"}).String()
	}

	log.Printf("Accessing competitions page: %s", competitionsURL)

	// Get competitions page
	resp, err = s.request("GET", competitionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get competitions page: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseCompetitionsTable(string(body))
}

func (s *Scraper) parseCompetitionsTable(htmlContent string) ([]CompetitionInfo, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var competitions []CompetitionInfo
	var parseTable func(*html.Node)

	parseTable = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			// Found a table, look for rows
			s.parseTableRows(n, &competitions)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseTable(c)
		}
	}

	parseTable(doc)

	log.Printf("Found %d competitions", len(competitions))
	return competitions, nil
}

func (s *Scraper) parseTableRows(table *html.Node, competitions *[]CompetitionInfo) {
	var processRow func(*html.Node)
	processRow = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			comp := s.extractCompetitionFromRow(n)
			if comp.RawName != "" {
				*competitions = append(*competitions, comp)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processRow(c)
		}
	}
	processRow(table)
}

func (s *Scraper) extractCompetitionFromRow(row *html.Node) CompetitionInfo {
	var cells []string
	var infoForm *FormData
	var extractCellsAndForms func(*html.Node)

	extractCellsAndForms = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td") {
			text := s.extractText(n)
			cells = append(cells, strings.TrimSpace(text))

			// Look for forms in this cell
			form := s.findFormInNode(n)
			if form != nil {
				infoForm = form
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractCellsAndForms(c)
		}
	}

	extractCellsAndForms(row)

	if len(cells) < 3 {
		return CompetitionInfo{}
	}

	comp := CompetitionInfo{
		RawDate:     cells[0],
		RawName:     cells[1],
		RawLocation: cells[2],
		Details:     make(map[string]string),
	}

	// Get detailed info if we found an info form
	if infoForm != nil {
		details, err := s.getCompetitionDetails(infoForm)
		if err != nil {
			log.Printf("Failed to get details for %s: %v", comp.RawName, err)
		} else {
			comp.Details = details
		}
	}

	return comp
}

type FormData struct {
	Action string
	Fields map[string]string
}

func (s *Scraper) findFormInNode(n *html.Node) *FormData {
	var findForm func(*html.Node) *FormData
	findForm = func(node *html.Node) *FormData {
		if node.Type == html.ElementNode && node.Data == "form" {
			form := &FormData{
				Fields: make(map[string]string),
			}

			// Get form action
			for _, attr := range node.Attr {
				if attr.Key == "action" {
					form.Action = attr.Val
					break
				}
			}

			// Get all input fields
			s.extractFormFields(node, form.Fields)
			return form
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if result := findForm(c); result != nil {
				return result
			}
		}
		return nil
	}

	return findForm(n)
}

func (s *Scraper) extractFormFields(form *html.Node, fields map[string]string) {
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				} else if attr.Key == "value" {
					value = attr.Val
				}
			}
			if name != "" {
				fields[name] = value
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(form)
}

func (s *Scraper) getCompetitionDetails(form *FormData) (map[string]string, error) {
	log.Printf("Getting competition details for ID %s", form.Fields["competition_id"])
	// Prepare form data
	data := url.Values{}
	for key, value := range form.Fields {
		data.Set(key, value)
	}

	// Make URL absolute
	var detailURL string
	if strings.HasPrefix(form.Action, "http") {
		detailURL = form.Action
	} else {
		baseU, _ := url.Parse(baseURL)
		detailURL = baseU.ResolveReference(&url.URL{Path: form.Action}).String()
	}

	// POST to get details
	resp, err := s.request("POST", detailURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseDetailsPage(string(body))
}

func (s *Scraper) parseDetailsPage(htmlContent string) (map[string]string, error) {
	details := make(map[string]string)

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	// Look for tables or key-value pairs
	s.extractKeyValuePairs(doc, details)

	return details, nil
}

func (s *Scraper) extractKeyValuePairs(n *html.Node, details map[string]string) {
	if n.Type == html.ElementNode && n.Data == "table" {
		// Look for table rows with key-value pairs
		s.extractTableKeyValues(n, details)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.extractKeyValuePairs(c, details)
	}
}

func (s *Scraper) extractTableKeyValues(table *html.Node, details map[string]string) {
	var processRow func(*html.Node)
	processRow = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			s.extractTableCells(n, &cells)

			if len(cells) == 2 {
				key := strings.TrimSpace(cells[0])
				value := strings.TrimSpace(cells[1])
				if key != "" && value != "" {
					details[key] = value
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			processRow(c)
		}
	}
	processRow(table)
}

func (s *Scraper) extractTableCells(row *html.Node, cells *[]string) {
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			*cells = append(*cells, s.extractText(n))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(row)
}

func (s *Scraper) extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(s.extractText(c))
	}
	return text.String()
}

func (s *Scraper) processWithLLM(ctx context.Context, comp CompetitionInfo) (*Event, error) {
	// Create prompt for Claude Haiku
	prompt := s.createPrompt(comp)

	// Call LLM API
	response, err := s.callLLM(ctx, prompt, "{")
	if err != nil {
		log.Printf("LLM call failed for %s, using fallback: %v", comp.RawName, err)
		return s.createFallbackEvent(comp), nil
	}

	// Parse JSON response
	var event Event
	if err := json.Unmarshal([]byte("{"+response), &event); err != nil {
		log.Printf("JSON parsing failed for %s, using fallback: %v", comp.RawName, err)
		return s.createFallbackEvent(comp), nil
	}

	log.Printf("Processed competition: %s", comp.RawName)
	return &event, nil
}

func (s *Scraper) createPrompt(comp CompetitionInfo) string {
	return fmt.Sprintf(`You are a data conversion specialist tasked with transforming swimming competition data into a structured JSON Event object. Your goal is to accurately process the provided information and generate a well-formatted JSON output.

Here is the raw data for the swimming competition:

<raw_date>
%s
</raw_date>

<raw_name>
%s
</raw_name>

<raw_location>
%s
</raw_location>

<raw_details>
%s
</raw_details>

Instructions:
1. Process the raw data to create a JSON Event object with the following structure:
   {
     "date": "<formatted date>",
     "time": "<time or 'TBD'>",
     "name": "<competition name>",
     "location": "<short location name>", 
     "location_full": "<full address if available>",
     "is_home": <true/false>,
     "type": "<meet type>",
     "levels": "<level info or empty string>",
     "details": "<additional details or important info>"
   }

2. Follow these specific guidelines for each field:
   - date: Format as "Day, Month Date, Year" (e.g., "Saturday, August 23, 2025"). Assume the next occurrence of the date if year is not provided. This is for the %s season. The year is currently %s.
   - time: Extract from details if available, otherwise use "TBD". Format as "HH:MM AM/PM CT". This is the start time of a meet or event.
   - name: Use the provided raw name.
   - location: Extract a short location name from the raw location.
   - location_full: Use the full address if available.
   - is_home: Set to true if the location is ETHS or Evanston, otherwise false.
   - type: Classify the event type based on the following criteria:
     * "invitational" for invites/tournaments
     * "relay" for relay meets
     * "meeting" for meetings/info sessions
     * "conference" for conference meets
     * "championship" for sectionals/state
     * "special" for special events like senior night
     * "meet" for regular dual meets
   - levels: Extract level information (e.g., JV, Varsity) if mentioned, otherwise leave as an empty string.
   - details: Include any additional important information, such as bus departure times.

3. Your final output must be valid JSON only, without any additional text or explanation.`, comp.RawDate, comp.RawName, comp.RawLocation, comp.Details, currentSeason, currentYear)
}

func (s *Scraper) callLLM(ctx context.Context, prompt string, assistantPrefill string) (string, error) {
	client := anthropic.NewClient()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}
	if assistantPrefill != "" {
		messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(assistantPrefill)))
	}
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 2048,
		Messages:  messages,
		Model:     anthropic.ModelClaude3_5HaikuLatest,
	})
	if err != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", err)
	}
	var result strings.Builder
	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			result.WriteString(block.Text)
		default:
			panic(fmt.Sprintf("unexpected block type %T: %v", block, block))
		}
	}
	return result.String(), nil
}

func (s *Scraper) createFallbackEvent(comp CompetitionInfo) *Event {
	return &Event{
		Date:         comp.RawDate,
		Time:         "TBD",
		Name:         comp.RawName,
		Location:     comp.RawLocation,
		LocationFull: comp.RawLocation,
		IsHome:       strings.Contains(strings.ToLower(comp.RawLocation), "eths") || strings.Contains(strings.ToLower(comp.RawLocation), "evanston"),
		Type:         "Meet",
		Levels:       "",
	}
}

func (s *Scraper) classifyEventType(name string) string {
	lower := strings.ToLower(name)

	if strings.Contains(lower, "meeting") {
		return "Meeting"
	}
	if strings.Contains(lower, "invite") || strings.Contains(lower, "invitational") {
		return "Invitational"
	}
	if strings.Contains(lower, "relay") {
		return "Relay Meet"
	}
	if strings.Contains(lower, "conference") || strings.Contains(lower, "csl") {
		return "Conference"
	}
	if strings.Contains(lower, "sectional") || strings.Contains(lower, "state") || strings.Contains(lower, "ihsa") {
		return "Championship"
	}
	if strings.Contains(lower, "senior") || strings.Contains(lower, "awards") {
		return "Special Event"
	}

	return "Dual Meet"
}

func (s *Scraper) generateHTML(ctx context.Context, events []Event) error {
	log.Printf("Generating HTML with %d events", len(events))

	tmpl, err := htmlTemplate.ParseFiles("index.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := PageData{Events: events}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	err = os.WriteFile("index.html", buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	log.Printf("HTML generated successfully: index.html")
	return nil
}

func (s *Scraper) generateCalendar(ctx context.Context, events []Event) error {
	log.Printf("Generating ICS calendar with %d events", len(events))

	tmpl, err := textTemplate.ParseFiles("calendar.ics.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse calendar template: %w", err)
	}

	data := CalendarData{
		Events:      events,
		CreatedTime: time.Now().UTC().Format("20060102T150405Z"),
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute calendar template: %w", err)
	}

	err = os.WriteFile("calendar.ics", buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write calendar file: %w", err)
	}

	log.Printf("Calendar generated successfully: calendar.ics")
	return nil
}

func (s *Scraper) Run(ctx context.Context) error {
	log.Printf("Starting scraper pipeline (from: %s, to: %s)",
		s.config.Pipeline.StartFromPhase, s.config.Pipeline.StopAtPhase)

	var competitions []CompetitionInfo
	var events []Event
	var err error

	// Phase 1: Scrape raw data
	if s.shouldRunPhase("scrape") {
		log.Printf("=== Phase 1: Scraping competition data ===")
		if err := s.establishSession(ctx); err != nil {
			return fmt.Errorf("failed to establish session: %w", err)
		}

		competitions, err = s.scrapeCompetitionsPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to scrape competitions: %w", err)
		}

		if err := s.saveRawData(competitions); err != nil {
			return fmt.Errorf("failed to save raw data: %w", err)
		}

		if s.config.Pipeline.StopAtPhase == "scrape" {
			log.Printf("Stopping after scrape phase as requested")
			return nil
		}
	}

	// Phase 2: Process with LLM
	if s.shouldRunPhase("process") {
		log.Printf("=== Phase 2: Processing with LLM ===")
		if competitions == nil {
			competitions, err = s.loadRawData()
			if err != nil {
				return fmt.Errorf("failed to load raw data: %w", err)
			}
		}

		for i, comp := range competitions {
			if s.config.Verbose {
				log.Printf("Processing %d/%d: %s", i+1, len(competitions), comp.RawName)
			}
			event, err := s.processWithLLM(ctx, comp)
			if err != nil {
				log.Printf("Failed to process competition %s: %v", comp.RawName, err)
				continue
			}
			events = append(events, *event)
		}

		if len(events) == 0 {
			return fmt.Errorf("no events processed successfully")
		}

		if err := s.saveProcessedData(events); err != nil {
			return fmt.Errorf("failed to save processed data: %w", err)
		}

		if s.config.Pipeline.StopAtPhase == "process" {
			log.Printf("Stopping after process phase as requested")
			return nil
		}
	}

	// Phase 3: Generate HTML
	if s.shouldRunPhase("html") {
		log.Printf("=== Phase 3: Generating HTML ===")
		if events == nil {
			events, err = s.loadProcessedData()
			if err != nil {
				return fmt.Errorf("failed to load processed data: %w", err)
			}
		}

		if err := s.generateHTML(ctx, events); err != nil {
			return err
		}

		if s.config.Pipeline.StopAtPhase == "html" {
			log.Printf("Stopping after HTML phase as requested")
			return nil
		}
	}

	// Phase 4: Generate Calendar
	if s.shouldRunPhase("calendar") {
		log.Printf("=== Phase 4: Generating Calendar ===")
		if events == nil {
			events, err = s.loadProcessedData()
			if err != nil {
				return fmt.Errorf("failed to load processed data: %w", err)
			}
		}

		if err := s.generateCalendar(ctx, events); err != nil {
			return err
		}
	}

	log.Printf("Pipeline completed successfully")
	return nil
}

// shouldRunPhase determines if a phase should be executed based on configuration
func (s *Scraper) shouldRunPhase(phase string) bool {
	phases := []string{"scrape", "process", "html", "calendar"}

	startIdx := 0
	endIdx := len(phases) - 1

	// Find start index
	if s.config.Pipeline.StartFromPhase != "" {
		for i, p := range phases {
			if p == s.config.Pipeline.StartFromPhase {
				startIdx = i
				break
			}
		}
	}

	// Find end index
	if s.config.Pipeline.StopAtPhase != "" {
		for i, p := range phases {
			if p == s.config.Pipeline.StopAtPhase {
				endIdx = i
				break
			}
		}
	}

	// Find current phase index
	currentIdx := -1
	for i, p := range phases {
		if p == phase {
			currentIdx = i
			break
		}
	}

	return currentIdx >= startIdx && currentIdx <= endIdx
}

func main() {
	var (
		startFrom = flag.String("from", "", "Start from phase: scrape, process, html, calendar")
		stopAt    = flag.String("to", "", "Stop at phase: scrape, process, html, calendar")
		verbose   = flag.Bool("v", false, "Verbose logging")
		help      = flag.Bool("h", false, "Show help")
	)

	flag.Usage = func() {
		prog := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, `ETHS Girls Swimming Calendar Scraper

Usage: %s [OPTIONS]

Pipeline Phases:
  scrape     - Scrape competition data from wildkitaquatics.com
  process    - Process scraped data with LLM (requires ANTHROPIC_API_KEY)
  html       - Generate HTML calendar from processed data
  calendar   - Generate ICS calendar file from processed data

Options:
`, prog)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  %s                            # Run full pipeline (scrape -> process -> html -> calendar)
  %s -from scrape -to scrape    # Only scrape data, save to scraped_data.json
  %s -from process              # Skip scraping, process existing data
  %s -from html                 # Only generate HTML from processed_events.json
  %s -from calendar             # Only generate ICS calendar from processed_events.json
  %s -v                         # Run with verbose logging

Files:
  scraped_data.json      # Raw scraped competition data
  processed_events.json  # LLM-processed event data  
  index.html             # Generated calendar HTML
  calendar.ics           # Generated ICS calendar file
`, prog, prog, prog, prog, prog, prog)
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	config := Config{
		Pipeline: Pipeline{
			StartFromPhase: *startFrom,
			StopAtPhase:    *stopAt,
		},
		Verbose: *verbose,
	}

	// Validate phase names
	validPhases := map[string]bool{"scrape": true, "process": true, "html": true, "calendar": true}
	if config.Pipeline.StartFromPhase != "" && !validPhases[config.Pipeline.StartFromPhase] {
		log.Fatalf("Invalid start phase: %s (must be: scrape, process, html, calendar)", config.Pipeline.StartFromPhase)
	}
	if config.Pipeline.StopAtPhase != "" && !validPhases[config.Pipeline.StopAtPhase] {
		log.Fatalf("Invalid stop phase: %s (must be: scrape, process, html, calendar)", config.Pipeline.StopAtPhase)
	}

	scraper := NewScraper(config)
	if err := scraper.Run(context.Background()); err != nil {
		log.Fatalf("Scraper failed: %v", err)
	}
}
