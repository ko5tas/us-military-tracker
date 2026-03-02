package kml

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

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
			name := fmt.Sprintf("%s (%s)", a.Callsign, a.Type)
			desc := fmt.Sprintf("Branch: %s\nAltitude: %d ft\nSpeed: %.1f kts\nHeading: %.1f\nMission: %s",
				a.Branch, a.Altitude, a.Speed, a.Heading, a.Mission)
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
		desc := fmt.Sprintf("Type: %s\nMMSI: %s\nSpeed: %.1f kts\nHeading: %.1f",
			v.Type, v.MMSI, v.Speed, v.Heading)
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
