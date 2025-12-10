package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Data Structures

type Event struct {
	Date        string  `json:"date"`
	Datetime    string  `json:"datetime"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	EventLink   string  `json:"event_link"`
	Venue       string  `json:"venue"`
	Address     string  `json:"address"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type MapboxResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates [2]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

type APIResponse struct {
	Events      []Event `json:"events"`
	MapboxToken string  `json:"mapbox_token"`
}

// Global Variables
var (
	eventsCache []Event
	cacheTime   time.Time
	mutex       sync.RWMutex
	dataFile    = "events.json"
)

// Helper Functions

func geocodeAddress(address string) (float64, float64, error) {
	accessToken := os.Getenv("MAPBOX_ACCESS_TOKEN")
	if accessToken == "" {
		return 0, 0, fmt.Errorf("MAPBOX_ACCESS_TOKEN not set")
	}

	baseURL := "https://api.mapbox.com/search/geocode/v6/forward"
	params := url.Values{}
	params.Add("q", address)
	params.Add("access_token", accessToken)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := http.Get(requestURL)
	if err != nil {
		return 0, 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	var result MapboxResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("error decoding json response: %v", err)
	}

	if len(result.Features) == 0 {
		return 0, 0, fmt.Errorf("number of features returned was zero")
	}

	longitude := result.Features[0].Geometry.Coordinates[0]
	latitude := result.Features[0].Geometry.Coordinates[1]

	return longitude, latitude, nil
}

func scrapeEvents() ([]Event, error) {
	log.Println("Scraping events from flagpole.com...")
	resp, err := http.Get("https://flagpole.com/events/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	var eventList []Event

	doc.Find(".tribe-common-g-row.tribe-events-calendar-list__event-row").Each(func(index int, event *goquery.Selection) {
		dateAttr, exists := event.Find("time.tribe-events-calendar-list__event-datetime").Attr("datetime")
		if !exists || !strings.HasPrefix(dateAttr, today) {
			return
		}

		datetime := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-datetime").Text())
		category := strings.TrimSpace(event.Find(".tribe-events-event-categories a").Text())
		title := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-title").Text())
		eventLink, _ := event.Find(".tribe-events-calendar-list__event-title-link").Attr("href")
		venue := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-venue-title").Text())
		address := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-venue-address").Text())
		description := strings.TrimSpace(event.Find(".tribe-events-calendar-list__event-description p").Text())

		longitude, latitude, err := geocodeAddress(address)
		if err != nil {
			log.Printf("Error geocoding address '%s': %v", address, err)
			// Keep going even if geocoding fails, maybe set to 0,0 or omit
			latitude = 0
			longitude = 0
		} else {
			// Small delay to be nice to the API if processing many
			time.Sleep(100 * time.Millisecond)
		}

		eventList = append(eventList, Event{
			Date:        dateAttr,
			Datetime:    datetime,
			Category:    category,
			Title:       title,
			EventLink:   eventLink,
			Venue:       venue,
			Address:     address,
			Description: description,
			Latitude:    latitude,
			Longitude:   longitude,
		})
	})
	
	log.Printf("Scraped %d events.", len(eventList))
	return eventList, nil
}

func saveEventsToFile(events []Event) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile, data, 0644)
}

func loadEventsFromFile() ([]Event, error) {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, err
	}
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func getEvents() ([]Event, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Check if we need to scrape (e.g., file doesn't exist or is old)
	// For simplicity, let's just check if it exists and scrape if not.
	// You might want to add logic to re-scrape daily.
	
	// If in-memory cache is empty, try loading from file
	if len(eventsCache) == 0 {
		if _, err := os.Stat(dataFile); err == nil {
			// File exists, load it
			events, err := loadEventsFromFile()
			if err == nil {
				eventsCache = events
				log.Println("Loaded events from local file.")
			}
		}
	}

	// If still empty (file didn't exist or error), scrape
	if len(eventsCache) == 0 {
		events, err := scrapeEvents()
		if err != nil {
			return nil, err
		}
		eventsCache = events
		if err := saveEventsToFile(events); err != nil {
			log.Printf("Warning: Failed to save events to file: %v", err)
		}
	}

	return eventsCache, nil
}

// HTTP Handlers

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, err := getEvents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Events:      events,
		MapboxToken: os.Getenv("MAPBOX_ACCESS_TOKEN"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS if running separately, harmless otherwise
	json.NewEncoder(w).Encode(response)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Serve static files
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// API endpoint
	http.HandleFunc("/api/events", apiHandler)

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
