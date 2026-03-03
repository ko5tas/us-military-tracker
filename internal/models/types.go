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
	Tag         string    `json:"tag,omitempty"`
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
	Summary   string     `json:"summary,omitempty"`
}

// EnrichedAsset is the AI council's output for a single asset.
type EnrichedAsset struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Branch      string `json:"branch"`
	Mission     string `json:"mission"`
	Description string `json:"description"`
	Confidence  string `json:"confidence"`
	Agreement   int    `json:"agreement"`
	TotalVotes  int    `json:"total_votes"`
}

// CouncilResult holds the full output of one council cycle.
type CouncilResult struct {
	Assets    []EnrichedAsset `json:"assets"`
	Summary   string          `json:"summary"`
	Chairman  string          `json:"chairman"`
	Score     float64         `json:"score"`
	Timestamp time.Time       `json:"timestamp"`
}
