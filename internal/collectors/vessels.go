package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/ko5tas/us-military-tracker/internal/models"
)

// aisMessage represents the JSON structure received from AISStream.io.
type aisMessage struct {
	MessageType string `json:"MessageType"`
	MetaData    struct {
		MMSI      int     `json:"MMSI"`
		ShipName  string  `json:"ShipName"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		TimeUTC   string  `json:"time_utc"`
	} `json:"MetaData"`
	Message json.RawMessage `json:"Message"`
}

// positionReportWrapper holds the nested PositionReport data.
type positionReportWrapper struct {
	PositionReport struct {
		Sog         float64 `json:"Sog"`
		TrueHeading float64 `json:"TrueHeading"`
	} `json:"PositionReport"`
}

// parseAISMessage parses a raw AIS JSON message from AISStream.io into a models.Vessel.
func parseAISMessage(raw []byte) (models.Vessel, error) {
	var msg aisMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return models.Vessel{}, fmt.Errorf("unmarshal AIS message: %w", err)
	}

	ts, err := time.Parse(time.RFC3339, msg.MetaData.TimeUTC)
	if err != nil {
		ts = time.Now().UTC()
	}

	v := models.Vessel{
		MMSI:      fmt.Sprintf("%d", msg.MetaData.MMSI),
		Name:      msg.MetaData.ShipName,
		Lat:       msg.MetaData.Latitude,
		Lon:       msg.MetaData.Longitude,
		Source:    "aisstream.io",
		Timestamp: ts,
	}

	// Extract speed and heading from PositionReport messages.
	if msg.MessageType == "PositionReport" {
		var pr positionReportWrapper
		if err := json.Unmarshal(msg.Message, &pr); err == nil {
			v.Speed = pr.PositionReport.Sog
			v.Heading = pr.PositionReport.TrueHeading
		}
	}

	return v, nil
}

// militaryPrefixes maps vessel name prefixes to their military branch.
var militaryPrefixes = []struct {
	Prefix string
	Branch string
}{
	{"USS ", "US Navy"},
	{"USNS ", "US Navy"},
	{"USCGC ", "US Coast Guard"},
}

// FilterMilitaryVessels returns only vessels whose names match known military
// naming patterns (USS, USNS, USCGC prefixes). It also sets the Branch field.
func FilterMilitaryVessels(vessels []models.Vessel) []models.Vessel {
	var result []models.Vessel
	for _, v := range vessels {
		upper := strings.ToUpper(v.Name)
		for _, mp := range militaryPrefixes {
			if strings.HasPrefix(upper, mp.Prefix) {
				v.Branch = mp.Branch
				result = append(result, v)
				break
			}
		}
	}
	return result
}

// subscriptionMessage is the JSON payload sent to AISStream.io to start receiving data.
type subscriptionMessage struct {
	APIKey        string     `json:"APIKey"`
	BoundingBoxes [][][2]float64 `json:"BoundingBoxes"`
}

// CollectVessels connects to AISStream.io via WebSocket, collects AIS position
// reports for the specified duration, and returns only military vessels.
func CollectVessels(ctx context.Context, apiKey string, duration time.Duration) ([]models.Vessel, error) {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "wss://stream.aisstream.io/v0/stream", nil)
	if err != nil {
		return nil, fmt.Errorf("dial AISStream: %w", err)
	}
	defer conn.CloseNow()

	// Subscribe to worldwide AIS data.
	sub := subscriptionMessage{
		APIKey: apiKey,
		BoundingBoxes: [][][2]float64{
			{{-90, -180}, {90, 180}},
		},
	}
	subJSON, err := json.Marshal(sub)
	if err != nil {
		return nil, fmt.Errorf("marshal subscription: %w", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, subJSON); err != nil {
		return nil, fmt.Errorf("send subscription: %w", err)
	}

	// Collect messages until the context expires.
	var vessels []models.Vessel
	seen := make(map[string]bool)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			// Context deadline exceeded is the normal exit path.
			if ctx.Err() != nil {
				break
			}
			log.Printf("AISStream read error: %v", err)
			break
		}

		v, err := parseAISMessage(data)
		if err != nil {
			log.Printf("parse AIS message: %v", err)
			continue
		}

		// Deduplicate by MMSI, keeping the latest position.
		if seen[v.MMSI] {
			// Update existing vessel with newer data.
			for i := range vessels {
				if vessels[i].MMSI == v.MMSI {
					vessels[i] = v
					break
				}
			}
		} else {
			seen[v.MMSI] = true
			vessels = append(vessels, v)
		}
	}

	conn.Close(websocket.StatusNormalClosure, "collection complete")

	return FilterMilitaryVessels(vessels), nil
}
