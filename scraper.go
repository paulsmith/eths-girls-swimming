// ABOUTME: Web scraper for Wildkits aquatics competition data
// ABOUTME: Scrapes competition details from wildkitaquatics.com and generates HTML calendar
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"golang.org/x/net/html"
)

const (
	baseURL          = "https://wildkitaquatics.com/main/EvanstonGirlsSwimming"
	userAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
	competitionsPath = "/competitions" // Adjust based on actual path
)

type Scraper struct {
	client *http.Client
}

func NewScraper() *Scraper {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &Scraper{client: client}
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
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
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
	response, err := s.callLLM(ctx, prompt)
	if err != nil {
		log.Printf("LLM call failed for %s, using fallback: %v", comp.RawName, err)
		return s.createFallbackEvent(comp), nil
	}

	// Parse JSON response
	var event Event
	if err := json.Unmarshal([]byte(response), &event); err != nil {
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
   - date: Format as "Day, Month Date, Year" (e.g., "Saturday, August 23, 2025"). Assume the next occurrence of the date if year is not provided.
   - time: Extract from details if available, otherwise use "TBD". Format as "HH:MM AM/PM CT".
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
     * "" (empty string) for regular dual meets
   - levels: Extract level information (e.g., JV, Varsity) if mentioned, otherwise leave as an empty string.
   - details: Include any additional important information, such as bus departure times.

3. Your final output must be valid JSON only, without any additional text or explanation.`, comp.RawDate, comp.RawName, comp.RawLocation, comp.Details)
}

func (s *Scraper) callLLM(ctx context.Context, prompt string) (string, error) {
	client := anthropic.NewClient()
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
		Model: anthropic.ModelClaude3_5HaikuLatest,
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
		TypeClass:    "",
		Levels:       "",
	}
}

func (s *Scraper) formatDate(rawDate string) string {
	// Simple date formatting - in production would use proper date parsing
	if strings.Contains(rawDate, ",") {
		return rawDate // Already formatted
	}
	return rawDate // Return as-is for now
}

func (s *Scraper) extractShortLocation(location string) string {
	if strings.Contains(strings.ToLower(location), "eths") {
		return "ETHS"
	}

	// Extract school name before comma or address
	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}

	return location
}

func (s *Scraper) isHome(location string) bool {
	lower := strings.ToLower(location)
	return strings.Contains(lower, "eths") || strings.Contains(lower, "evanston")
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

func (s *Scraper) getTypeClass(name string) string {
	eventType := s.classifyEventType(name)

	switch eventType {
	case "Meeting":
		return "meeting"
	case "Invitational":
		return "invitational"
	case "Relay Meet":
		return "relay"
	case "Conference":
		return "conference"
	case "Championship":
		return "championship"
	case "Special Event":
		return "special"
	default:
		return ""
	}
}

func (s *Scraper) extractLevels(name string) string {
	lower := strings.ToLower(name)

	var levels []string
	if strings.Contains(lower, "jv") {
		levels = append(levels, "JV")
	}
	if strings.Contains(lower, "varsity") {
		levels = append(levels, "Varsity")
	}
	if strings.Contains(lower, "frosh") {
		levels = append(levels, "Freshman")
	}
	if strings.Contains(lower, "diving") {
		levels = append(levels, "Diving")
	}
	if strings.Contains(lower, "jv2") {
		levels = append(levels, "JV2")
	}
	if strings.Contains(lower, "jv3") {
		levels = append(levels, "JV3")
	}

	if len(levels) > 0 {
		return strings.Join(levels, ", ")
	}

	return ""
}

func (s *Scraper) generateHTML(ctx context.Context, events []Event) error {
	log.Printf("Generating HTML with %d events", len(events))

	tmpl, err := template.ParseFiles("index.tmpl")
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

func (s *Scraper) Run(ctx context.Context) error {
	log.Printf("Starting scraper...")

	if err := s.establishSession(ctx); err != nil {
		return fmt.Errorf("failed to establish session: %w", err)
	}

	competitions, err := s.scrapeCompetitionsPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to scrape competitions: %w", err)
	}

	var events []Event
	for _, comp := range competitions {
		event, err := s.processWithLLM(ctx, comp)
		if err != nil {
			log.Printf("Failed to process competition %s: %v", comp.RawName, err)
			continue
		}
		events = append(events, *event)
	}

	if len(events) == 0 {
		return fmt.Errorf("no events found")
	}

	return s.generateHTML(ctx, events)
}

func main() {
	scraper := NewScraper()
	if err := scraper.Run(context.Background()); err != nil {
		log.Fatalf("Scraper failed: %v", err)
	}
}
