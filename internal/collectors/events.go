package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// Package-level base URLs allow tests to override via httptest servers.
var gdeltBaseURL = "https://api.gdeltproject.org/api/v2/geo/geo"
var acledBaseURL = "https://api.acleddata.com/acled/read"

// gdeltResponse represents the GeoJSON response from GDELT's geo API.
type gdeltResponse struct {
	Features []gdeltFeature `json:"features"`
}

type gdeltFeature struct {
	Properties gdeltProperties `json:"properties"`
	Geometry   gdeltGeometry   `json:"geometry"`
}

type gdeltProperties struct {
	Name           string `json:"name"`
	HTML           string `json:"html"`
	URLPubTimeSeq  string `json:"urlpubtimeseq"`
}

type gdeltGeometry struct {
	Coordinates []float64 `json:"coordinates"`
}

// acledResponse represents the JSON response from the ACLED API.
type acledResponse struct {
	Data []acledEvent `json:"data"`
}

type acledEvent struct {
	EventIDCnty string `json:"event_id_cnty"`
	EventType   string `json:"event_type"`
	Notes       string `json:"notes"`
	Latitude    string `json:"latitude"`
	Longitude   string `json:"longitude"`
	EventDate   string `json:"event_date"`
	Source      string `json:"source"`
}

// fetchGDELT queries the GDELT geo API for military events and returns
// them as a slice of models.Event.
func fetchGDELT(ctx context.Context, baseURL string) ([]models.Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gdelt: create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("query", "military")
	q.Set("format", "GeoJSON")
	q.Set("maxrows", "50")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gdelt: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gdelt: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gdelt: read body: %w", err)
	}

	var gResp gdeltResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, fmt.Errorf("gdelt: decode JSON: %w", err)
	}

	events := make([]models.Event, 0, len(gResp.Features))
	for _, f := range gResp.Features {
		var lat, lon float64
		if len(f.Geometry.Coordinates) >= 2 {
			// GeoJSON coordinates are [lon, lat]
			lon = f.Geometry.Coordinates[0]
			lat = f.Geometry.Coordinates[1]
		}

		// Parse the GDELT timestamp (format: YYYYMMDDHHmmSS)
		var ts time.Time
		var dateStr string
		if len(f.Properties.URLPubTimeSeq) >= 8 {
			dateStr = f.Properties.URLPubTimeSeq[:4] + "-" +
				f.Properties.URLPubTimeSeq[4:6] + "-" +
				f.Properties.URLPubTimeSeq[6:8]
			parsed, err := time.Parse("20060102150405", f.Properties.URLPubTimeSeq)
			if err == nil {
				ts = parsed
			} else {
				ts = time.Now().UTC()
			}
		} else {
			ts = time.Now().UTC()
			dateStr = ts.Format("2006-01-02")
		}

		events = append(events, models.Event{
			ID:          fmt.Sprintf("gdelt-%s", f.Properties.URLPubTimeSeq),
			Type:        "military_event",
			Title:       f.Properties.Name,
			Description: f.Properties.Name,
			Lat:         lat,
			Lon:         lon,
			Source:      "gdelt",
			URL:         f.Properties.HTML,
			Date:        dateStr,
			Timestamp:   ts,
		})
	}

	return events, nil
}

// fetchACLED queries the ACLED API for battle/conflict events and returns
// them as a slice of models.Event. The baseURL parameter allows tests to
// inject a mock server URL.
func fetchACLED(ctx context.Context, apiKey string, baseURL string) ([]models.Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("acled: create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("key", apiKey)
	q.Set("event_type", "Battles")
	q.Set("limit", "50")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("acled: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("acled: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("acled: read body: %w", err)
	}

	var aResp acledResponse
	if err := json.Unmarshal(body, &aResp); err != nil {
		return nil, fmt.Errorf("acled: decode JSON: %w", err)
	}

	events := make([]models.Event, 0, len(aResp.Data))
	for _, d := range aResp.Data {
		lat, _ := strconv.ParseFloat(d.Latitude, 64)
		lon, _ := strconv.ParseFloat(d.Longitude, 64)

		var ts time.Time
		parsed, err := time.Parse("2006-01-02", d.EventDate)
		if err == nil {
			ts = parsed
		} else {
			ts = time.Now().UTC()
		}

		events = append(events, models.Event{
			ID:          d.EventIDCnty,
			Type:        d.EventType,
			Title:       d.EventType,
			Description: d.Notes,
			Lat:         lat,
			Lon:         lon,
			Source:      "acled",
			Date:        d.EventDate,
			Timestamp:   ts,
		})
	}

	return events, nil
}

// CollectEvents fetches military events from GDELT and ACLED in parallel.
// If acledKey is empty, ACLED is skipped silently. Failures from either
// source are logged as warnings but do not prevent returning results from
// the other source.
func CollectEvents(ctx context.Context, acledKey string) ([]models.Event, error) {
	var (
		mu          sync.Mutex
		allEvents   []models.Event
		wg          sync.WaitGroup
	)

	// Fetch GDELT
	wg.Add(1)
	go func() {
		defer wg.Done()
		events, err := fetchGDELT(ctx, gdeltBaseURL)
		if err != nil {
			log.Printf("WARNING: GDELT fetch failed: %v", err)
			return
		}
		mu.Lock()
		allEvents = append(allEvents, events...)
		mu.Unlock()
	}()

	// Fetch ACLED (skip if no key)
	if acledKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			events, err := fetchACLED(ctx, acledKey, acledBaseURL)
			if err != nil {
				log.Printf("WARNING: ACLED fetch failed: %v", err)
				return
			}
			mu.Lock()
			allEvents = append(allEvents, events...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	return allEvents, nil
}
