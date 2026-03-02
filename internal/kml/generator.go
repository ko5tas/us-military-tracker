package kml

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// icaoTypeLookup maps ICAO type designator codes (and common short codes) to
// human-readable aircraft names. The map is checked case-insensitively by
// resolveTypeName.
var icaoTypeLookup = map[string]string{
	// Heavy transport / tanker
	"C17":   "C-17 Globemaster III",
	"C17A":  "C-17 Globemaster III",
	"C5":    "C-5M Super Galaxy",
	"C5M":   "C-5M Super Galaxy",
	"C130":  "C-130 Hercules",
	"C130J": "C-130J Super Hercules",
	"C130H": "C-130H Hercules",
	"C2":    "C-2A Greyhound",
	"C2A":   "C-2A Greyhound",
	"C40":   "C-40A Clipper",
	"C40A":  "C-40A Clipper",
	"C32":   "C-32A (Boeing 757-200)",
	"C12":   "C-12 Huron",
	"C26":   "C-26 Metroliner",
	"C37":   "C-37A Gulfstream V",
	"KC135": "KC-135 Stratotanker",
	"K35R":  "KC-135 Stratotanker",
	"KC10":  "KC-10 Extender",
	"KC46":  "KC-46A Pegasus",
	"KC46A": "KC-46A Pegasus",
	"K46":   "KC-46A Pegasus",

	// Bombers
	"B52":  "B-52 Stratofortress",
	"B52H": "B-52H Stratofortress",
	"B1":   "B-1B Lancer",
	"B1B":  "B-1B Lancer",
	"B2":   "B-2A Spirit",
	"B2A":  "B-2A Spirit",
	"B21":  "B-21 Raider",

	// Fighters
	"F16":  "F-16 Fighting Falcon",
	"F16C": "F-16C Fighting Falcon",
	"F15":  "F-15 Eagle",
	"F15C": "F-15C Eagle",
	"F15E": "F-15E Strike Eagle",
	"F22":  "F-22A Raptor",
	"F22A": "F-22A Raptor",
	"F35":  "F-35 Lightning II",
	"F35A": "F-35A Lightning II",
	"F35B": "F-35B Lightning II",
	"F35C": "F-35C Lightning II",
	"F18":  "F/A-18 Hornet",
	"FA18": "F/A-18 Hornet",
	"F18E": "F/A-18E Super Hornet",
	"F18F": "F/A-18F Super Hornet",

	// ISR / Patrol / EW
	"P8":   "P-8A Poseidon",
	"P8A":  "P-8A Poseidon",
	"P3":   "P-3C Orion",
	"P3C":  "P-3C Orion",
	"E3":   "E-3 Sentry (AWACS)",
	"E3A":  "E-3A Sentry (AWACS)",
	"E3B":  "E-3B Sentry (AWACS)",
	"E6":   "E-6B Mercury (TACAMO)",
	"E6B":  "E-6B Mercury (TACAMO)",
	"E2":   "E-2 Hawkeye",
	"E2C":  "E-2C Hawkeye",
	"E2D":  "E-2D Advanced Hawkeye",
	"E8":   "E-8C JSTARS",
	"E8C":  "E-8C JSTARS",
	"RC135": "RC-135 Rivet Joint",
	"R135":  "RC-135 Rivet Joint",
	"RC26":  "RC-26B Condor",
	"U2":    "U-2 Dragon Lady",
	"U2S":   "U-2S Dragon Lady",
	"EP3":   "EP-3E Aries II",

	// UAVs / RPAs
	"MQ9":  "MQ-9 Reaper",
	"MQ9A": "MQ-9A Reaper",
	"RQ4":  "RQ-4 Global Hawk",
	"RQ4B": "RQ-4B Global Hawk",
	"MQ4":  "MQ-4C Triton",
	"MQ4C": "MQ-4C Triton",
	"MQ1":  "MQ-1 Predator",

	// Tiltrotor / Helicopter
	"V22":  "V-22 Osprey",
	"MV22": "MV-22B Osprey",
	"CV22": "CV-22B Osprey",
	"H60":  "H-60 Black Hawk",
	"UH60": "UH-60 Black Hawk",
	"MH60": "MH-60 Seahawk",
	"SH60": "SH-60 Seahawk",
	"H47":  "CH-47 Chinook",
	"CH47": "CH-47 Chinook",
	"H53":  "CH-53 Sea Stallion",
	"CH53": "CH-53E Super Stallion",
	"CH53K": "CH-53K King Stallion",
	"AH64": "AH-64 Apache",
	"H64":  "AH-64 Apache",
	"AH1":  "AH-1Z Viper",
	"AH1Z": "AH-1Z Viper",

	// Allied / NATO common types
	"A400": "A400M Atlas",
	"A400M": "A400M Atlas",
	"C295": "C-295",
	"EUFI": "Eurofighter Typhoon",
	"RFAL": "Rafale",
	"F2TH": "Dassault Falcon 2000",
	"E550": "E-550A CAEW",

	// Trainer / Special
	"T38":  "T-38 Talon",
	"T6":   "T-6 Texan II",
	"T45":  "T-45 Goshawk",
	"AC130": "AC-130J Ghostrider",
	"MC130": "MC-130J Commando II",
	"HC130": "HC-130J Combat King II",
	"MC12":  "MC-12W Liberty",
}

// resolveTypeName returns a human-readable name for the given ICAO type code.
// It performs a case-insensitive lookup after stripping hyphens and spaces.
// If no match is found, it returns the original type string.
func resolveTypeName(typeCode string) string {
	if typeCode == "" {
		return "Unknown Type"
	}
	// Normalize: uppercase, strip hyphens and spaces
	key := strings.ToUpper(typeCode)
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")

	if name, ok := icaoTypeLookup[key]; ok {
		return name
	}
	// If the raw code already looks like a full name (contains a space), return as-is
	if strings.Contains(typeCode, " ") {
		return typeCode
	}
	// Return original code with a note that it was not resolved
	return typeCode
}

// KML is the root element of a KML document.
type KML struct {
	XMLName  xml.Name `xml:"kml"`
	XMLNS    string   `xml:"xmlns,attr"`
	Document Document `xml:"Document"`
}

// Document represents a KML Document element.
type Document struct {
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	Folders     []Folder `xml:"Folder"`
}

// Folder represents a KML Folder element.
type Folder struct {
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark,omitempty"`
	Folders    []Folder    `xml:"Folder,omitempty"`
}

// Placemark represents a KML Placemark element.
type Placemark struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Point       Point  `xml:"Point"`
}

// Point represents a KML Point element.
type Point struct {
	Coordinates string `xml:"coordinates"`
}

// Generate creates a KML file at the given outputPath from the collected data.
// The description includes the timestamp, chairman name, and confidence score.
func Generate(outputPath string, data *models.CollectedData, chairman string, score float64) error {
	doc := KML{
		XMLNS: "http://www.opengis.net/kml/2.2",
		Document: Document{
			Name: "US Military Tracker",
			Description: fmt.Sprintf("Generated: %s | Chairman: %s | Score: %.2f",
				data.Timestamp.Format(time.RFC3339), chairman, score),
		},
	}

	var folders []Folder

	// Aircraft folder (grouped by branch)
	if len(data.Aircraft) > 0 {
		folders = append(folders, buildAircraftFolder(data.Aircraft))
	}

	// Vessels folder
	if len(data.Vessels) > 0 {
		folders = append(folders, buildVesselsFolder(data.Vessels))
	}

	// Bases folder
	if len(data.Bases) > 0 {
		folders = append(folders, buildBasesFolder(data.Bases))
	}

	// Events folder
	if len(data.Events) > 0 {
		folders = append(folders, buildEventsFolder(data.Events))
	}

	// News folder (only geolocated items)
	if len(data.News) > 0 {
		newsFolder := buildNewsFolder(data.News)
		if len(newsFolder.Placemarks) > 0 || len(newsFolder.Folders) > 0 {
			folders = append(folders, newsFolder)
		}
	}

	// Intelligence Summary folder (from AI council analysis)
	if data.Summary != "" {
		folders = append(folders, buildIntelligenceFolder(data.Summary))
	}

	doc.Document.Folders = folders

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create KML file: %w", err)
	}
	defer f.Close()

	// Write XML header
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("write XML header: %w", err)
	}

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")

	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encode KML: %w", err)
	}

	return nil
}

// buildAircraftFolder creates the Aircraft folder with sub-folders grouped by branch.
func buildAircraftFolder(aircraft []models.Aircraft) Folder {
	// Group aircraft by branch
	branchMap := make(map[string][]models.Aircraft)
	for _, a := range aircraft {
		branch := a.Branch
		if branch == "" {
			branch = "Unknown"
		}
		branchMap[branch] = append(branchMap[branch], a)
	}

	var subFolders []Folder
	for branch, planes := range branchMap {
		var placemarks []Placemark
		for _, a := range planes {
			// Resolve display name: prefer callsign, fall back to hex code
			displayCallsign := a.Callsign
			if displayCallsign == "" {
				displayCallsign = a.Hex
			}

			// Resolve human-readable type name from ICAO code
			typeName := resolveTypeName(a.Type)

			name := fmt.Sprintf("%s (%s)", displayCallsign, typeName)

			// Build description with only non-empty fields
			var descParts []string
			descParts = append(descParts, fmt.Sprintf("Type: %s", typeName))
			if a.Callsign != "" {
				descParts = append(descParts, fmt.Sprintf("Callsign: %s", a.Callsign))
			}
			descParts = append(descParts, fmt.Sprintf("Hex: %s", a.Hex))
			if a.Branch != "" {
				descParts = append(descParts, fmt.Sprintf("Branch: %s", a.Branch))
			}
			if a.Mission != "" {
				descParts = append(descParts, fmt.Sprintf("Mission: %s", a.Mission))
			}
			descParts = append(descParts, fmt.Sprintf("Altitude: %d ft", a.Altitude))
			descParts = append(descParts, fmt.Sprintf("Speed: %.1f kts", a.Speed))
			descParts = append(descParts, fmt.Sprintf("Heading: %.1f", a.Heading))
			if a.Source != "" {
				descParts = append(descParts, fmt.Sprintf("Source: %s", a.Source))
			}

			desc := strings.Join(descParts, "\n")

			placemarks = append(placemarks, Placemark{
				Name:        name,
				Description: desc,
				Point: Point{
					Coordinates: fmt.Sprintf("%v,%v,%d", a.Lon, a.Lat, a.Altitude),
				},
			})
		}
		subFolders = append(subFolders, Folder{
			Name:       branch,
			Placemarks: placemarks,
		})
	}

	return Folder{
		Name:    fmt.Sprintf("Aircraft (%d tracked)", len(aircraft)),
		Folders: subFolders,
	}
}

// buildVesselsFolder creates the Vessels folder.
func buildVesselsFolder(vessels []models.Vessel) Folder {
	var placemarks []Placemark
	for _, v := range vessels {
		var descParts []string
		if v.Branch != "" {
			descParts = append(descParts, fmt.Sprintf("Branch: %s", v.Branch))
		}
		if v.Class != "" {
			descParts = append(descParts, fmt.Sprintf("Status: %s", v.Class))
		}
		if v.Type != "" && v.Type != "carrier_strike_group" {
			descParts = append(descParts, fmt.Sprintf("Type: %s", v.Type))
		} else if v.Type == "carrier_strike_group" {
			descParts = append(descParts, "Type: Aircraft Carrier Strike Group")
		}
		if v.MMSI != "" {
			descParts = append(descParts, fmt.Sprintf("MMSI: %s", v.MMSI))
		}
		if v.Speed > 0 {
			descParts = append(descParts, fmt.Sprintf("Speed: %.1f kts", v.Speed))
		}
		if v.Heading > 0 {
			descParts = append(descParts, fmt.Sprintf("Heading: %.1f", v.Heading))
		}
		if v.Source != "" {
			descParts = append(descParts, fmt.Sprintf("Source: %s", v.Source))
		}
		desc := strings.Join(descParts, "\n")

		placemarks = append(placemarks, Placemark{
			Name:        v.Name,
			Description: desc,
			Point: Point{
				Coordinates: fmt.Sprintf("%v,%v,0", v.Lon, v.Lat),
			},
		})
	}

	return Folder{
		Name:       fmt.Sprintf("Vessels (%d tracked)", len(vessels)),
		Placemarks: placemarks,
	}
}

// buildBasesFolder creates the Bases folder.
func buildBasesFolder(bases []models.Base) Folder {
	var placemarks []Placemark
	for _, b := range bases {
		desc := fmt.Sprintf("Branch: %s\nCountry: %s\nType: %s",
			b.Branch, b.Country, b.Type)
		placemarks = append(placemarks, Placemark{
			Name:        b.Name,
			Description: desc,
			Point: Point{
				Coordinates: fmt.Sprintf("%v,%v,0", b.Lon, b.Lat),
			},
		})
	}

	return Folder{
		Name:       fmt.Sprintf("Bases (%d tracked)", len(bases)),
		Placemarks: placemarks,
	}
}

// buildEventsFolder creates the Events folder.
func buildEventsFolder(events []models.Event) Folder {
	var placemarks []Placemark
	for _, e := range events {
		desc := fmt.Sprintf("Type: %s\nSource: %s\nDate: %s\n%s",
			e.Type, e.Source, e.Date, e.Description)
		placemarks = append(placemarks, Placemark{
			Name:        e.Title,
			Description: desc,
			Point: Point{
				Coordinates: fmt.Sprintf("%v,%v,0", e.Lon, e.Lat),
			},
		})
	}

	return Folder{
		Name:       fmt.Sprintf("Events (%d tracked)", len(events)),
		Placemarks: placemarks,
	}
}

// buildNewsFolder creates the News folder, skipping items where both Lat and Lon are 0.
func buildNewsFolder(news []models.NewsItem) Folder {
	var placemarks []Placemark
	for _, n := range news {
		// Skip items where both Lat and Lon are 0
		if n.Lat == 0 && n.Lon == 0 {
			continue
		}
		placemarks = append(placemarks, Placemark{
			Name:        n.Title,
			Description: fmt.Sprintf("Source: %s\n%s", n.Source, n.Description),
			Point: Point{
				Coordinates: fmt.Sprintf("%v,%v,0", n.Lon, n.Lat),
			},
		})
	}

	return Folder{
		Name:       fmt.Sprintf("News (%d tracked)", len(placemarks)),
		Placemarks: placemarks,
	}
}

// buildIntelligenceFolder creates a folder containing the AI council's intelligence
// summary as a single Placemark placed at 0,0 (global overview, not location-specific).
func buildIntelligenceFolder(summary string) Folder {
	return Folder{
		Name: "Intelligence Summary",
		Placemarks: []Placemark{
			{
				Name:        "AI Council Assessment",
				Description: summary,
				Point: Point{
					Coordinates: "0,0,0",
				},
			},
		},
	}
}
