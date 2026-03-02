package collectors

import (
	"testing"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

func TestParseAISMessage_PositionReport(t *testing.T) {
	raw := []byte(`{
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
	}`)

	v, err := parseAISMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.MMSI != "369970120" {
		t.Errorf("MMSI = %q, want %q", v.MMSI, "369970120")
	}
	if v.Name != "USS NIMITZ" {
		t.Errorf("Name = %q, want %q", v.Name, "USS NIMITZ")
	}
	if v.Lat != 32.7157 {
		t.Errorf("Lat = %f, want %f", v.Lat, 32.7157)
	}
	if v.Lon != -117.1611 {
		t.Errorf("Lon = %f, want %f", v.Lon, -117.1611)
	}
	if v.Speed != 12.5 {
		t.Errorf("Speed = %f, want %f", v.Speed, 12.5)
	}
	if v.Heading != 180 {
		t.Errorf("Heading = %f, want %f", v.Heading, 180.0)
	}
	if v.Source != "aisstream.io" {
		t.Errorf("Source = %q, want %q", v.Source, "aisstream.io")
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-03-02T14:30:00Z")
	if !v.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp = %v, want %v", v.Timestamp, expectedTime)
	}
}

func TestParseAISMessage_NonPositionReport(t *testing.T) {
	// A non-PositionReport message type should still parse metadata but have zero Speed/Heading.
	raw := []byte(`{
		"MessageType": "StaticDataReport",
		"MetaData": {
			"MMSI": 123456789,
			"ShipName": "USNS MERCY",
			"latitude": 34.0522,
			"longitude": -118.2437,
			"time_utc": "2026-03-02T15:00:00Z"
		},
		"Message": {
			"StaticDataReport": {}
		}
	}`)

	v, err := parseAISMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.MMSI != "123456789" {
		t.Errorf("MMSI = %q, want %q", v.MMSI, "123456789")
	}
	if v.Name != "USNS MERCY" {
		t.Errorf("Name = %q, want %q", v.Name, "USNS MERCY")
	}
	if v.Speed != 0 {
		t.Errorf("Speed = %f, want 0", v.Speed)
	}
	if v.Heading != 0 {
		t.Errorf("Heading = %f, want 0", v.Heading)
	}
}

func TestParseAISMessage_InvalidJSON(t *testing.T) {
	raw := []byte(`{invalid json`)
	_, err := parseAISMessage(raw)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseAISMessage_MMSIAsString(t *testing.T) {
	// Verify MMSI is converted from numeric JSON to string.
	raw := []byte(`{
		"MessageType": "PositionReport",
		"MetaData": {
			"MMSI": 1,
			"ShipName": "TEST",
			"latitude": 0,
			"longitude": 0,
			"time_utc": "2026-01-01T00:00:00Z"
		},
		"Message": {
			"PositionReport": {
				"Sog": 0,
				"TrueHeading": 0
			}
		}
	}`)

	v, err := parseAISMessage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.MMSI != "1" {
		t.Errorf("MMSI = %q, want %q", v.MMSI, "1")
	}
}

func TestFilterMilitaryVessels(t *testing.T) {
	vessels := []models.Vessel{
		{MMSI: "1", Name: "USS NIMITZ"},
		{MMSI: "2", Name: "USNS MERCY"},
		{MMSI: "3", Name: "USCGC HAMILTON"},
		{MMSI: "4", Name: "CARGO SHIP ONE"},
		{MMSI: "5", Name: "TANKER PACIFIC"},
		{MMSI: "6", Name: "USS ABRAHAM LINCOLN"},
	}

	military := FilterMilitaryVessels(vessels)

	if len(military) != 4 {
		t.Fatalf("got %d military vessels, want 4", len(military))
	}

	expectedNames := map[string]bool{
		"USS NIMITZ":           true,
		"USNS MERCY":          true,
		"USCGC HAMILTON":      true,
		"USS ABRAHAM LINCOLN": true,
	}

	for _, v := range military {
		if !expectedNames[v.Name] {
			t.Errorf("unexpected vessel in results: %q", v.Name)
		}
	}
}

func TestFilterMilitaryVessels_Empty(t *testing.T) {
	military := FilterMilitaryVessels(nil)
	if len(military) != 0 {
		t.Errorf("got %d vessels, want 0 for nil input", len(military))
	}

	military = FilterMilitaryVessels([]models.Vessel{})
	if len(military) != 0 {
		t.Errorf("got %d vessels, want 0 for empty input", len(military))
	}
}

func TestFilterMilitaryVessels_NoMilitary(t *testing.T) {
	vessels := []models.Vessel{
		{MMSI: "1", Name: "CARGO SHIP"},
		{MMSI: "2", Name: "TANKER ATLANTIC"},
	}

	military := FilterMilitaryVessels(vessels)
	if len(military) != 0 {
		t.Errorf("got %d vessels, want 0", len(military))
	}
}

func TestFilterMilitaryVessels_CaseInsensitive(t *testing.T) {
	vessels := []models.Vessel{
		{MMSI: "1", Name: "uss nimitz"},
		{MMSI: "2", Name: "Usns Mercy"},
		{MMSI: "3", Name: "Uscgc Hamilton"},
	}

	military := FilterMilitaryVessels(vessels)
	if len(military) != 3 {
		t.Fatalf("got %d military vessels, want 3 (case-insensitive matching)", len(military))
	}
}

func TestFilterMilitaryVessels_SetsBranch(t *testing.T) {
	vessels := []models.Vessel{
		{MMSI: "1", Name: "USS NIMITZ"},
		{MMSI: "2", Name: "USNS MERCY"},
		{MMSI: "3", Name: "USCGC HAMILTON"},
	}

	military := FilterMilitaryVessels(vessels)

	branchMap := make(map[string]string)
	for _, v := range military {
		branchMap[v.Name] = v.Branch
	}

	if branchMap["USS NIMITZ"] != "US Navy" {
		t.Errorf("USS Branch = %q, want %q", branchMap["USS NIMITZ"], "US Navy")
	}
	if branchMap["USNS MERCY"] != "US Navy" {
		t.Errorf("USNS Branch = %q, want %q", branchMap["USNS MERCY"], "US Navy")
	}
	if branchMap["USCGC HAMILTON"] != "US Coast Guard" {
		t.Errorf("USCGC Branch = %q, want %q", branchMap["USCGC HAMILTON"], "US Coast Guard")
	}
}
