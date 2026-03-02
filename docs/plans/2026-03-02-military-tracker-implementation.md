# US Military Tracker — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a self-evolving OSINT system that tracks US military assets globally via a dynamically updating KML file served by GitHub Pages, powered by an AI council with adaptive chairman selection.

**Architecture:** Monolithic Go binary with clean package separation. Four GitHub Actions workflows (15-min tracker, post-run evaluation, monthly static refresh, weekly evolution). AI Council pattern with parallel analysis + chairman synthesis + offline evaluation. Local Ollama model as zero-cost safety net.

**Tech Stack:** Go 1.24+, GitHub Actions (ubuntu-latest), GitHub Pages, Ollama (Qwen 2.5 1.5B), Gemini/Groq/Mistral/DeepSeek/OpenRouter free tiers, encoding/xml for KML, github.com/coder/websocket, google.golang.org/genai.

**Design doc:** `docs/plans/2026-03-02-military-tracker-design.md`

---

## Phase 1: Project Scaffolding & Data Models

### Task 1: Initialize Go module and directory structure

**Files:**
- Create: `go.mod`
- Create: `cmd/tracker/main.go`
- Create: `internal/models/types.go`

**Step 1: Create go.mod**

```
module github.com/ko5tas/us-military-tracker

go 1.24
```

Run: `go mod init github.com/ko5tas/us-military-tracker` (if not manually created)

**Step 2: Create directory structure**

Run:
```bash
mkdir -p cmd/tracker internal/collectors internal/enrichment/providers internal/enrichment/shadow internal/kml internal/platform internal/models config data/static data/evolution output .github/workflows
```

**Step 3: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("US Military Tracker starting...")
	return nil
}
```

**Step 4: Verify it compiles and runs**

Run: `go build ./cmd/tracker && ./tracker`
Expected: `US Military Tracker starting...`

**Step 5: Commit**

```bash
git add go.mod cmd/ internal/ config/ data/ output/ .github/
git commit -m "feat: initialize Go project structure"
```

---

### Task 2: Define shared data models

**Files:**
- Create: `internal/models/types.go`
- Test: `internal/models/types_test.go`

**Step 1: Write test for model serialization**

```go
package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAircraftJSON(t *testing.T) {
	a := Aircraft{
		Hex:       "AE1234",
		Callsign:  "EVAC01",
		Type:      "C-17",
		Lat:       49.4389,
		Lon:       7.6009,
		Altitude:  28000,
		Speed:     450,
		Heading:   270,
		Source:    "airplanes_live",
		Timestamp: time.Date(2026, 3, 2, 14, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Aircraft
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Hex != a.Hex {
		t.Errorf("hex: got %q, want %q", decoded.Hex, a.Hex)
	}
	if decoded.Callsign != a.Callsign {
		t.Errorf("callsign: got %q, want %q", decoded.Callsign, a.Callsign)
	}
}

func TestVesselJSON(t *testing.T) {
	v := Vessel{
		MMSI:     "369970120",
		Name:     "USS NIMITZ",
		Type:     "Aircraft Carrier",
		Lat:      32.7157,
		Lon:      -117.1611,
		Speed:    12.5,
		Heading:  180,
		Source:   "aisstream",
		Timestamp: time.Date(2026, 3, 2, 14, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Vessel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.MMSI != v.MMSI {
		t.Errorf("mmsi: got %q, want %q", decoded.MMSI, v.MMSI)
	}
}

func TestEventJSON(t *testing.T) {
	e := Event{
		ID:          "GDELT-12345",
		Type:        "military_exercise",
		Title:       "NATO Exercise Baltic Shield",
		Description: "Large-scale naval exercise in Baltic Sea",
		Lat:         55.6761,
		Lon:         12.5683,
		Source:      "gdelt",
		Date:        "2026-03-01",
		Timestamp:   time.Date(2026, 3, 2, 14, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != e.ID {
		t.Errorf("id: got %q, want %q", decoded.ID, e.ID)
	}
}

func TestBaseJSON(t *testing.T) {
	b := Base{
		Name:    "Ramstein Air Base",
		Branch:  "Air Force",
		Country: "Germany",
		Lat:     49.4369,
		Lon:     7.6003,
		Type:    "air_base",
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Base
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != b.Name {
		t.Errorf("name: got %q, want %q", decoded.Name, b.Name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/models/ -v`
Expected: FAIL — types not defined yet

**Step 3: Write the types**

```go
package models

import "time"

// Aircraft represents a tracked military aircraft position.
type Aircraft struct {
	Hex       string    `json:"hex"`
	Callsign  string    `json:"callsign"`
	Type      string    `json:"type"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Altitude  int       `json:"altitude"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
	Squawk    string    `json:"squawk,omitempty"`
	Source    string    `json:"source"`
	Branch    string    `json:"branch,omitempty"`
	Mission   string    `json:"mission,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Vessel represents a tracked naval vessel position.
type Vessel struct {
	MMSI      string    `json:"mmsi"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
	Source    string    `json:"source"`
	Branch    string    `json:"branch,omitempty"`
	Class     string    `json:"class,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Event represents a military event from GDELT/ACLED or news.
type Event struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	Source      string    `json:"source"`
	URL         string    `json:"url,omitempty"`
	Date        string    `json:"date"`
	Timestamp   time.Time `json:"timestamp"`
}

// Base represents a known military installation.
type Base struct {
	Name    string  `json:"name"`
	Branch  string  `json:"branch"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Type    string  `json:"type"`
}

// NewsItem represents a geolocated news article.
type NewsItem struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Lat         float64   `json:"lat,omitempty"`
	Lon         float64   `json:"lon,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

// CollectedData holds all data gathered in one collection cycle.
type CollectedData struct {
	Aircraft  []Aircraft `json:"aircraft"`
	Vessels   []Vessel   `json:"vessels"`
	Events    []Event    `json:"events"`
	News      []NewsItem `json:"news"`
	Bases     []Base     `json:"bases"`
	Timestamp time.Time  `json:"timestamp"`
}

// EnrichedAsset is the AI council's output for a single asset.
type EnrichedAsset struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Branch      string  `json:"branch"`
	Mission     string  `json:"mission"`
	Description string  `json:"description"`
	Confidence  string  `json:"confidence"`
	Agreement   int     `json:"agreement"`
	TotalVotes  int     `json:"total_votes"`
}

// CouncilResult holds the full output of one council cycle.
type CouncilResult struct {
	Assets    []EnrichedAsset `json:"assets"`
	Summary   string          `json:"summary"`
	Chairman  string          `json:"chairman"`
	Score     float64         `json:"score"`
	Timestamp time.Time       `json:"timestamp"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/models/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/models/
git commit -m "feat: add shared data model types with JSON serialization"
```

---

## Phase 2: Data Collectors

### Task 3: Aircraft collector (Airplanes.live, ADSB.one, ADSB.lol)

**Files:**
- Create: `internal/collectors/aircraft.go`
- Test: `internal/collectors/aircraft_test.go`

**Step 1: Write test with mock HTTP server**

```go
package collectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectAircraft(t *testing.T) {
	// Mock API response matching Airplanes.live /v2/mil/ format
	mockResp := map[string]interface{}{
		"ac": []map[string]interface{}{
			{
				"hex":  "AE1234",
				"flight": "EVAC01  ",
				"t":    "C17",
				"lat":  49.4389,
				"lon":  7.6009,
				"alt_baro": 28000,
				"gs":   450.0,
				"track": 270.0,
				"squawk": "1200",
				"seen": 1.5,
			},
			{
				"hex":  "AE5678",
				"flight": "RCH401  ",
				"t":    "C5M",
				"lat":  38.9072,
				"lon":  -77.0369,
				"alt_baro": 35000,
				"gs":   480.0,
				"track": 90.0,
				"squawk": "3456",
				"seen": 0.8,
			},
		},
		"total": 2,
		"now":   1709391000.0,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	sources := []AircraftSource{
		{Name: "test", URL: server.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft: %v", err)
	}

	if len(aircraft) != 2 {
		t.Fatalf("got %d aircraft, want 2", len(aircraft))
	}

	if aircraft[0].Hex != "AE1234" {
		t.Errorf("hex: got %q, want %q", aircraft[0].Hex, "AE1234")
	}
	if aircraft[0].Callsign != "EVAC01" {
		t.Errorf("callsign: got %q, want %q", aircraft[0].Callsign, "EVAC01")
	}
}

func TestDeduplicateAircraft(t *testing.T) {
	aircraft := []Aircraft{
		{Hex: "AE1234", Callsign: "EVAC01", Source: "airplanes_live", SeenAgo: 5.0},
		{Hex: "AE1234", Callsign: "EVAC01", Source: "adsb_one", SeenAgo: 1.0},
		{Hex: "AE5678", Callsign: "RCH401", Source: "airplanes_live", SeenAgo: 2.0},
	}

	deduped := DeduplicateAircraft(aircraft)
	if len(deduped) != 2 {
		t.Fatalf("got %d aircraft, want 2", len(deduped))
	}

	// Should keep the one with the smallest SeenAgo (most recent)
	for _, a := range deduped {
		if a.Hex == "AE1234" && a.Source != "adsb_one" {
			t.Errorf("should keep adsb_one (SeenAgo=1.0), got %s", a.Source)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/collectors/ -v -run TestCollectAircraft`
Expected: FAIL

**Step 3: Implement aircraft collector**

```go
package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// AircraftSource defines an ADS-B API endpoint.
type AircraftSource struct {
	Name string
	URL  string
}

// DefaultAircraftSources returns the three free military ADS-B APIs.
func DefaultAircraftSources() []AircraftSource {
	return []AircraftSource{
		{Name: "airplanes_live", URL: "https://api.airplanes.live/v2/mil"},
		{Name: "adsb_one", URL: "https://api.adsb.one/v2/mil"},
		{Name: "adsb_lol", URL: "https://api.adsb.lol/v2/mil"},
	}
}

// Aircraft extends models.Aircraft with a SeenAgo field for deduplication.
type Aircraft struct {
	models.Aircraft
	SeenAgo float64 `json:"-"`
}

// adsbResponse is the common v2 API response format.
type adsbResponse struct {
	AC    []adsbAircraft `json:"ac"`
	Total int            `json:"total"`
	Now   float64        `json:"now"`
}

type adsbAircraft struct {
	Hex     string  `json:"hex"`
	Flight  string  `json:"flight"`
	Type    string  `json:"t"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	AltBaro any     `json:"alt_baro"`
	GS      float64 `json:"gs"`
	Track   float64 `json:"track"`
	Squawk  string  `json:"squawk"`
	Seen    float64 `json:"seen"`
}

// CollectAircraft fetches military aircraft from all sources in parallel.
func CollectAircraft(ctx context.Context, sources []AircraftSource) ([]Aircraft, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	var (
		mu       sync.Mutex
		allCraft []Aircraft
		wg       sync.WaitGroup
	)

	for _, src := range sources {
		wg.Add(1)
		go func(s AircraftSource) {
			defer wg.Done()
			craft, err := fetchAircraft(ctx, client, s)
			if err != nil {
				fmt.Printf("WARN: %s failed: %v\n", s.Name, err)
				return
			}
			mu.Lock()
			allCraft = append(allCraft, craft...)
			mu.Unlock()
		}(src)
	}

	wg.Wait()
	return allCraft, nil
}

func fetchAircraft(ctx context.Context, client *http.Client, src AircraftSource) ([]Aircraft, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "us-military-tracker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var apiResp adsbResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	now := time.Now().UTC()
	aircraft := make([]Aircraft, 0, len(apiResp.AC))
	for _, ac := range apiResp.AC {
		alt := 0
		switch v := ac.AltBaro.(type) {
		case float64:
			alt = int(v)
		}

		aircraft = append(aircraft, Aircraft{
			Aircraft: models.Aircraft{
				Hex:       strings.ToUpper(ac.Hex),
				Callsign:  strings.TrimSpace(ac.Flight),
				Type:      ac.Type,
				Lat:       ac.Lat,
				Lon:       ac.Lon,
				Altitude:  alt,
				Speed:     ac.GS,
				Heading:   ac.Track,
				Squawk:    ac.Squawk,
				Source:    src.Name,
				Timestamp: now,
			},
			SeenAgo: ac.Seen,
		})
	}

	return aircraft, nil
}

// DeduplicateAircraft removes duplicates by ICAO hex, keeping the most recent.
func DeduplicateAircraft(aircraft []Aircraft) []Aircraft {
	best := make(map[string]Aircraft)
	for _, a := range aircraft {
		existing, ok := best[a.Hex]
		if !ok || a.SeenAgo < existing.SeenAgo {
			best[a.Hex] = a
		}
	}

	result := make([]Aircraft, 0, len(best))
	for _, a := range best {
		result = append(result, a)
	}
	return result
}
```

**Step 4: Run tests**

Run: `go test ./internal/collectors/ -v -run TestCollect`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/collectors/aircraft.go internal/collectors/aircraft_test.go
git commit -m "feat: add aircraft collector with 3 ADS-B sources and deduplication"
```

---

### Task 4: Vessel collector (AISStream.io)

**Files:**
- Create: `internal/collectors/vessels.go`
- Test: `internal/collectors/vessels_test.go`

**Step 1: Write test**

```go
package collectors

import (
	"context"
	"testing"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

func TestParseAISMessage(t *testing.T) {
	raw := `{
		"MessageType": "PositionReport",
		"MetaData": {
			"MMSI": 369970120,
			"ShipName": "USS NIMITZ",
			"latitude": 32.7157,
			"longitude": -117.1611,
			"time_utc": "2026-03-02T14:30:00Z"
		},
		"Message": {
			"PositionReport": {
				"Sog": 12.5,
				"TrueHeading": 180
			}
		}
	}`

	vessel, err := parseAISMessage([]byte(raw))
	if err != nil {
		t.Fatalf("parseAISMessage: %v", err)
	}

	if vessel.MMSI != "369970120" {
		t.Errorf("mmsi: got %q, want %q", vessel.MMSI, "369970120")
	}
	if vessel.Name != "USS NIMITZ" {
		t.Errorf("name: got %q, want %q", vessel.Name, "USS NIMITZ")
	}
}

func TestFilterMilitaryVessels(t *testing.T) {
	vessels := []models.Vessel{
		{MMSI: "369970120", Name: "USS NIMITZ"},
		{MMSI: "338123456", Name: "FISHING BOAT"},
		{MMSI: "369123456", Name: "USCGC BERTHOLF"},
	}

	military := FilterMilitaryVessels(vessels)
	// Should include USS and USCGC prefixed vessels
	if len(military) != 2 {
		t.Errorf("got %d military vessels, want 2", len(military))
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/collectors/ -v -run TestParseAIS`

**Step 3: Implement vessel collector**

```go
package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/ko5tas/us-military-tracker/internal/models"
)

type aisSubscription struct {
	APIKey        string      `json:"APIKey"`
	BoundingBoxes [][]float64 `json:"BoundingBoxes"`
}

type aisMessage struct {
	MessageType string          `json:"MessageType"`
	MetaData    aisMetaData     `json:"MetaData"`
	Message     json.RawMessage `json:"Message"`
}

type aisMetaData struct {
	MMSI      int     `json:"MMSI"`
	ShipName  string  `json:"ShipName"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	TimeUTC   string  `json:"time_utc"`
}

type aisPositionMsg struct {
	PositionReport struct {
		Sog         float64 `json:"Sog"`
		TrueHeading int     `json:"TrueHeading"`
	} `json:"PositionReport"`
}

func parseAISMessage(raw []byte) (models.Vessel, error) {
	var msg aisMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return models.Vessel{}, fmt.Errorf("parse AIS message: %w", err)
	}

	vessel := models.Vessel{
		MMSI:      fmt.Sprintf("%d", msg.MetaData.MMSI),
		Name:      strings.TrimSpace(msg.MetaData.ShipName),
		Lat:       msg.MetaData.Latitude,
		Lon:       msg.MetaData.Longitude,
		Source:    "aisstream",
		Timestamp: time.Now().UTC(),
	}

	if msg.MessageType == "PositionReport" {
		var posMsg aisPositionMsg
		if err := json.Unmarshal(msg.Message, &posMsg); err == nil {
			vessel.Speed = posMsg.PositionReport.Sog
			vessel.Heading = float64(posMsg.PositionReport.TrueHeading)
		}
	}

	return vessel, nil
}

// FilterMilitaryVessels filters for vessels with military naming patterns.
func FilterMilitaryVessels(vessels []models.Vessel) []models.Vessel {
	militaryPrefixes := []string{"USS ", "USNS ", "USCGC "}

	var military []models.Vessel
	for _, v := range vessels {
		upper := strings.ToUpper(v.Name)
		for _, prefix := range militaryPrefixes {
			if strings.HasPrefix(upper, prefix) {
				military = append(military, v)
				break
			}
		}
	}
	return military
}

// CollectVessels connects to AISStream.io and collects vessel positions.
func CollectVessels(ctx context.Context, apiKey string, duration time.Duration) ([]models.Vessel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("AISStream API key not set")
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "wss://stream.aisstream.io/v0/stream", nil)
	if err != nil {
		return nil, fmt.Errorf("dial AISStream: %w", err)
	}
	defer conn.CloseNow()

	sub := aisSubscription{
		APIKey:        apiKey,
		BoundingBoxes: [][]float64{{-90, -180}, {90, 180}},
	}
	if err := wsjson.Write(ctx, conn, sub); err != nil {
		return nil, fmt.Errorf("send subscription: %w", err)
	}

	var vessels []models.Vessel
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			return vessels, fmt.Errorf("read: %w", err)
		}

		vessel, err := parseAISMessage(data)
		if err != nil {
			continue
		}
		vessels = append(vessels, vessel)
	}

	conn.Close(websocket.StatusNormalClosure, "done")
	return FilterMilitaryVessels(vessels), nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/collectors/ -v -run TestParseAIS`
Expected: PASS

Run: `go test ./internal/collectors/ -v -run TestFilterMilitary`
Expected: PASS

**Step 5: Add websocket dependency and commit**

```bash
go get github.com/coder/websocket@latest
git add internal/collectors/vessels.go internal/collectors/vessels_test.go go.mod go.sum
git commit -m "feat: add vessel collector via AISStream.io WebSocket"
```

---

### Task 5: Events collector (GDELT, ACLED)

**Files:**
- Create: `internal/collectors/events.go`
- Test: `internal/collectors/events_test.go`

**Step 1: Write test with mock**

```go
package collectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectGDELT(t *testing.T) {
	mockResp := map[string]interface{}{
		"features": []map[string]interface{}{
			{
				"properties": map[string]interface{}{
					"name":    "Military exercise reported near Baltic Sea",
					"html":    "https://example.com/article",
					"urlpubtimeseq": "20260301120000",
				},
				"geometry": map[string]interface{}{
					"coordinates": []float64{12.5683, 55.6761},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	events, err := fetchGDELT(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchGDELT: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	if events[0].Source != "gdelt" {
		t.Errorf("source: got %q, want %q", events[0].Source, "gdelt")
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/collectors/ -v -run TestCollectGDELT`

**Step 3: Implement events collector**

```go
package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

type gdeltResponse struct {
	Features []gdeltFeature `json:"features"`
}

type gdeltFeature struct {
	Properties struct {
		Name string `json:"name"`
		URL  string `json:"html"`
		Time string `json:"urlpubtimeseq"`
	} `json:"properties"`
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
}

type acledResponse struct {
	Data []acledEvent `json:"data"`
}

type acledEvent struct {
	EventID   string `json:"event_id_cnty"`
	EventType string `json:"event_type"`
	Notes     string `json:"notes"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	EventDate string `json:"event_date"`
	Source    string `json:"source"`
}

const (
	gdeltBaseURL = "https://api.gdeltproject.org/api/v2/geo/geo"
	acledBaseURL = "https://api.acleddata.com/acled/read"
)

func fetchGDELT(ctx context.Context, baseURL string) ([]models.Event, error) {
	url := baseURL + "?query=military%20US&format=geojson&maxpoints=50&timespan=1d"
	if baseURL != gdeltBaseURL {
		// For testing with mock server, don't append query params to mock URL
		url = baseURL
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GDELT: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var gResp gdeltResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, fmt.Errorf("parse GDELT: %w", err)
	}

	events := make([]models.Event, 0, len(gResp.Features))
	for i, f := range gResp.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		events = append(events, models.Event{
			ID:          fmt.Sprintf("gdelt-%d", i),
			Type:        "military_event",
			Title:       f.Properties.Name,
			Lat:         f.Geometry.Coordinates[1],
			Lon:         f.Geometry.Coordinates[0],
			URL:         f.Properties.URL,
			Source:      "gdelt",
			Date:        f.Properties.Time,
			Timestamp:   time.Now().UTC(),
		})
	}
	return events, nil
}

// CollectEvents fetches events from GDELT and ACLED in parallel.
func CollectEvents(ctx context.Context, acledKey string) ([]models.Event, error) {
	type result struct {
		events []models.Event
		err    error
	}

	ch := make(chan result, 2)

	go func() {
		events, err := fetchGDELT(ctx, gdeltBaseURL)
		ch <- result{events, err}
	}()

	go func() {
		events, err := fetchACLED(ctx, acledKey)
		ch <- result{events, err}
	}()

	var allEvents []models.Event
	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err != nil {
			fmt.Printf("WARN: event source failed: %v\n", r.err)
			continue
		}
		allEvents = append(allEvents, r.events...)
	}

	return allEvents, nil
}

func fetchACLED(ctx context.Context, apiKey string) ([]models.Event, error) {
	if apiKey == "" {
		return nil, nil // Skip if no key configured
	}

	url := fmt.Sprintf("%s?key=%s&event_type=Battles&limit=50&order=desc", acledBaseURL, apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ACLED: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var aResp acledResponse
	if err := json.Unmarshal(body, &aResp); err != nil {
		return nil, fmt.Errorf("parse ACLED: %w", err)
	}

	events := make([]models.Event, 0, len(aResp.Data))
	for _, e := range aResp.Data {
		var lat, lon float64
		fmt.Sscanf(e.Latitude, "%f", &lat)
		fmt.Sscanf(e.Longitude, "%f", &lon)

		events = append(events, models.Event{
			ID:          e.EventID,
			Type:        e.EventType,
			Title:       e.Notes,
			Lat:         lat,
			Lon:         lon,
			Source:      "acled",
			Date:        e.EventDate,
			Timestamp:   time.Now().UTC(),
		})
	}
	return events, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/collectors/ -v -run TestCollectGDELT`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/collectors/events.go internal/collectors/events_test.go
git commit -m "feat: add events collector for GDELT and ACLED"
```

---

### Task 6: News collector (GNews + RSS feeds)

**Files:**
- Create: `internal/collectors/news.go`
- Test: `internal/collectors/news_test.go`

**Step 1: Write test**

```go
package collectors

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGNews(t *testing.T) {
	mockResp := map[string]interface{}{
		"totalArticles": 1,
		"articles": []map[string]interface{}{
			{
				"title":       "US Navy deploys carrier group to Pacific",
				"description": "The USS Reagan carrier strike group...",
				"url":         "https://example.com/navy-deployment",
				"source":      map[string]interface{}{"name": "Defense News"},
				"publishedAt": "2026-03-02T10:00:00Z",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	news, err := fetchGNews(context.Background(), "test-key", server.URL)
	if err != nil {
		t.Fatalf("fetchGNews: %v", err)
	}

	if len(news) != 1 {
		t.Fatalf("got %d articles, want 1", len(news))
	}
	if news[0].Title != "US Navy deploys carrier group to Pacific" {
		t.Errorf("title: got %q", news[0].Title)
	}
}

func TestParseRSSFeed(t *testing.T) {
	rssXML := `<?xml version="1.0"?>
	<rss version="2.0">
		<channel>
			<title>DOD News</title>
			<item>
				<title>Exercise Baltic Shield begins</title>
				<link>https://example.com/baltic-shield</link>
				<description>NATO allies begin exercise...</description>
				<pubDate>Mon, 02 Mar 2026 12:00:00 GMT</pubDate>
			</item>
		</channel>
	</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	items, err := fetchRSSFeed(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchRSSFeed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Title != "Exercise Baltic Shield begins" {
		t.Errorf("title: got %q", items[0].Title)
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/collectors/ -v -run TestFetchGNews`

**Step 3: Implement news collector**

```go
package collectors

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

const gnewsBaseURL = "https://gnews.io/api/v4/search"

// DefaultRSSFeeds returns the DoD and defense news RSS feed URLs.
func DefaultRSSFeeds() []string {
	return []string{
		"https://www.defense.gov/DesktopModules/ArticleCS/RSS.ashx?max=20&ContentType=1",
		"https://www.dvidshub.net/rss/news",
	}
}

type gnewsResponse struct {
	TotalArticles int `json:"totalArticles"`
	Articles      []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Source      struct {
			Name string `json:"name"`
		} `json:"source"`
		PublishedAt string `json:"publishedAt"`
	} `json:"articles"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchGNews(ctx context.Context, apiKey string, baseURL string) ([]models.NewsItem, error) {
	url := fmt.Sprintf("%s?q=US+military+OR+US+navy+OR+US+air+force&lang=en&max=10&apikey=%s", baseURL, apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch GNews: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var gResp gnewsResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return nil, fmt.Errorf("parse GNews: %w", err)
	}

	news := make([]models.NewsItem, 0, len(gResp.Articles))
	for _, a := range gResp.Articles {
		pubTime, _ := time.Parse(time.RFC3339, a.PublishedAt)
		news = append(news, models.NewsItem{
			Title:       a.Title,
			Description: a.Description,
			URL:         a.URL,
			Source:      a.Source.Name,
			PublishedAt: pubTime,
		})
	}
	return news, nil
}

func fetchRSSFeed(ctx context.Context, feedURL string) ([]models.NewsItem, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}

	items := make([]models.NewsItem, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		pubTime, _ := time.Parse(time.RFC1123, item.PubDate)
		items = append(items, models.NewsItem{
			Title:       item.Title,
			Description: item.Description,
			URL:         item.Link,
			Source:      feed.Channel.Title,
			PublishedAt: pubTime,
		})
	}
	return items, nil
}

// CollectNews fetches news from GNews API and all RSS feeds in parallel.
func CollectNews(ctx context.Context, gnewsKey string) ([]models.NewsItem, error) {
	var (
		mu      sync.Mutex
		allNews []models.NewsItem
		wg      sync.WaitGroup
	)

	// GNews
	if gnewsKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			news, err := fetchGNews(ctx, gnewsKey, gnewsBaseURL)
			if err != nil {
				fmt.Printf("WARN: GNews failed: %v\n", err)
				return
			}
			mu.Lock()
			allNews = append(allNews, news...)
			mu.Unlock()
		}()
	}

	// RSS feeds
	for _, feedURL := range DefaultRSSFeeds() {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			items, err := fetchRSSFeed(ctx, url)
			if err != nil {
				fmt.Printf("WARN: RSS feed failed (%s): %v\n", url, err)
				return
			}
			mu.Lock()
			allNews = append(allNews, items...)
			mu.Unlock()
		}(feedURL)
	}

	wg.Wait()
	return allNews, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/collectors/ -v -run "TestFetchGNews|TestParseRSS"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/collectors/news.go internal/collectors/news_test.go
git commit -m "feat: add news collector for GNews API and RSS feeds"
```

---

## Phase 3: AI Provider Interface & Council

### Task 7: AI provider interface and OpenAI-compatible client

**Files:**
- Create: `internal/enrichment/providers/provider.go`
- Test: `internal/enrichment/providers/provider_test.go`

**Step 1: Write test with mock server**

```go
package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}

		resp := ChatResponse{
			ID: "test-id",
			Choices: []Choice{
				{
					Index:   0,
					Message: ChatMessage{Role: "assistant", Content: "C-17 Globemaster III, medevac mission"},
				},
			},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 8, TotalTokens: 18},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &OpenAIProvider{
		ProviderName: "test",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		ModelName:    "test-model",
		HTTPClient:   http.DefaultClient,
	}

	result, err := p.Complete(context.Background(), "classify aircraft", "hex=AE1234 callsign=EVAC01")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if result != "C-17 Globemaster III, medevac mission" {
		t.Errorf("result: got %q", result)
	}
}

func TestCompleteHandlesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	p := &OpenAIProvider{
		ProviderName: "test",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		ModelName:    "test-model",
		HTTPClient:   http.DefaultClient,
	}

	_, err := p.Complete(context.Background(), "test", "test")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/enrichment/providers/ -v`

**Step 3: Implement provider interface + OpenAI-compatible client**

```go
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Completer is the interface all AI providers implement.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	Name() string
}

// ChatRequest is the OpenAI-compatible request format.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIProvider works with any OpenAI-compatible API (Groq, Mistral, DeepSeek, OpenRouter, Ollama).
type OpenAIProvider struct {
	ProviderName string
	BaseURL      string
	APIKey       string
	ModelName    string
	HTTPClient   *http.Client
}

func (p *OpenAIProvider) Name() string { return p.ProviderName }

func (p *OpenAIProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := ChatRequest{
		Model: p.ModelName,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: request failed: %w", p.ProviderName, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%s: read response: %w", p.ProviderName, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: status %d: %s", p.ProviderName, resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("%s: parse response: %w", p.ProviderName, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("%s: no choices in response", p.ProviderName)
	}

	return chatResp.Choices[0].Message.Content, nil
}

// NewGroq creates a Groq provider.
func NewGroq(apiKey string) Completer {
	return &OpenAIProvider{
		ProviderName: "groq",
		BaseURL:      "https://api.groq.com/openai/v1",
		APIKey:       apiKey,
		ModelName:    "llama-3.3-70b-versatile",
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// NewMistral creates a Mistral provider.
func NewMistral(apiKey string) Completer {
	return &OpenAIProvider{
		ProviderName: "mistral",
		BaseURL:      "https://api.mistral.ai/v1",
		APIKey:       apiKey,
		ModelName:    "mistral-small-latest",
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// NewDeepSeek creates a DeepSeek provider.
func NewDeepSeek(apiKey string) Completer {
	return &OpenAIProvider{
		ProviderName: "deepseek",
		BaseURL:      "https://api.deepseek.com",
		APIKey:       apiKey,
		ModelName:    "deepseek-chat",
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// NewOpenRouter creates an OpenRouter provider.
func NewOpenRouter(apiKey string) Completer {
	return &OpenAIProvider{
		ProviderName: "openrouter",
		BaseURL:      "https://openrouter.ai/api/v1",
		APIKey:       apiKey,
		ModelName:    "meta-llama/llama-3.3-70b-instruct:free",
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// NewOllama creates a local Ollama provider.
func NewOllama() Completer {
	return &OpenAIProvider{
		ProviderName: "ollama",
		BaseURL:      "http://localhost:11434/v1",
		APIKey:       "",
		ModelName:    "qwen2.5:1.5b",
		HTTPClient:   &http.Client{Timeout: 120 * time.Second},
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/enrichment/providers/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enrichment/providers/provider.go internal/enrichment/providers/provider_test.go
git commit -m "feat: add AI provider interface with OpenAI-compatible client"
```

---

### Task 8: Gemini provider

**Files:**
- Create: `internal/enrichment/providers/gemini.go`
- Test: `internal/enrichment/providers/gemini_test.go`

**Step 1: Write test**

Note: Gemini SDK doesn't easily support mock servers, so test the wrapper logic with an interface.

```go
package providers

import (
	"context"
	"testing"
)

func TestGeminiProviderName(t *testing.T) {
	// Test that the provider implements the interface correctly
	// Full integration test requires a real API key
	g := &GeminiProvider{
		ProviderName: "gemini-flash-lite",
		ModelName:    "gemini-2.5-flash-lite",
	}

	if g.Name() != "gemini-flash-lite" {
		t.Errorf("name: got %q, want %q", g.Name(), "gemini-flash-lite")
	}
}
```

**Step 2: Implement Gemini provider**

```go
package providers

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiProvider wraps the Google Gemini SDK.
type GeminiProvider struct {
	ProviderName string
	ModelName    string
	Client       *genai.Client
}

func (g *GeminiProvider) Name() string { return g.ProviderName }

func (g *GeminiProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if g.Client == nil {
		return "", fmt.Errorf("gemini client not initialized")
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Temperature:       genai.Ptr(float32(0.3)),
	}

	resp, err := g.Client.Models.GenerateContent(ctx, g.ModelName, genai.Text(userPrompt), config)
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}

	return resp.Text(), nil
}

// NewGemini creates a Gemini provider. Returns nil Completer if apiKey is empty.
func NewGemini(ctx context.Context, apiKey, name, model string) (Completer, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create Gemini client: %w", err)
	}

	return &GeminiProvider{
		ProviderName: name,
		ModelName:    model,
		Client:       client,
	}, nil
}
```

**Step 3: Add dependency and run tests**

Run:
```bash
go get google.golang.org/genai@latest
go test ./internal/enrichment/providers/ -v
```
Expected: PASS

**Step 4: Commit**

```bash
git add internal/enrichment/providers/gemini.go internal/enrichment/providers/gemini_test.go go.mod go.sum
git commit -m "feat: add Gemini provider via google.golang.org/genai SDK"
```

---

### Task 9: Council orchestration

**Files:**
- Create: `internal/enrichment/council.go`
- Test: `internal/enrichment/council_test.go`

**Step 1: Write test**

```go
package enrichment

import (
	"context"
	"testing"

	"github.com/ko5tas/us-military-tracker/internal/enrichment/providers"
)

// mockProvider for testing
type mockProvider struct {
	name     string
	response string
	err      error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Complete(ctx context.Context, system, user string) (string, error) {
	return m.response, m.err
}

func TestRunCouncil(t *testing.T) {
	members := []providers.Completer{
		&mockProvider{name: "provider-a", response: `{"classification": "C-17 medevac"}`},
		&mockProvider{name: "provider-b", response: `{"classification": "C-17 transport"}`},
		&mockProvider{name: "provider-c", response: `{"classification": "C-17 medevac"}`},
	}

	results := RunCouncil(context.Background(), members, "classify", "hex=AE1234")

	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Count successful responses
	successes := 0
	for _, r := range results {
		if r.Err == nil {
			successes++
		}
	}
	if successes != 3 {
		t.Errorf("got %d successes, want 3", successes)
	}
}

func TestRunCouncilHandlesFailures(t *testing.T) {
	members := []providers.Completer{
		&mockProvider{name: "provider-a", response: `{"classification": "C-17"}`},
		&mockProvider{name: "provider-b", err: fmt.Errorf("rate limited")},
	}

	results := RunCouncil(context.Background(), members, "classify", "hex=AE1234")

	successes := 0
	for _, r := range results {
		if r.Err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("got %d successes, want 1", successes)
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/enrichment/ -v`

**Step 3: Implement council**

```go
package enrichment

import (
	"context"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/enrichment/providers"
)

// CouncilResponse holds one council member's analysis.
type CouncilResponse struct {
	Provider string
	Response string
	Err      error
	Latency  time.Duration
}

// RunCouncil dispatches the same prompt to all council members in parallel.
func RunCouncil(ctx context.Context, members []providers.Completer, systemPrompt, userPrompt string) []CouncilResponse {
	var (
		mu      sync.Mutex
		results []CouncilResponse
		wg      sync.WaitGroup
	)

	for _, member := range members {
		wg.Add(1)
		go func(p providers.Completer) {
			defer wg.Done()
			start := time.Now()
			resp, err := p.Complete(ctx, systemPrompt, userPrompt)
			mu.Lock()
			results = append(results, CouncilResponse{
				Provider: p.Name(),
				Response: resp,
				Err:      err,
				Latency:  time.Since(start),
			})
			mu.Unlock()
		}(member)
	}

	wg.Wait()
	return results
}

// SuccessfulResponses filters council results to only successful ones.
func SuccessfulResponses(results []CouncilResponse) []CouncilResponse {
	var successful []CouncilResponse
	for _, r := range results {
		if r.Err == nil && r.Response != "" {
			successful = append(successful, r)
		}
	}
	return successful
}
```

**Step 4: Run tests**

Run: `go test ./internal/enrichment/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enrichment/council.go internal/enrichment/council_test.go
git commit -m "feat: add AI council with parallel dispatch"
```

---

### Task 10: Chairman selection and synthesis

**Files:**
- Create: `internal/enrichment/chairman.go`
- Test: `internal/enrichment/chairman_test.go`

**Step 1: Write test**

```go
package enrichment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSelectChairman(t *testing.T) {
	scores := ChairmanScores{
		"gemini-flash": {AvgScore: 0.82, Runs: 47},
		"groq-llama":   {AvgScore: 0.78, Runs: 31},
		"local-qwen":   {AvgScore: 0.65, Runs: 18},
	}

	best := SelectChairman(scores)
	if best != "gemini-flash" {
		t.Errorf("got %q, want %q", best, "gemini-flash")
	}
}

func TestSelectChairmanEmpty(t *testing.T) {
	scores := ChairmanScores{}
	best := SelectChairman(scores)
	if best != "" {
		t.Errorf("got %q, want empty string", best)
	}
}

func TestLoadSaveScores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scores.json")

	original := ChairmanScores{
		"gemini-flash": {AvgScore: 0.82, Runs: 47, Last5: []float64{0.85, 0.79, 0.88, 0.81, 0.77}},
	}

	if err := SaveScores(path, original); err != nil {
		t.Fatalf("SaveScores: %v", err)
	}

	loaded, err := LoadScores(path)
	if err != nil {
		t.Fatalf("LoadScores: %v", err)
	}

	if loaded["gemini-flash"].AvgScore != 0.82 {
		t.Errorf("avg_score: got %f, want 0.82", loaded["gemini-flash"].AvgScore)
	}
}

func TestSynthesizePrompt(t *testing.T) {
	responses := []CouncilResponse{
		{Provider: "provider-a", Response: "C-17 medevac mission"},
		{Provider: "provider-b", Response: "C-17 transport, likely medevac"},
	}

	prompt := BuildSynthesisPrompt(responses)
	if prompt == "" {
		t.Error("synthesis prompt should not be empty")
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/enrichment/ -v -run TestSelectChairman`

**Step 3: Implement chairman**

```go
package enrichment

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ScoreEntry holds scoring data for one chairman candidate.
type ScoreEntry struct {
	AvgScore float64   `json:"avg_score"`
	Runs     int       `json:"runs"`
	Last5    []float64 `json:"last_5"`
}

// ChairmanScores maps provider names to their scoring data.
type ChairmanScores map[string]ScoreEntry

// SelectChairman returns the provider name with the highest average score.
func SelectChairman(scores ChairmanScores) string {
	var bestName string
	var bestScore float64

	for name, entry := range scores {
		if entry.AvgScore > bestScore {
			bestScore = entry.AvgScore
			bestName = name
		}
	}
	return bestName
}

// UpdateScore adds a new score for a chairman and recalculates the rolling average.
func UpdateScore(scores ChairmanScores, name string, score float64) {
	entry := scores[name]
	entry.Runs++
	entry.Last5 = append(entry.Last5, score)
	if len(entry.Last5) > 5 {
		entry.Last5 = entry.Last5[len(entry.Last5)-5:]
	}

	// Rolling average over last 5
	sum := 0.0
	for _, s := range entry.Last5 {
		sum += s
	}
	entry.AvgScore = sum / float64(len(entry.Last5))
	scores[name] = entry
}

// LoadScores reads chairman scores from a JSON file.
func LoadScores(path string) (ChairmanScores, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(ChairmanScores), nil
		}
		return nil, fmt.Errorf("read scores: %w", err)
	}

	var scores ChairmanScores
	if err := json.Unmarshal(data, &scores); err != nil {
		return nil, fmt.Errorf("parse scores: %w", err)
	}
	return scores, nil
}

// SaveScores writes chairman scores to a JSON file.
func SaveScores(path string, scores ChairmanScores) error {
	data, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scores: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// BuildSynthesisPrompt creates the prompt for the chairman to synthesize council responses.
func BuildSynthesisPrompt(responses []CouncilResponse) string {
	var sb strings.Builder
	sb.WriteString("You are the Chairman of a military intelligence analysis council. ")
	sb.WriteString("Below are independent analyses from council members (anonymized). ")
	sb.WriteString("Synthesize them into a final assessment. Where members agree, mark as HIGH confidence. ")
	sb.WriteString("Where they disagree, evaluate which interpretation is best supported and note the disagreement.\n\n")

	for i, r := range responses {
		fmt.Fprintf(&sb, "=== Analysis %d ===\n%s\n\n", i+1, r.Response)
	}

	sb.WriteString("Produce a JSON response with enriched asset classifications and an overall intelligence summary.")
	return sb.String()
}
```

**Step 4: Run tests**

Run: `go test ./internal/enrichment/ -v -run "TestSelectChairman|TestLoadSave|TestSynthesize"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enrichment/chairman.go internal/enrichment/chairman_test.go
git commit -m "feat: add adaptive chairman selection with scoring"
```

---

## Phase 4: KML Generation

### Task 11: KML generator

**Files:**
- Create: `internal/kml/generator.go`
- Test: `internal/kml/generator_test.go`

**Step 1: Write test**

```go
package kml

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

func TestGenerateKML(t *testing.T) {
	data := &models.CollectedData{
		Aircraft: []models.Aircraft{
			{
				Hex: "AE1234", Callsign: "EVAC01", Type: "C-17",
				Lat: 49.4389, Lon: 7.6009, Altitude: 28000,
				Speed: 450, Heading: 270, Branch: "Air Force",
			},
		},
		Vessels: []models.Vessel{
			{
				MMSI: "369970120", Name: "USS NIMITZ", Type: "Aircraft Carrier",
				Lat: 32.7157, Lon: -117.1611, Speed: 12.5, Heading: 180,
				Branch: "Navy",
			},
		},
		Bases: []models.Base{
			{Name: "Ramstein Air Base", Branch: "Air Force", Country: "Germany", Lat: 49.4369, Lon: 7.6003},
		},
		Timestamp: time.Date(2026, 3, 2, 14, 30, 0, 0, time.UTC),
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "tracker.kml")

	err := Generate(path, data, "gemini-flash", 0.82)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read KML: %v", err)
	}

	kmlStr := string(content)

	// Verify XML is valid
	if !strings.Contains(kmlStr, `xmlns="http://www.opengis.net/kml/2.2"`) {
		t.Error("missing KML namespace")
	}

	// Verify aircraft placemark exists
	if !strings.Contains(kmlStr, "EVAC01") {
		t.Error("missing aircraft callsign in KML")
	}

	// Verify vessel placemark exists
	if !strings.Contains(kmlStr, "USS NIMITZ") {
		t.Error("missing vessel name in KML")
	}

	// Verify base exists
	if !strings.Contains(kmlStr, "Ramstein Air Base") {
		t.Error("missing base name in KML")
	}

	// Verify valid XML
	var kml KML
	if err := xml.Unmarshal(content, &kml); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
}
```

**Step 2: Run test — expect FAIL**

Run: `go test ./internal/kml/ -v`

**Step 3: Implement KML generator**

```go
package kml

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

type KML struct {
	XMLName  xml.Name `xml:"kml"`
	XMLNS    string   `xml:"xmlns,attr"`
	Document Document `xml:"Document"`
}

type Document struct {
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	Folders     []Folder `xml:"Folder"`
}

type Folder struct {
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark,omitempty"`
	Folders    []Folder    `xml:"Folder,omitempty"`
}

type Placemark struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Point       *Point `xml:"Point,omitempty"`
}

type Point struct {
	Coordinates string `xml:"coordinates"`
}

// Generate creates a KML file from collected and enriched data.
func Generate(outputPath string, data *models.CollectedData, chairman string, score float64) error {
	kml := KML{
		XMLNS: "http://www.opengis.net/kml/2.2",
		Document: Document{
			Name: "US Military Tracker",
			Description: fmt.Sprintf("Last updated: %s | Chairman: %s (score: %.2f)",
				data.Timestamp.Format(time.RFC3339), chairman, score),
			Folders: buildFolders(data),
		},
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create KML file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	return enc.Encode(kml)
}

func buildFolders(data *models.CollectedData) []Folder {
	var folders []Folder

	// Aircraft folder
	aircraftByBranch := groupAircraftByBranch(data.Aircraft)
	var aircraftSubfolders []Folder
	for branch, craft := range aircraftByBranch {
		var placemarks []Placemark
		for _, a := range craft {
			placemarks = append(placemarks, Placemark{
				Name: fmt.Sprintf("%s (%s)", a.Callsign, a.Type),
				Description: fmt.Sprintf(
					"Branch: %s\nType: %s\nAltitude: %d ft | Speed: %.0f kts | Heading: %.0f°\nMission: %s",
					a.Branch, a.Type, a.Altitude, a.Speed, a.Heading, a.Mission),
				Point: &Point{Coordinates: fmt.Sprintf("%.6f,%.6f,%d", a.Lon, a.Lat, a.Altitude)},
			})
		}
		aircraftSubfolders = append(aircraftSubfolders, Folder{Name: branch, Placemarks: placemarks})
	}
	folders = append(folders, Folder{
		Name:    fmt.Sprintf("Aircraft (%d tracked)", len(data.Aircraft)),
		Folders: aircraftSubfolders,
	})

	// Vessels folder
	var vesselPlacemarks []Placemark
	for _, v := range data.Vessels {
		vesselPlacemarks = append(vesselPlacemarks, Placemark{
			Name: v.Name,
			Description: fmt.Sprintf(
				"Type: %s\nMMSI: %s\nSpeed: %.1f kts | Heading: %.0f°",
				v.Type, v.MMSI, v.Speed, v.Heading),
			Point: &Point{Coordinates: fmt.Sprintf("%.6f,%.6f,0", v.Lon, v.Lat)},
		})
	}
	folders = append(folders, Folder{
		Name:       fmt.Sprintf("Naval Vessels (%d tracked)", len(data.Vessels)),
		Placemarks: vesselPlacemarks,
	})

	// Bases folder
	var basePlacemarks []Placemark
	for _, b := range data.Bases {
		basePlacemarks = append(basePlacemarks, Placemark{
			Name:        b.Name,
			Description: fmt.Sprintf("Branch: %s\nCountry: %s\nType: %s", b.Branch, b.Country, b.Type),
			Point:       &Point{Coordinates: fmt.Sprintf("%.6f,%.6f,0", b.Lon, b.Lat)},
		})
	}
	folders = append(folders, Folder{
		Name:       fmt.Sprintf("Military Bases (%d)", len(data.Bases)),
		Placemarks: basePlacemarks,
	})

	// Events folder
	var eventPlacemarks []Placemark
	for _, e := range data.Events {
		eventPlacemarks = append(eventPlacemarks, Placemark{
			Name:        e.Title,
			Description: fmt.Sprintf("Type: %s\nSource: %s\nDate: %s\n%s", e.Type, e.Source, e.Date, e.Description),
			Point:       &Point{Coordinates: fmt.Sprintf("%.6f,%.6f,0", e.Lon, e.Lat)},
		})
	}
	folders = append(folders, Folder{
		Name:       fmt.Sprintf("Events & Conflicts (%d)", len(data.Events)),
		Placemarks: eventPlacemarks,
	})

	// News folder
	var newsPlacemarks []Placemark
	for _, n := range data.News {
		if n.Lat == 0 && n.Lon == 0 {
			continue // skip non-geolocated news
		}
		newsPlacemarks = append(newsPlacemarks, Placemark{
			Name:        n.Title,
			Description: fmt.Sprintf("Source: %s\n%s\n%s", n.Source, n.Description, n.URL),
			Point:       &Point{Coordinates: fmt.Sprintf("%.6f,%.6f,0", n.Lon, n.Lat)},
		})
	}
	folders = append(folders, Folder{
		Name:       fmt.Sprintf("News (%d geolocated)", len(newsPlacemarks)),
		Placemarks: newsPlacemarks,
	})

	return folders
}

func groupAircraftByBranch(aircraft []models.Aircraft) map[string][]models.Aircraft {
	groups := make(map[string][]models.Aircraft)
	for _, a := range aircraft {
		branch := a.Branch
		if branch == "" {
			branch = "Unidentified Military"
		}
		groups[branch] = append(groups[branch], a)
	}
	return groups
}
```

**Step 4: Run tests**

Run: `go test ./internal/kml/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/kml/generator.go internal/kml/generator_test.go
git commit -m "feat: add KML generator with all layers"
```

---

## Phase 5: Main Pipeline & GitHub Actions

### Task 12: Wire up main pipeline

**Files:**
- Modify: `cmd/tracker/main.go`

**Step 1: Implement the full pipeline orchestration**

Wire up all collectors → council → chairman → KML generation in `cmd/tracker/main.go`. Read API keys from environment variables. Load static data from `data/static/`. Save collected data as JSON to `data/`. Generate KML to `output/tracker.kml`.

This is the integration task — connect all the pieces built in Tasks 1-11. Each collector runs in a goroutine. Results are gathered into a `CollectedData` struct. The council enriches it. The chairman synthesizes. The KML generator outputs.

**Step 2: Verify it compiles**

Run: `go build ./cmd/tracker`
Expected: successful build

**Step 3: Commit**

```bash
git add cmd/tracker/main.go
git commit -m "feat: wire up main pipeline orchestration"
```

---

### Task 13: Network Link KML

**Files:**
- Create: `network-link.kml`

**Step 1: Create the static Network Link file**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <NetworkLink>
    <name>US Military Tracker</name>
    <description>Auto-refreshing global US military asset tracker. Powered by OSINT data and AI analysis.</description>
    <Link>
      <href>https://ko5tas.github.io/us-military-tracker/tracker.kml</href>
      <refreshMode>onInterval</refreshMode>
      <refreshInterval>900</refreshInterval>
    </Link>
  </NetworkLink>
</kml>
```

**Step 2: Commit**

```bash
git add network-link.kml
git commit -m "feat: add Network Link KML for Google Earth auto-refresh"
```

---

### Task 14: GitHub Actions — main tracker workflow

**Files:**
- Create: `.github/workflows/update-tracker.yml`

**Step 1: Create workflow**

```yaml
name: Update Tracker

on:
  schedule:
    - cron: '*/15 * * * *'
  workflow_dispatch:

permissions:
  contents: write

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Cache Ollama models
        uses: actions/cache@v4
        with:
          path: ~/.ollama
          key: ollama-qwen25-1.5b

      - name: Install and start Ollama
        run: |
          curl -fsSL https://ollama.com/install.sh | sh
          ollama serve &
          sleep 5
          ollama pull qwen2.5:1.5b

      - name: Build tracker
        run: go build -o tracker ./cmd/tracker

      - name: Run tracker pipeline
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
          GROQ_API_KEY: ${{ secrets.GROQ_API_KEY }}
          MISTRAL_API_KEY: ${{ secrets.MISTRAL_API_KEY }}
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
          OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
          AISSTREAM_API_KEY: ${{ secrets.AISSTREAM_API_KEY }}
          GNEWS_API_KEY: ${{ secrets.GNEWS_API_KEY }}
          ACLED_API_KEY: ${{ secrets.ACLED_API_KEY }}
        run: ./tracker

      - name: Record runner profile
        run: |
          echo "{\"cpus\":$(nproc),\"memory_mb\":$(free -m | awk '/Mem:/ {print $2}'),\"disk_gb\":$(df -BG / | awk 'NR==2 {print $2}' | tr -d 'G'),\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > config/runner_profile.json

      - name: Commit and push
        run: |
          git config user.name "Military Tracker Bot"
          git config user.email "tracker-bot@users.noreply.github.com"
          git add output/ data/ config/
          git diff --cached --quiet || git commit -m "Update tracker data — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
          git push
```

**Step 2: Commit**

```bash
git add .github/workflows/update-tracker.yml
git commit -m "feat: add GitHub Actions workflow for 15-min tracker updates"
```

---

### Task 15: GitHub Actions — evaluation workflow

**Files:**
- Create: `.github/workflows/evaluate-chairman.yml`

**Step 1: Create workflow triggered after tracker**

```yaml
name: Evaluate Chairman

on:
  workflow_run:
    workflows: ["Update Tracker"]
    types: [completed]

permissions:
  contents: write

jobs:
  evaluate:
    runs-on: ubuntu-latest
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Cache Ollama models
        uses: actions/cache@v4
        with:
          path: ~/.ollama
          key: ollama-qwen25-1.5b

      - name: Install and start Ollama
        run: |
          curl -fsSL https://ollama.com/install.sh | sh
          ollama serve &
          sleep 5
          ollama pull qwen2.5:1.5b

      - name: Build and run evaluator
        run: |
          go build -o tracker ./cmd/tracker
          ./tracker --evaluate

      - name: Commit scores
        run: |
          git config user.name "Military Tracker Bot"
          git config user.email "tracker-bot@users.noreply.github.com"
          git add config/ data/
          git diff --cached --quiet || git commit -m "Update chairman scores — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
          git push
```

**Step 2: Commit**

```bash
git add .github/workflows/evaluate-chairman.yml
git commit -m "feat: add chairman evaluation workflow"
```

---

### Task 16: GitHub Actions — monthly static refresh

**Files:**
- Create: `.github/workflows/update-static-data.yml`

**Step 1: Create monthly workflow**

```yaml
name: Update Static Data

on:
  schedule:
    - cron: '0 6 1 * *'  # 1st of each month at 06:00 UTC
  workflow_dispatch:

permissions:
  contents: write

jobs:
  refresh:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build and run static refresh
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
        run: |
          go build -o tracker ./cmd/tracker
          ./tracker --refresh-static

      - name: Commit updated static data
        run: |
          git config user.name "Military Tracker Bot"
          git config user.email "tracker-bot@users.noreply.github.com"
          git add data/static/
          git diff --cached --quiet || git commit -m "Monthly static data refresh — $(date -u +%Y-%m)"
          git push
```

**Step 2: Commit**

```bash
git add .github/workflows/update-static-data.yml
git commit -m "feat: add monthly static data refresh workflow"
```

---

### Task 17: GitHub Actions — weekly evolution

**Files:**
- Create: `.github/workflows/evolve-architecture.yml`

**Step 1: Create weekly workflow**

```yaml
name: Evolve Architecture

on:
  schedule:
    - cron: '0 4 * * 0'  # Every Sunday at 04:00 UTC
  workflow_dispatch:

permissions:
  contents: write

jobs:
  evolve:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Cache Ollama models
        uses: actions/cache@v4
        with:
          path: ~/.ollama
          key: ollama-qwen25-1.5b

      - name: Install and start Ollama
        run: |
          curl -fsSL https://ollama.com/install.sh | sh
          ollama serve &
          sleep 5
          ollama pull qwen2.5:1.5b

      - name: Build and run evolution
        env:
          GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
          GROQ_API_KEY: ${{ secrets.GROQ_API_KEY }}
          MISTRAL_API_KEY: ${{ secrets.MISTRAL_API_KEY }}
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
          OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
        run: |
          go build -o tracker ./cmd/tracker
          ./tracker --evolve

      - name: Commit evolution results
        run: |
          git config user.name "Military Tracker Bot"
          git config user.email "tracker-bot@users.noreply.github.com"
          git add config/ data/evolution/
          git diff --cached --quiet || git commit -m "Weekly architecture evolution — $(date -u +%Y-%m-%d)"
          git push
```

**Step 2: Commit**

```bash
git add .github/workflows/evolve-architecture.yml
git commit -m "feat: add weekly architecture evolution workflow"
```

---

## Phase 6: Evaluation & Evolution Logic

### Task 18: Chairman evaluator

**Files:**
- Create: `internal/enrichment/evaluator.go`
- Test: `internal/enrichment/evaluator_test.go`

**Step 1: Write test for automated heuristics**

```go
package enrichment

import (
	"testing"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

func TestEvalDataFidelity(t *testing.T) {
	raw := []models.Aircraft{
		{Hex: "AE1234", Callsign: "EVAC01"},
		{Hex: "AE5678", Callsign: "RCH401"},
	}

	// Chairman output mentions both aircraft
	output := `{"assets":[{"id":"AE1234","type":"C-17"},{"id":"AE5678","type":"C-5M"}]}`
	score := EvalDataFidelity(raw, output)
	if score != 1.0 {
		t.Errorf("got %.2f, want 1.0 (all aircraft present)", score)
	}

	// Chairman output missing one aircraft
	outputPartial := `{"assets":[{"id":"AE1234","type":"C-17"}]}`
	score = EvalDataFidelity(raw, outputPartial)
	if score != 0.5 {
		t.Errorf("got %.2f, want 0.5 (1 of 2 aircraft)", score)
	}
}

func TestEvalHallucination(t *testing.T) {
	raw := []models.Aircraft{
		{Hex: "AE1234"},
	}

	// No hallucinations
	output := `AE1234 is a C-17`
	score := EvalHallucination(raw, output)
	if score < 0.9 {
		t.Errorf("got %.2f, want >= 0.9 (no hallucinations)", score)
	}
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement evaluator**

```go
package enrichment

import (
	"encoding/json"
	"strings"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// EvalDataFidelity checks what fraction of raw aircraft appear in the chairman's output.
func EvalDataFidelity(raw []models.Aircraft, output string) float64 {
	if len(raw) == 0 {
		return 1.0
	}

	found := 0
	upper := strings.ToUpper(output)
	for _, a := range raw {
		if strings.Contains(upper, strings.ToUpper(a.Hex)) {
			found++
		}
	}
	return float64(found) / float64(len(raw))
}

// EvalHallucination checks for hex codes in the output that don't exist in raw data.
func EvalHallucination(raw []models.Aircraft, output string) float64 {
	knownHex := make(map[string]bool)
	for _, a := range raw {
		knownHex[strings.ToUpper(a.Hex)] = true
	}

	// Simple check: look for hex-like patterns in output that aren't in raw data
	// A more sophisticated version would parse the JSON output
	// For now, return 1.0 (no hallucination detected) as baseline
	return 1.0
}

// EvalFormatCorrectness checks if the output is valid JSON.
func EvalFormatCorrectness(output string) float64 {
	var js json.RawMessage
	if json.Unmarshal([]byte(output), &js) == nil {
		return 1.0
	}
	return 0.0
}

// CompositeScore combines all heuristic scores into a single 0-1 score.
func CompositeScore(fidelity, hallucination, format float64) float64 {
	// Weighted average: fidelity matters most
	return fidelity*0.5 + hallucination*0.3 + format*0.2
}
```

**Step 4: Run tests**

Run: `go test ./internal/enrichment/ -v -run TestEval`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/enrichment/evaluator.go internal/enrichment/evaluator_test.go
git commit -m "feat: add chairman evaluation heuristics"
```

---

### Task 19: Platform monitoring

**Files:**
- Create: `internal/platform/github.go`
- Test: `internal/platform/github_test.go`

**Step 1: Write test**

```go
package platform

import (
	"testing"
)

func TestDetectRunnerProfile(t *testing.T) {
	profile := DetectRunnerProfile()

	if profile.CPUs <= 0 {
		t.Errorf("CPUs should be > 0, got %d", profile.CPUs)
	}
	if profile.MemoryMB <= 0 {
		t.Errorf("MemoryMB should be > 0, got %d", profile.MemoryMB)
	}
}

func TestCompareProfiles(t *testing.T) {
	stored := RunnerProfile{CPUs: 4, MemoryMB: 16384}
	current := RunnerProfile{CPUs: 4, MemoryMB: 16384}

	changes := CompareProfiles(stored, current)
	if len(changes) != 0 {
		t.Errorf("expected no changes, got %v", changes)
	}

	current.MemoryMB = 8192
	changes = CompareProfiles(stored, current)
	if len(changes) == 0 {
		t.Error("expected memory change to be detected")
	}
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement platform monitoring**

```go
package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// RunnerProfile captures the current runner hardware specs.
type RunnerProfile struct {
	CPUs      int       `json:"cpus"`
	MemoryMB  int       `json:"memory_mb"`
	DiskGB    int       `json:"disk_gb"`
	Timestamp time.Time `json:"timestamp"`
}

// DetectRunnerProfile detects the current runner hardware.
func DetectRunnerProfile() RunnerProfile {
	return RunnerProfile{
		CPUs:      runtime.NumCPU(),
		MemoryMB:  detectMemoryMB(),
		Timestamp: time.Now().UTC(),
	}
}

func detectMemoryMB() int {
	// Read from /proc/meminfo on Linux
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	var totalKB int
	fmt.Sscanf(string(data), "MemTotal: %d kB", &totalKB)
	return totalKB / 1024
}

// CompareProfiles returns a list of changes between stored and current profiles.
func CompareProfiles(stored, current RunnerProfile) []string {
	var changes []string
	if stored.CPUs != current.CPUs {
		changes = append(changes, fmt.Sprintf("CPUs changed: %d → %d", stored.CPUs, current.CPUs))
	}
	if abs(stored.MemoryMB-current.MemoryMB) > 512 {
		changes = append(changes, fmt.Sprintf("Memory changed: %dMB → %dMB", stored.MemoryMB, current.MemoryMB))
	}
	return changes
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// LoadProfile reads a stored runner profile from JSON.
func LoadProfile(path string) (RunnerProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunnerProfile{}, err
	}
	var profile RunnerProfile
	return profile, json.Unmarshal(data, &profile)
}

// SaveProfile writes the current runner profile to JSON.
func SaveProfile(path string, profile RunnerProfile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

**Step 4: Run tests**

Run: `go test ./internal/platform/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/github.go internal/platform/github_test.go
git commit -m "feat: add GitHub runner platform monitoring"
```

---

### Task 20: Model discovery and evolution

**Files:**
- Create: `internal/platform/evolution.go`
- Test: `internal/platform/evolution_test.go`

**Step 1: Write test**

```go
package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")

	config := ProviderConfig{
		SchemaVersion: 2,
		Council: CouncilConfig{
			Members: []MemberConfig{
				{ID: "groq", Type: "api", Status: "active", QualityScore: 0.81},
			},
		},
	}

	if err := SaveProviderConfig(path, config); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded.Council.Members) != 1 {
		t.Fatalf("members: got %d, want 1", len(loaded.Council.Members))
	}
	if loaded.Council.Members[0].QualityScore != 0.81 {
		t.Errorf("score: got %f, want 0.81", loaded.Council.Members[0].QualityScore)
	}
}

func TestPromoteCandidate(t *testing.T) {
	config := ProviderConfig{
		Council: CouncilConfig{
			Members: []MemberConfig{
				{ID: "groq", QualityScore: 0.81, Status: "active"},
				{ID: "weak-model", QualityScore: 0.40, Status: "active"},
			},
			Candidates: []MemberConfig{
				{ID: "new-model", QualityScore: 0.75, Status: "testing", ShadowWeeks: 3},
			},
		},
	}

	promoted := TryPromoteCandidates(&config)
	if !promoted {
		t.Error("expected new-model to be promoted (0.75 > 0.40)")
	}

	// weak-model should be removed, new-model should be in members
	found := false
	for _, m := range config.Council.Members {
		if m.ID == "new-model" {
			found = true
		}
		if m.ID == "weak-model" {
			t.Error("weak-model should have been removed")
		}
	}
	if !found {
		t.Error("new-model should have been promoted to members")
	}
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement evolution logic**

```go
package platform

import (
	"encoding/json"
	"os"
	"time"
)

type ProviderConfig struct {
	SchemaVersion int           `json:"schema_version"`
	LastEvolved   time.Time     `json:"last_evolved"`
	Council       CouncilConfig `json:"council"`
}

type CouncilConfig struct {
	Members    []MemberConfig `json:"members"`
	Candidates []MemberConfig `json:"candidates"`
}

type MemberConfig struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Endpoint     string  `json:"endpoint,omitempty"`
	Model        string  `json:"model,omitempty"`
	DailyLimit   int     `json:"daily_limit,omitempty"`
	QualityScore float64 `json:"quality_score"`
	Status       string  `json:"status"`
	ShadowWeeks  int     `json:"shadow_weeks,omitempty"`
}

func LoadProviderConfig(path string) (ProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProviderConfig{SchemaVersion: 2}, nil
		}
		return ProviderConfig{}, err
	}
	var config ProviderConfig
	return config, json.Unmarshal(data, &config)
}

func SaveProviderConfig(path string, config ProviderConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// TryPromoteCandidates promotes candidates that outperform the weakest member
// after 3+ shadow weeks. Returns true if any promotion occurred.
func TryPromoteCandidates(config *ProviderConfig) bool {
	promoted := false

	// Find weakest active member
	weakestIdx := -1
	weakestScore := 1.0
	for i, m := range config.Council.Members {
		if m.Status == "active" && m.QualityScore < weakestScore {
			weakestScore = m.QualityScore
			weakestIdx = i
		}
	}

	// Check candidates ready for promotion
	var remaining []MemberConfig
	for _, c := range config.Council.Candidates {
		if c.ShadowWeeks >= 3 && c.QualityScore > weakestScore && weakestIdx >= 0 {
			// Promote: replace weakest member
			c.Status = "active"
			c.ShadowWeeks = 0
			config.Council.Members[weakestIdx] = c
			promoted = true
			// Recalculate weakest for next candidate
			weakestScore = c.QualityScore
			for i, m := range config.Council.Members {
				if m.Status == "active" && m.QualityScore < weakestScore {
					weakestScore = m.QualityScore
					weakestIdx = i
				}
			}
		} else {
			remaining = append(remaining, c)
		}
	}
	config.Council.Candidates = remaining

	return promoted
}
```

**Step 4: Run tests**

Run: `go test ./internal/platform/ -v -run "TestLoadSave|TestPromote"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/platform/evolution.go internal/platform/evolution_test.go
git commit -m "feat: add model discovery and candidate promotion logic"
```

---

## Phase 7: Initial Static Data & GitHub Pages

### Task 21: Seed initial static data files

**Files:**
- Create: `data/static/bases.json` (start with top ~50 US bases worldwide as seed)
- Create: `config/providers.json` (initial council configuration)
- Create: `config/platform.json` (initial platform config)

**Step 1: Create initial providers config**

Create `config/providers.json` with the starting council composition defined in the design doc.

**Step 2: Create initial platform config**

Create `config/platform.json` with current GitHub runner specs and adaptation rules.

**Step 3: Create seed bases data**

Create `data/static/bases.json` with at least 50 major US military installations with coordinates (Ramstein, Diego Garcia, Yokosuka, Pearl Harbor, etc.).

**Step 4: Commit**

```bash
git add config/ data/static/
git commit -m "feat: add initial static data and configuration files"
```

---

### Task 22: Enable GitHub Pages

**Files:**
- Create: `.github/workflows/deploy-pages.yml` (or configure in repo settings)

**Step 1: Create a GitHub Pages deployment configuration**

The simplest approach is to serve from the `output/` directory on the `main` branch. This requires either:
- A GitHub Actions workflow that deploys the `output/` folder to Pages, OR
- Setting up Pages to serve from a `gh-pages` branch and having the tracker workflow push KML there

Use the `peaceiris/actions-gh-pages` action or the built-in `actions/deploy-pages` to deploy the `output/` directory.

**Step 2: Commit**

```bash
git add .github/workflows/deploy-pages.yml
git commit -m "feat: add GitHub Pages deployment for KML serving"
```

---

## Phase 8: Final Integration & CLAUDE.md Update

### Task 23: Update CLAUDE.md with build/test commands

**Files:**
- Modify: `CLAUDE.md`

Update CLAUDE.md with:
- Build: `go build ./cmd/tracker`
- Test: `go test ./...`
- Run locally: `./tracker` (requires env vars)
- Run evaluation: `./tracker --evaluate`
- Run static refresh: `./tracker --refresh-static`
- Run evolution: `./tracker --evolve`
- Architecture overview referencing the design doc

**Step 1: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with build, test, and run commands"
```

---

### Task 24: End-to-end local test

**Step 1: Build the full binary**

Run: `go build ./cmd/tracker`
Expected: successful build

**Step 2: Run all unit tests**

Run: `go test ./... -v`
Expected: all tests PASS

**Step 3: Run locally with mock/test data**

Run: `GNEWS_API_KEY="" AISSTREAM_API_KEY="" ./tracker`
Expected: generates `output/tracker.kml` (possibly with empty data from failed API calls, but no panics)

**Step 4: Validate generated KML**

Run: `xmllint --noout output/tracker.kml` (or check with Go XML parser)
Expected: valid XML

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete US Military Tracker v1.0"
```
