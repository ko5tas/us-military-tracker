package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGDELT_Success(t *testing.T) {
	resp := `{
		"features": [
			{
				"properties": {
					"name": "Military exercise reported near Baltic Sea",
					"html": "https://example.com/article",
					"urlpubtimeseq": "20260301120000"
				},
				"geometry": {
					"coordinates": [12.5683, 55.6761]
				}
			},
			{
				"properties": {
					"name": "Naval patrol spotted in South China Sea",
					"html": "https://example.com/article2",
					"urlpubtimeseq": "20260301140000"
				},
				"geometry": {
					"coordinates": [114.1694, 22.3193]
				}
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}))
	defer srv.Close()

	events, err := fetchGDELT(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetchGDELT returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	e := events[0]
	if e.Title != "Military exercise reported near Baltic Sea" {
		t.Errorf("Title: got %q, want %q", e.Title, "Military exercise reported near Baltic Sea")
	}
	if e.URL != "https://example.com/article" {
		t.Errorf("URL: got %q, want %q", e.URL, "https://example.com/article")
	}
	if e.Source != "gdelt" {
		t.Errorf("Source: got %q, want %q", e.Source, "gdelt")
	}
	// GeoJSON coordinates are [lon, lat]
	if e.Lat != 55.6761 {
		t.Errorf("Lat: got %v, want %v", e.Lat, 55.6761)
	}
	if e.Lon != 12.5683 {
		t.Errorf("Lon: got %v, want %v", e.Lon, 12.5683)
	}
	if e.Date != "2026-03-01" {
		t.Errorf("Date: got %q, want %q", e.Date, "2026-03-01")
	}
	if e.Type != "military_event" {
		t.Errorf("Type: got %q, want %q", e.Type, "military_event")
	}

	e2 := events[1]
	if e2.Lat != 22.3193 {
		t.Errorf("Event 2 Lat: got %v, want %v", e2.Lat, 22.3193)
	}
	if e2.Lon != 114.1694 {
		t.Errorf("Event 2 Lon: got %v, want %v", e2.Lon, 114.1694)
	}
}

func TestFetchGDELT_EmptyFeatures(t *testing.T) {
	resp := `{"features": []}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}))
	defer srv.Close()

	events, err := fetchGDELT(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetchGDELT returned error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestFetchGDELT_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchGDELT(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for server error response, got nil")
	}
}

func TestFetchACLED_Success(t *testing.T) {
	resp := `{
		"data": [
			{
				"event_id_cnty": "USA12345",
				"event_type": "Battles",
				"notes": "Military engagement in the region",
				"latitude": "55.6761",
				"longitude": "12.5683",
				"event_date": "2026-03-01",
				"source": "Reuters"
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key is passed
		key := r.URL.Query().Get("key")
		if key != "test-api-key" {
			t.Errorf("expected API key 'test-api-key', got %q", key)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}))
	defer srv.Close()

	events, err := fetchACLED(context.Background(), "test-api-key", srv.URL)
	if err != nil {
		t.Fatalf("fetchACLED returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.ID != "USA12345" {
		t.Errorf("ID: got %q, want %q", e.ID, "USA12345")
	}
	if e.Type != "Battles" {
		t.Errorf("Type: got %q, want %q", e.Type, "Battles")
	}
	if e.Title != "Battles" {
		t.Errorf("Title: got %q, want %q", e.Title, "Battles")
	}
	if e.Description != "Military engagement in the region" {
		t.Errorf("Description: got %q, want %q", e.Description, "Military engagement in the region")
	}
	if e.Lat != 55.6761 {
		t.Errorf("Lat: got %v, want %v", e.Lat, 55.6761)
	}
	if e.Lon != 12.5683 {
		t.Errorf("Lon: got %v, want %v", e.Lon, 12.5683)
	}
	if e.Date != "2026-03-01" {
		t.Errorf("Date: got %q, want %q", e.Date, "2026-03-01")
	}
	if e.Source != "acled" {
		t.Errorf("Source: got %q, want %q", e.Source, "acled")
	}
}

func TestFetchACLED_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchACLED(context.Background(), "test-key", srv.URL)
	if err == nil {
		t.Fatal("expected error for server error response, got nil")
	}
}

func TestCollectEvents_SkipsACLEDWhenKeyEmpty(t *testing.T) {
	gdeltResp := `{
		"features": [
			{
				"properties": {
					"name": "Test event",
					"html": "https://example.com",
					"urlpubtimeseq": "20260301120000"
				},
				"geometry": {
					"coordinates": [10.0, 20.0]
				}
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(gdeltResp))
	}))
	defer srv.Close()

	// Override the default GDELT URL for testing
	origGDELT := gdeltBaseURL
	gdeltBaseURL = srv.URL
	defer func() { gdeltBaseURL = origGDELT }()

	events, err := CollectEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("CollectEvents returned error: %v", err)
	}

	// Should only have GDELT events, ACLED skipped
	if len(events) != 1 {
		t.Fatalf("expected 1 event (GDELT only), got %d", len(events))
	}
	if events[0].Source != "gdelt" {
		t.Errorf("Source: got %q, want %q", events[0].Source, "gdelt")
	}
}

func TestCollectEvents_GDELTFailureStillReturnsACLED(t *testing.T) {
	// GDELT server that fails
	gdeltSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer gdeltSrv.Close()

	// ACLED server that succeeds
	acledResp := `{
		"data": [
			{
				"event_id_cnty": "TEST001",
				"event_type": "Battles",
				"notes": "Test conflict",
				"latitude": "30.0",
				"longitude": "40.0",
				"event_date": "2026-03-01",
				"source": "TestSource"
			}
		]
	}`
	acledSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(acledResp))
	}))
	defer acledSrv.Close()

	origGDELT := gdeltBaseURL
	origACLED := acledBaseURL
	gdeltBaseURL = gdeltSrv.URL
	acledBaseURL = acledSrv.URL
	defer func() {
		gdeltBaseURL = origGDELT
		acledBaseURL = origACLED
	}()

	events, err := CollectEvents(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("CollectEvents returned error: %v", err)
	}

	// GDELT failed but ACLED should still be present
	if len(events) != 1 {
		t.Fatalf("expected 1 event (ACLED only), got %d", len(events))
	}
	if events[0].Source != "acled" {
		t.Errorf("Source: got %q, want %q", events[0].Source, "acled")
	}
	if events[0].ID != "TEST001" {
		t.Errorf("ID: got %q, want %q", events[0].ID, "TEST001")
	}
}

func TestCollectEvents_BothSourcesCombined(t *testing.T) {
	gdeltResp := `{
		"features": [
			{
				"properties": {
					"name": "GDELT event",
					"html": "https://example.com/gdelt",
					"urlpubtimeseq": "20260301120000"
				},
				"geometry": {
					"coordinates": [10.0, 20.0]
				}
			}
		]
	}`
	gdeltSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(gdeltResp))
	}))
	defer gdeltSrv.Close()

	acledResp := `{
		"data": [
			{
				"event_id_cnty": "ACLED001",
				"event_type": "Battles",
				"notes": "ACLED event",
				"latitude": "30.0",
				"longitude": "40.0",
				"event_date": "2026-03-01",
				"source": "TestSource"
			}
		]
	}`
	acledSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(acledResp))
	}))
	defer acledSrv.Close()

	origGDELT := gdeltBaseURL
	origACLED := acledBaseURL
	gdeltBaseURL = gdeltSrv.URL
	acledBaseURL = acledSrv.URL
	defer func() {
		gdeltBaseURL = origGDELT
		acledBaseURL = origACLED
	}()

	events, err := CollectEvents(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("CollectEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events (combined), got %d", len(events))
	}

	// Check we have one from each source
	sources := map[string]bool{}
	for _, e := range events {
		sources[e.Source] = true
	}
	if !sources["gdelt"] {
		t.Error("expected a gdelt event in results")
	}
	if !sources["acled"] {
		t.Error("expected an acled event in results")
	}
}
