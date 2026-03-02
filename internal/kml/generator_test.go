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

func sampleData() *models.CollectedData {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	return &models.CollectedData{
		Aircraft: []models.Aircraft{
			{
				Hex:       "AE1234",
				Callsign:  "REACH01",
				Type:      "C-17A",
				Lat:       38.9072,
				Lon:       -77.0369,
				Altitude:  35000,
				Speed:     450.5,
				Heading:   90.0,
				Branch:    "USAF",
				Mission:   "transport",
				Timestamp: ts,
			},
		},
		Vessels: []models.Vessel{
			{
				MMSI:    "369970120",
				Name:    "USS NIMITZ",
				Type:    "aircraft_carrier",
				Lat:     32.7157,
				Lon:     -117.1611,
				Speed:   18.5,
				Heading: 270.0,
				Branch:  "USN",
			},
		},
		Bases: []models.Base{
			{
				Name:    "Ramstein Air Base",
				Branch:  "USAF",
				Country: "Germany",
				Lat:     49.4369,
				Lon:     7.6003,
				Type:    "air_base",
			},
		},
		Events: []models.Event{
			{
				Title:       "Joint Pacific Exercise",
				Type:        "military_exercise",
				Source:      "gdelt",
				Date:        "2026-03-01",
				Description: "Annual joint military exercise.",
				Lat:         21.3069,
				Lon:         -157.8583,
			},
		},
		News: []models.NewsItem{
			{
				Title:       "Geolocated News",
				Description: "A news item with valid location.",
				URL:         "https://example.com/news/1",
				Source:      "reuters",
				Lat:         36.8529,
				Lon:         -75.9780,
			},
			{
				Title:       "Non-Geolocated News",
				Description: "A news item with zero lat/lon.",
				URL:         "https://example.com/news/2",
				Source:      "ap",
				Lat:         0,
				Lon:         0,
			},
		},
		Timestamp: ts,
	}
}

func TestGenerateKML(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "test.kml")

	data := sampleData()
	err := Generate(outputPath, data, "claude", 0.95)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	kmlStr := string(content)

	// Verify XML header is present
	if !strings.HasPrefix(kmlStr, xml.Header) {
		t.Error("KML output should start with XML header")
	}

	// Verify KML namespace
	if !strings.Contains(kmlStr, "http://www.opengis.net/kml/2.2") {
		t.Error("KML output should contain OGC KML namespace")
	}

	// Verify aircraft callsign is in the output
	if !strings.Contains(kmlStr, "REACH01") {
		t.Error("KML output should contain aircraft callsign REACH01")
	}

	// Verify vessel name is in the output
	if !strings.Contains(kmlStr, "USS NIMITZ") {
		t.Error("KML output should contain vessel name USS NIMITZ")
	}

	// Verify base name is in the output
	if !strings.Contains(kmlStr, "Ramstein Air Base") {
		t.Error("KML output should contain base name Ramstein Air Base")
	}

	// Verify event title is in the output
	if !strings.Contains(kmlStr, "Joint Pacific Exercise") {
		t.Error("KML output should contain event title")
	}

	// Verify geolocated news is present
	if !strings.Contains(kmlStr, "Geolocated News") {
		t.Error("KML output should contain geolocated news item")
	}

	// Verify non-geolocated news (0,0) is skipped
	if strings.Contains(kmlStr, "Non-Geolocated News") {
		t.Error("KML output should NOT contain news items with lat/lon 0,0")
	}

	// Verify chairman name in description
	if !strings.Contains(kmlStr, "claude") {
		t.Error("KML output should contain chairman name")
	}

	// Verify score in description
	if !strings.Contains(kmlStr, "0.95") {
		t.Error("KML output should contain score")
	}

	// Verify folder names include counts
	if !strings.Contains(kmlStr, "Aircraft (1 tracked)") {
		t.Error("KML output should contain aircraft folder with count")
	}
	if !strings.Contains(kmlStr, "Vessels (1 tracked)") {
		t.Error("KML output should contain vessels folder with count")
	}
	if !strings.Contains(kmlStr, "Bases (1 tracked)") {
		t.Error("KML output should contain bases folder with count")
	}

	// Verify coordinates format: lon,lat,altitude (longitude first)
	if !strings.Contains(kmlStr, "-77.0369,38.9072,35000") {
		t.Error("KML output should contain aircraft coordinates in lon,lat,alt format")
	}

	// Verify style definitions are present
	if !strings.Contains(kmlStr, `id="usaf"`) {
		t.Error("KML output should contain USAF style definition")
	}
	if !strings.Contains(kmlStr, `id="vessel"`) {
		t.Error("KML output should contain vessel style definition")
	}
	if !strings.Contains(kmlStr, `id="event"`) {
		t.Error("KML output should contain event style definition")
	}

	// Verify styleUrl references are set on placemarks
	if !strings.Contains(kmlStr, "#usaf") {
		t.Error("KML output should reference USAF style on aircraft")
	}
	if !strings.Contains(kmlStr, "#vessel") {
		t.Error("KML output should reference vessel style on vessels")
	}
	if !strings.Contains(kmlStr, "#event") {
		t.Error("KML output should reference event style on events")
	}

	// Verify icon hrefs are present
	if !strings.Contains(kmlStr, "maps.google.com/mapfiles/kml/paddle/") {
		t.Error("KML output should contain Google Maps paddle icon URLs")
	}
}

func TestGenerateKMLValidXML(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "valid.kml")

	data := sampleData()
	err := Generate(outputPath, data, "claude", 0.95)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Unmarshal back into KML struct to verify valid XML
	var kmlDoc KML
	if err := xml.Unmarshal(content, &kmlDoc); err != nil {
		t.Fatalf("Failed to unmarshal KML XML: %v", err)
	}

	// Verify document name is set
	if kmlDoc.Document.Name == "" {
		t.Error("KML Document should have a name")
	}

	// Verify folders exist
	if len(kmlDoc.Document.Folders) == 0 {
		t.Error("KML Document should have folders")
	}
}

func TestGenerateNewsSkipsZeroLatLon(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "news.kml")

	data := &models.CollectedData{
		News: []models.NewsItem{
			{
				Title:       "Has Location",
				Description: "A news item with valid location.",
				URL:         "https://example.com/news/1",
				Source:      "reuters",
				Lat:         36.8529,
				Lon:         -75.9780,
			},
			{
				Title:       "Zero Lat Only",
				Description: "Lat is zero, lon is not.",
				URL:         "https://example.com/news/2",
				Source:      "ap",
				Lat:         0,
				Lon:         10.0,
			},
			{
				Title:       "Zero Lon Only",
				Description: "Lon is zero, lat is not.",
				URL:         "https://example.com/news/3",
				Source:      "bbc",
				Lat:         10.0,
				Lon:         0,
			},
			{
				Title:       "Both Zero",
				Description: "Both lat and lon are zero.",
				URL:         "https://example.com/news/4",
				Source:      "cnn",
				Lat:         0,
				Lon:         0,
			},
		},
		Timestamp: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}

	err := Generate(outputPath, data, "claude", 0.85)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	kmlStr := string(content)

	// "Has Location" should be present (non-zero lat and lon)
	if !strings.Contains(kmlStr, "Has Location") {
		t.Error("KML should contain news item with valid location")
	}

	// "Zero Lat Only" should still be present (only both zero means skip)
	// Actually, the requirement says "skip items where both Lat and Lon are 0"
	if !strings.Contains(kmlStr, "Zero Lat Only") {
		t.Error("KML should contain news item where only lat is zero")
	}

	if !strings.Contains(kmlStr, "Zero Lon Only") {
		t.Error("KML should contain news item where only lon is zero")
	}

	// "Both Zero" should NOT be present
	if strings.Contains(kmlStr, "Both Zero") {
		t.Error("KML should NOT contain news items where both lat and lon are 0")
	}
}

func TestGenerateAircraftGroupedByBranch(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "branches.kml")

	data := &models.CollectedData{
		Aircraft: []models.Aircraft{
			{
				Hex:      "AE1234",
				Callsign: "REACH01",
				Type:     "C-17A",
				Lat:      38.9072,
				Lon:      -77.0369,
				Altitude: 35000,
				Speed:    450.5,
				Heading:  90.0,
				Branch:   "USAF",
			},
			{
				Hex:      "AE5678",
				Callsign: "NAVY01",
				Type:     "P-8A",
				Lat:      32.7157,
				Lon:      -117.1611,
				Altitude: 25000,
				Speed:    400.0,
				Heading:  180.0,
				Branch:   "USN",
			},
			{
				Hex:      "AE9999",
				Callsign: "REACH02",
				Type:     "C-5M",
				Lat:      39.0,
				Lon:      -76.0,
				Altitude: 30000,
				Speed:    420.0,
				Heading:  45.0,
				Branch:   "USAF",
			},
		},
		Timestamp: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}

	err := Generate(outputPath, data, "claude", 0.90)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	kmlStr := string(content)

	// Aircraft should be grouped by branch - check for branch sub-folders
	if !strings.Contains(kmlStr, "USAF") {
		t.Error("KML should contain USAF branch folder")
	}
	if !strings.Contains(kmlStr, "USN") {
		t.Error("KML should contain USN branch folder")
	}

	// Total aircraft count in folder name
	if !strings.Contains(kmlStr, "Aircraft (3 tracked)") {
		t.Error("KML should show total aircraft count of 3")
	}
}
