package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// AircraftSource represents an ADS-B API endpoint to fetch military aircraft from.
type AircraftSource struct {
	Name string
	URL  string
}

// CollectedAircraft embeds models.Aircraft and adds a SeenAgo field used for
// deduplication (lower SeenAgo means the record was seen more recently).
type CollectedAircraft struct {
	models.Aircraft
	SeenAgo float64 `json:"-"`
}

// adsbResponse is the top-level JSON structure returned by the ADS-B v2 APIs.
type adsbResponse struct {
	Aircraft []adsbAircraft `json:"ac"`
	Total    int            `json:"total"`
	Now      float64        `json:"now"`
}

// adsbAircraft represents a single aircraft entry in the ADS-B API response.
// The alt_baro field is a json.RawMessage because it can be either a number
// or the string "ground".
type adsbAircraft struct {
	Hex    string          `json:"hex"`
	Flight string          `json:"flight"`
	Type   string          `json:"t"`
	Lat    float64         `json:"lat"`
	Lon    float64         `json:"lon"`
	AltBar json.RawMessage `json:"alt_baro"`
	Speed  float64         `json:"gs"`
	Track  float64         `json:"track"`
	Squawk string          `json:"squawk"`
	Seen   float64         `json:"seen"`
}

// DefaultAircraftSources returns the 3 default free ADS-B military endpoints.
func DefaultAircraftSources() []AircraftSource {
	return []AircraftSource{
		{Name: "airplanes.live", URL: "https://api.airplanes.live/v2/mil"},
		{Name: "adsb.one", URL: "https://api.adsb.one/v2/mil"},
		{Name: "adsb.lol", URL: "https://api.adsb.lol/v2/mil"},
	}
}

// CollectAircraft fetches military aircraft from all given sources in parallel.
// If a source fails, a warning is logged and collection continues with the
// remaining sources. The function returns the combined results from all
// successful sources.
func CollectAircraft(ctx context.Context, sources []AircraftSource) ([]CollectedAircraft, error) {
	type result struct {
		aircraft []CollectedAircraft
		err      error
		source   string
	}

	var wg sync.WaitGroup
	ch := make(chan result, len(sources))

	for _, src := range sources {
		wg.Add(1)
		go func(s AircraftSource) {
			defer wg.Done()
			aircraft, err := fetchSource(ctx, s)
			ch <- result{aircraft: aircraft, err: err, source: s.Name}
		}(src)
	}

	// Close channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(ch)
	}()

	var all []CollectedAircraft
	for res := range ch {
		if res.err != nil {
			log.Printf("WARNING: failed to fetch from %s: %v", res.source, res.err)
			continue
		}
		all = append(all, res.aircraft...)
	}

	return all, nil
}

// fetchSource fetches aircraft data from a single ADS-B API source.
func fetchSource(ctx context.Context, src AircraftSource) ([]CollectedAircraft, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", src.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source %s returned status %d", src.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", src.Name, err)
	}

	var adsbResp adsbResponse
	if err := json.Unmarshal(body, &adsbResp); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", src.Name, err)
	}

	now := time.Now().UTC()
	aircraft := make([]CollectedAircraft, 0, len(adsbResp.Aircraft))

	for _, ac := range adsbResp.Aircraft {
		altitude := parseAltBaro(ac.AltBar)

		collected := CollectedAircraft{
			Aircraft: models.Aircraft{
				Hex:       ac.Hex,
				Callsign:  strings.TrimSpace(ac.Flight),
				Type:      ac.Type,
				Lat:       ac.Lat,
				Lon:       ac.Lon,
				Altitude:  altitude,
				Speed:     ac.Speed,
				Heading:   ac.Track,
				Squawk:    ac.Squawk,
				Source:    src.Name,
				Timestamp: now,
			},
			SeenAgo: ac.Seen,
		}
		aircraft = append(aircraft, collected)
	}

	return aircraft, nil
}

// parseAltBaro parses the alt_baro field which can be a float64 or "ground".
func parseAltBaro(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}

	// Try parsing as a number first.
	var altitude float64
	if err := json.Unmarshal(raw, &altitude); err == nil {
		return int(altitude)
	}

	// Try parsing as a string (e.g., "ground").
	var altStr string
	if err := json.Unmarshal(raw, &altStr); err == nil {
		if altStr == "ground" {
			return 0
		}
	}

	return 0
}

// DeduplicateAircraft removes duplicate aircraft entries by ICAO hex code,
// keeping the record with the lowest SeenAgo value (most recently seen).
func DeduplicateAircraft(aircraft []CollectedAircraft) []CollectedAircraft {
	if len(aircraft) == 0 {
		return aircraft
	}

	best := make(map[string]CollectedAircraft)

	for _, ac := range aircraft {
		existing, ok := best[ac.Hex]
		if !ok || ac.SeenAgo < existing.SeenAgo {
			best[ac.Hex] = ac
		}
	}

	result := make([]CollectedAircraft, 0, len(best))
	for _, ac := range best {
		result = append(result, ac)
	}

	return result
}
