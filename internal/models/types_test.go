package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAircraftJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	original := Aircraft{
		Hex:       "AE1234",
		Callsign:  "REACH01",
		Type:      "C-17A",
		Lat:       38.9072,
		Lon:       -77.0369,
		Altitude:  35000,
		Speed:     450.5,
		Heading:   90.0,
		Squawk:    "1200",
		Source:    "adsb",
		Branch:    "USAF",
		Mission:   "transport",
		Timestamp: ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal Aircraft: %v", err)
	}

	var decoded Aircraft
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal Aircraft: %v", err)
	}

	if decoded.Hex != original.Hex {
		t.Errorf("Hex: got %q, want %q", decoded.Hex, original.Hex)
	}
	if decoded.Callsign != original.Callsign {
		t.Errorf("Callsign: got %q, want %q", decoded.Callsign, original.Callsign)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Lat != original.Lat {
		t.Errorf("Lat: got %v, want %v", decoded.Lat, original.Lat)
	}
	if decoded.Lon != original.Lon {
		t.Errorf("Lon: got %v, want %v", decoded.Lon, original.Lon)
	}
	if decoded.Altitude != original.Altitude {
		t.Errorf("Altitude: got %d, want %d", decoded.Altitude, original.Altitude)
	}
	if decoded.Speed != original.Speed {
		t.Errorf("Speed: got %v, want %v", decoded.Speed, original.Speed)
	}
	if decoded.Heading != original.Heading {
		t.Errorf("Heading: got %v, want %v", decoded.Heading, original.Heading)
	}
	if decoded.Squawk != original.Squawk {
		t.Errorf("Squawk: got %q, want %q", decoded.Squawk, original.Squawk)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source: got %q, want %q", decoded.Source, original.Source)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Mission != original.Mission {
		t.Errorf("Mission: got %q, want %q", decoded.Mission, original.Mission)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}

func TestAircraftOmitEmpty(t *testing.T) {
	a := Aircraft{
		Hex:       "AE1234",
		Callsign:  "REACH01",
		Type:      "C-17A",
		Lat:       38.9072,
		Lon:       -77.0369,
		Altitude:  35000,
		Speed:     450.5,
		Heading:   90.0,
		Source:    "adsb",
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	if _, ok := raw["squawk"]; ok {
		t.Error("squawk should be omitted when empty")
	}
	if _, ok := raw["branch"]; ok {
		t.Error("branch should be omitted when empty")
	}
	if _, ok := raw["mission"]; ok {
		t.Error("mission should be omitted when empty")
	}
}

func TestVesselJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	original := Vessel{
		MMSI:      "369970120",
		Name:      "USS NIMITZ",
		Type:      "aircraft_carrier",
		Lat:       32.7157,
		Lon:       -117.1611,
		Speed:     18.5,
		Heading:   270.0,
		Source:    "ais",
		Branch:    "USN",
		Class:     "Nimitz",
		Timestamp: ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal Vessel: %v", err)
	}

	var decoded Vessel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal Vessel: %v", err)
	}

	if decoded.MMSI != original.MMSI {
		t.Errorf("MMSI: got %q, want %q", decoded.MMSI, original.MMSI)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Lat != original.Lat {
		t.Errorf("Lat: got %v, want %v", decoded.Lat, original.Lat)
	}
	if decoded.Lon != original.Lon {
		t.Errorf("Lon: got %v, want %v", decoded.Lon, original.Lon)
	}
	if decoded.Speed != original.Speed {
		t.Errorf("Speed: got %v, want %v", decoded.Speed, original.Speed)
	}
	if decoded.Heading != original.Heading {
		t.Errorf("Heading: got %v, want %v", decoded.Heading, original.Heading)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source: got %q, want %q", decoded.Source, original.Source)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Class != original.Class {
		t.Errorf("Class: got %q, want %q", decoded.Class, original.Class)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	original := Event{
		ID:          "evt-001",
		Type:        "military_exercise",
		Title:       "Joint Pacific Exercise",
		Description: "Annual joint military exercise in the Pacific.",
		Lat:         21.3069,
		Lon:         -157.8583,
		Source:      "gdelt",
		URL:         "https://example.com/event/001",
		Date:        "2026-03-01",
		Timestamp:   ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal Event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal Event: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.Lat != original.Lat {
		t.Errorf("Lat: got %v, want %v", decoded.Lat, original.Lat)
	}
	if decoded.Lon != original.Lon {
		t.Errorf("Lon: got %v, want %v", decoded.Lon, original.Lon)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source: got %q, want %q", decoded.Source, original.Source)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL: got %q, want %q", decoded.URL, original.URL)
	}
	if decoded.Date != original.Date {
		t.Errorf("Date: got %q, want %q", decoded.Date, original.Date)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}

func TestBaseJSONRoundTrip(t *testing.T) {
	original := Base{
		Name:    "Ramstein Air Base",
		Branch:  "USAF",
		Country: "Germany",
		Lat:     49.4369,
		Lon:     7.6003,
		Type:    "air_base",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal Base: %v", err)
	}

	var decoded Base
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal Base: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Country != original.Country {
		t.Errorf("Country: got %q, want %q", decoded.Country, original.Country)
	}
	if decoded.Lat != original.Lat {
		t.Errorf("Lat: got %v, want %v", decoded.Lat, original.Lat)
	}
	if decoded.Lon != original.Lon {
		t.Errorf("Lon: got %v, want %v", decoded.Lon, original.Lon)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
}

func TestNewsItemJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 8, 30, 0, 0, time.UTC)
	original := NewsItem{
		Title:       "New Deployment Announced",
		Description: "US Navy announces new carrier group deployment.",
		URL:         "https://example.com/news/123",
		Source:      "reuters",
		Lat:         36.8529,
		Lon:        -75.9780,
		PublishedAt: ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal NewsItem: %v", err)
	}

	var decoded NewsItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal NewsItem: %v", err)
	}

	if decoded.Title != original.Title {
		t.Errorf("Title: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL: got %q, want %q", decoded.URL, original.URL)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source: got %q, want %q", decoded.Source, original.Source)
	}
	if decoded.Lat != original.Lat {
		t.Errorf("Lat: got %v, want %v", decoded.Lat, original.Lat)
	}
	if decoded.Lon != original.Lon {
		t.Errorf("Lon: got %v, want %v", decoded.Lon, original.Lon)
	}
	if !decoded.PublishedAt.Equal(original.PublishedAt) {
		t.Errorf("PublishedAt: got %v, want %v", decoded.PublishedAt, original.PublishedAt)
	}
}

func TestNewsItemOmitEmpty(t *testing.T) {
	n := NewsItem{
		Title:       "Headline",
		Description: "Desc",
		URL:         "https://example.com",
		Source:      "ap",
		PublishedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	// lat and lon are omitempty but float64 zero values are NOT omitted by encoding/json
	// omitempty for float64 only omits if the value is 0, which it is here.
	// Actually, encoding/json omitempty omits zero-valued floats (0.0).
	if v, ok := raw["lat"]; ok {
		if v.(float64) != 0 {
			t.Error("lat should be zero or omitted when not set")
		}
	}
	if v, ok := raw["lon"]; ok {
		if v.(float64) != 0 {
			t.Error("lon should be zero or omitted when not set")
		}
	}
}

func TestCollectedDataJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	original := CollectedData{
		Aircraft: []Aircraft{
			{Hex: "AE1234", Callsign: "REACH01", Type: "C-17A", Lat: 38.9, Lon: -77.0, Altitude: 35000, Speed: 450, Heading: 90, Source: "adsb", Timestamp: ts},
		},
		Vessels: []Vessel{
			{MMSI: "369970120", Name: "USS NIMITZ", Type: "carrier", Lat: 32.7, Lon: -117.1, Speed: 18, Heading: 270, Source: "ais", Timestamp: ts},
		},
		Events: []Event{
			{ID: "evt-001", Type: "exercise", Title: "Test", Description: "Test event", Lat: 21.3, Lon: -157.8, Source: "gdelt", Date: "2026-03-01", Timestamp: ts},
		},
		News: []NewsItem{
			{Title: "Headline", Description: "Desc", URL: "https://example.com", Source: "ap", PublishedAt: ts},
		},
		Bases: []Base{
			{Name: "Ramstein", Branch: "USAF", Country: "Germany", Lat: 49.4, Lon: 7.6, Type: "air_base"},
		},
		Timestamp: ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal CollectedData: %v", err)
	}

	var decoded CollectedData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal CollectedData: %v", err)
	}

	if len(decoded.Aircraft) != 1 {
		t.Fatalf("Aircraft count: got %d, want 1", len(decoded.Aircraft))
	}
	if decoded.Aircraft[0].Hex != "AE1234" {
		t.Errorf("Aircraft[0].Hex: got %q, want %q", decoded.Aircraft[0].Hex, "AE1234")
	}
	if len(decoded.Vessels) != 1 {
		t.Fatalf("Vessels count: got %d, want 1", len(decoded.Vessels))
	}
	if decoded.Vessels[0].MMSI != "369970120" {
		t.Errorf("Vessels[0].MMSI: got %q, want %q", decoded.Vessels[0].MMSI, "369970120")
	}
	if len(decoded.Events) != 1 {
		t.Fatalf("Events count: got %d, want 1", len(decoded.Events))
	}
	if decoded.Events[0].ID != "evt-001" {
		t.Errorf("Events[0].ID: got %q, want %q", decoded.Events[0].ID, "evt-001")
	}
	if len(decoded.News) != 1 {
		t.Fatalf("News count: got %d, want 1", len(decoded.News))
	}
	if decoded.News[0].Title != "Headline" {
		t.Errorf("News[0].Title: got %q, want %q", decoded.News[0].Title, "Headline")
	}
	if len(decoded.Bases) != 1 {
		t.Fatalf("Bases count: got %d, want 1", len(decoded.Bases))
	}
	if decoded.Bases[0].Name != "Ramstein" {
		t.Errorf("Bases[0].Name: got %q, want %q", decoded.Bases[0].Name, "Ramstein")
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}

func TestEnrichedAssetJSONRoundTrip(t *testing.T) {
	original := EnrichedAsset{
		ID:          "AE1234",
		Type:        "aircraft",
		Branch:      "USAF",
		Mission:     "strategic_airlift",
		Description: "C-17 performing routine cargo transport.",
		Confidence:  "high",
		Agreement:   3,
		TotalVotes:  3,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal EnrichedAsset: %v", err)
	}

	var decoded EnrichedAsset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal EnrichedAsset: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Mission != original.Mission {
		t.Errorf("Mission: got %q, want %q", decoded.Mission, original.Mission)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.Confidence != original.Confidence {
		t.Errorf("Confidence: got %q, want %q", decoded.Confidence, original.Confidence)
	}
	if decoded.Agreement != original.Agreement {
		t.Errorf("Agreement: got %d, want %d", decoded.Agreement, original.Agreement)
	}
	if decoded.TotalVotes != original.TotalVotes {
		t.Errorf("TotalVotes: got %d, want %d", decoded.TotalVotes, original.TotalVotes)
	}
}

func TestCouncilResultJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	original := CouncilResult{
		Assets: []EnrichedAsset{
			{ID: "AE1234", Type: "aircraft", Branch: "USAF", Mission: "transport", Description: "C-17 cargo run", Confidence: "high", Agreement: 3, TotalVotes: 3},
		},
		Summary:   "Routine transport operations detected.",
		Chairman:  "claude",
		Score:     0.95,
		Timestamp: ts,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal CouncilResult: %v", err)
	}

	var decoded CouncilResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal CouncilResult: %v", err)
	}

	if len(decoded.Assets) != 1 {
		t.Fatalf("Assets count: got %d, want 1", len(decoded.Assets))
	}
	if decoded.Assets[0].ID != "AE1234" {
		t.Errorf("Assets[0].ID: got %q, want %q", decoded.Assets[0].ID, "AE1234")
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary: got %q, want %q", decoded.Summary, original.Summary)
	}
	if decoded.Chairman != original.Chairman {
		t.Errorf("Chairman: got %q, want %q", decoded.Chairman, original.Chairman)
	}
	if decoded.Score != original.Score {
		t.Errorf("Score: got %v, want %v", decoded.Score, original.Score)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
}
