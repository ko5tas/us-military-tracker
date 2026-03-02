package collectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultAircraftSources(t *testing.T) {
	sources := DefaultAircraftSources()

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	expected := map[string]string{
		"airplanes.live": "https://api.airplanes.live/v2/mil",
		"adsb.one":       "https://api.adsb.one/v2/mil",
		"adsb.lol":       "https://api.adsb.lol/v2/mil",
	}

	for _, src := range sources {
		wantURL, ok := expected[src.Name]
		if !ok {
			t.Errorf("unexpected source name: %q", src.Name)
			continue
		}
		if src.URL != wantURL {
			t.Errorf("source %q: got URL %q, want %q", src.Name, src.URL, wantURL)
		}
	}
}

func TestCollectAircraft(t *testing.T) {
	// Mock server returning a valid ADS-B response with alt_baro as float64
	// and a padded callsign.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ac": []map[string]interface{}{
				{
					"hex":      "AE1234",
					"flight":   "EVAC01  ",
					"t":        "C17",
					"lat":      49.4389,
					"lon":      7.6009,
					"alt_baro": 28000.0,
					"gs":       450.0,
					"track":    270.0,
					"squawk":   "1200",
					"seen":     1.5,
				},
			},
			"total": 1,
			"now":   1709391000.0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	sources := []AircraftSource{
		{Name: "test-source", URL: srv.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft returned error: %v", err)
	}

	if len(aircraft) != 1 {
		t.Fatalf("expected 1 aircraft, got %d", len(aircraft))
	}

	ac := aircraft[0]
	if ac.Hex != "AE1234" {
		t.Errorf("Hex: got %q, want %q", ac.Hex, "AE1234")
	}
	if ac.Callsign != "EVAC01" {
		t.Errorf("Callsign: got %q, want %q (should be trimmed)", ac.Callsign, "EVAC01")
	}
	if ac.Type != "C17" {
		t.Errorf("Type: got %q, want %q", ac.Type, "C17")
	}
	if ac.Lat != 49.4389 {
		t.Errorf("Lat: got %v, want %v", ac.Lat, 49.4389)
	}
	if ac.Lon != 7.6009 {
		t.Errorf("Lon: got %v, want %v", ac.Lon, 7.6009)
	}
	if ac.Altitude != 28000 {
		t.Errorf("Altitude: got %d, want %d", ac.Altitude, 28000)
	}
	if ac.Speed != 450.0 {
		t.Errorf("Speed: got %v, want %v", ac.Speed, 450.0)
	}
	if ac.Heading != 270.0 {
		t.Errorf("Heading: got %v, want %v", ac.Heading, 270.0)
	}
	if ac.Squawk != "1200" {
		t.Errorf("Squawk: got %q, want %q", ac.Squawk, "1200")
	}
	if ac.Source != "test-source" {
		t.Errorf("Source: got %q, want %q", ac.Source, "test-source")
	}
	if ac.SeenAgo != 1.5 {
		t.Errorf("SeenAgo: got %v, want %v", ac.SeenAgo, 1.5)
	}
}

func TestCollectAircraftAltBaroGround(t *testing.T) {
	// Test that alt_baro = "ground" is handled (altitude should be 0).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ac": []map[string]interface{}{
				{
					"hex":      "AE5678",
					"flight":   "TEST01",
					"t":        "C130",
					"lat":      38.0,
					"lon":      -77.0,
					"alt_baro": "ground",
					"gs":       0.0,
					"track":    0.0,
					"squawk":   "0000",
					"seen":     0.5,
				},
			},
			"total": 1,
			"now":   1709391000.0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	sources := []AircraftSource{
		{Name: "ground-test", URL: srv.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft returned error: %v", err)
	}

	if len(aircraft) != 1 {
		t.Fatalf("expected 1 aircraft, got %d", len(aircraft))
	}

	if aircraft[0].Altitude != 0 {
		t.Errorf("Altitude for ground: got %d, want 0", aircraft[0].Altitude)
	}
}

func TestCollectAircraftParallelSources(t *testing.T) {
	// Two servers returning different aircraft.
	makeHandler := func(hex, callsign string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"ac": []map[string]interface{}{
					{
						"hex":      hex,
						"flight":   callsign,
						"t":        "F16",
						"lat":      40.0,
						"lon":      -74.0,
						"alt_baro": 35000.0,
						"gs":       500.0,
						"track":    180.0,
						"squawk":   "7700",
						"seen":     2.0,
					},
				},
				"total": 1,
				"now":   1709391000.0,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})
	}

	srv1 := httptest.NewServer(makeHandler("AA1111", "VIPER01"))
	defer srv1.Close()
	srv2 := httptest.NewServer(makeHandler("BB2222", "VIPER02"))
	defer srv2.Close()

	sources := []AircraftSource{
		{Name: "source-1", URL: srv1.URL},
		{Name: "source-2", URL: srv2.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft returned error: %v", err)
	}

	if len(aircraft) != 2 {
		t.Fatalf("expected 2 aircraft from 2 sources, got %d", len(aircraft))
	}

	hexes := map[string]bool{}
	for _, ac := range aircraft {
		hexes[ac.Hex] = true
	}
	if !hexes["AA1111"] || !hexes["BB2222"] {
		t.Errorf("expected aircraft AA1111 and BB2222, got hexes: %v", hexes)
	}
}

func TestCollectAircraftSourceFailure(t *testing.T) {
	// One good server and one bad server (returns 500).
	goodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ac": []map[string]interface{}{
				{
					"hex":      "CC3333",
					"flight":   "GOOD01",
					"t":        "B52",
					"lat":      35.0,
					"lon":      -80.0,
					"alt_baro": 40000.0,
					"gs":       500.0,
					"track":    90.0,
					"squawk":   "1200",
					"seen":     1.0,
				},
			},
			"total": 1,
			"now":   1709391000.0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	goodSrv := httptest.NewServer(goodHandler)
	defer goodSrv.Close()
	badSrv := httptest.NewServer(badHandler)
	defer badSrv.Close()

	sources := []AircraftSource{
		{Name: "good-source", URL: goodSrv.URL},
		{Name: "bad-source", URL: badSrv.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft should not return error when one source fails: %v", err)
	}

	if len(aircraft) != 1 {
		t.Fatalf("expected 1 aircraft (from good source), got %d", len(aircraft))
	}

	if aircraft[0].Hex != "CC3333" {
		t.Errorf("expected aircraft from good source, got hex %q", aircraft[0].Hex)
	}
}

func TestCollectAircraftAllSourcesFail(t *testing.T) {
	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	srv := httptest.NewServer(badHandler)
	defer srv.Close()

	sources := []AircraftSource{
		{Name: "bad-1", URL: srv.URL},
		{Name: "bad-2", URL: srv.URL},
	}

	aircraft, err := CollectAircraft(context.Background(), sources)
	if err != nil {
		t.Fatalf("CollectAircraft should not error even if all sources fail: %v", err)
	}

	if len(aircraft) != 0 {
		t.Errorf("expected 0 aircraft when all sources fail, got %d", len(aircraft))
	}
}

func TestDeduplicateAircraft(t *testing.T) {
	aircraft := []CollectedAircraft{
		{SeenAgo: 5.0},
		{SeenAgo: 1.0},
		{SeenAgo: 3.0},
	}
	// Set the same hex for all three to test dedup.
	aircraft[0].Hex = "AE1234"
	aircraft[1].Hex = "AE1234"
	aircraft[2].Hex = "AE1234"

	// Set different sources to verify which one is kept.
	aircraft[0].Source = "source-a"
	aircraft[1].Source = "source-b"
	aircraft[2].Source = "source-c"

	result := DeduplicateAircraft(aircraft)

	if len(result) != 1 {
		t.Fatalf("expected 1 aircraft after dedup, got %d", len(result))
	}

	// The one with SeenAgo=1.0 (lowest) should be kept.
	if result[0].SeenAgo != 1.0 {
		t.Errorf("expected SeenAgo 1.0 (most recent), got %v", result[0].SeenAgo)
	}
	if result[0].Source != "source-b" {
		t.Errorf("expected source-b (most recent), got %q", result[0].Source)
	}
}

func TestDeduplicateAircraftMultipleHexes(t *testing.T) {
	aircraft := []CollectedAircraft{
		{SeenAgo: 2.0},
		{SeenAgo: 1.0},
		{SeenAgo: 3.0},
		{SeenAgo: 0.5},
	}
	aircraft[0].Hex = "AA1111"
	aircraft[1].Hex = "AA1111"
	aircraft[2].Hex = "BB2222"
	aircraft[3].Hex = "BB2222"

	aircraft[0].Source = "src-a"
	aircraft[1].Source = "src-b"
	aircraft[2].Source = "src-c"
	aircraft[3].Source = "src-d"

	result := DeduplicateAircraft(aircraft)

	if len(result) != 2 {
		t.Fatalf("expected 2 aircraft after dedup, got %d", len(result))
	}

	byHex := map[string]CollectedAircraft{}
	for _, ac := range result {
		byHex[ac.Hex] = ac
	}

	if aa, ok := byHex["AA1111"]; !ok {
		t.Error("missing AA1111 in dedup result")
	} else if aa.SeenAgo != 1.0 {
		t.Errorf("AA1111: expected SeenAgo 1.0, got %v", aa.SeenAgo)
	}

	if bb, ok := byHex["BB2222"]; !ok {
		t.Error("missing BB2222 in dedup result")
	} else if bb.SeenAgo != 0.5 {
		t.Errorf("BB2222: expected SeenAgo 0.5, got %v", bb.SeenAgo)
	}
}

func TestDeduplicateAircraftEmpty(t *testing.T) {
	result := DeduplicateAircraft(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 aircraft for nil input, got %d", len(result))
	}

	result = DeduplicateAircraft([]CollectedAircraft{})
	if len(result) != 0 {
		t.Errorf("expected 0 aircraft for empty input, got %d", len(result))
	}
}

func TestCollectAircraftContextCanceled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ac":    []map[string]interface{}{},
			"total": 0,
			"now":   1709391000.0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	sources := []AircraftSource{
		{Name: "canceled-source", URL: srv.URL},
	}

	aircraft, err := CollectAircraft(ctx, sources)
	// With a canceled context, the HTTP request should fail.
	// We should still get no error from CollectAircraft (it logs warnings),
	// but 0 aircraft.
	if err != nil {
		t.Fatalf("CollectAircraft should not return error on canceled context: %v", err)
	}
	if len(aircraft) != 0 {
		t.Errorf("expected 0 aircraft with canceled context, got %d", len(aircraft))
	}
}
