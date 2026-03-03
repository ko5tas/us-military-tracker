package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ko5tas/us-military-tracker/internal/collectors"
	"github.com/ko5tas/us-military-tracker/internal/enrichment"
	"github.com/ko5tas/us-military-tracker/internal/enrichment/providers"
	"github.com/ko5tas/us-military-tracker/internal/kml"
	"github.com/ko5tas/us-military-tracker/internal/models"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command-line flags
	evaluate := flag.Bool("evaluate", false, "run chairman evaluation mode")
	refreshStatic := flag.Bool("refresh-static", false, "refresh static data files")
	evolve := flag.Bool("evolve", false, "run model evolution")
	flag.Parse()

	if *evaluate {
		fmt.Println("evaluate: not implemented")
		return nil
	}
	if *refreshStatic {
		fmt.Println("refresh-static: not implemented")
		return nil
	}
	if *evolve {
		fmt.Println("evolve: not implemented")
		return nil
	}

	fmt.Println("US Military Tracker starting...")

	// Read API keys from environment variables
	geminiKey := os.Getenv("GEMINI_API_KEY")
	groqKey := os.Getenv("GROQ_API_KEY")
	mistralKey := os.Getenv("MISTRAL_API_KEY")
	deepseekKey := os.Getenv("DEEPSEEK_API_KEY")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	aisstreamKey := os.Getenv("AISSTREAM_API_KEY")
	gnewsKey := os.Getenv("GNEWS_API_KEY")
	acledKey := os.Getenv("ACLED_API_KEY")

	// Create a context with 10-minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// ── Collect phase ──────────────────────────────────────────────────
	fmt.Println("Starting collection phase...")

	var (
		collectedAircraft []collectors.CollectedAircraft
		vessels           []models.Vessel
		events            []models.Event
		news              []models.NewsItem
		mu                sync.Mutex
		wg                sync.WaitGroup
	)

	// Run all collectors in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		ac, err := collectors.CollectAircraft(ctx, collectors.DefaultAircraftSources())
		if err != nil {
			log.Printf("WARNING: aircraft collection failed: %v", err)
			return
		}
		ac = collectors.DeduplicateAircraft(ac)
		mu.Lock()
		collectedAircraft = ac
		mu.Unlock()
	}()

	// Vessels (collect for 30 seconds via WebSocket, skip if no key)
	if aisstreamKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := collectors.CollectVessels(ctx, aisstreamKey, 30*time.Second)
			if err != nil {
				log.Printf("WARNING: vessel collection failed: %v", err)
				return
			}
			mu.Lock()
			vessels = v
			mu.Unlock()
		}()
	} else {
		log.Println("Skipping vessel collection (no AISSTREAM_API_KEY)")
	}

	// Events
	wg.Add(1)
	go func() {
		defer wg.Done()
		ev, err := collectors.CollectEvents(ctx, acledKey)
		if err != nil {
			log.Printf("WARNING: events collection failed: %v", err)
			return
		}
		mu.Lock()
		events = ev
		mu.Unlock()
	}()

	// News
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := collectors.CollectNews(ctx, gnewsKey)
		if err != nil {
			log.Printf("WARNING: news collection failed: %v", err)
			return
		}
		mu.Lock()
		news = n
		mu.Unlock()
	}()

	wg.Wait()
	fmt.Println("Collection phase complete.")

	// Extract models.Aircraft from CollectedAircraft
	aircraft := make([]models.Aircraft, len(collectedAircraft))
	for i, ca := range collectedAircraft {
		aircraft[i] = ca.Aircraft
	}

	// Load bases from data/static/bases.json if it exists
	var bases []models.Base
	basesData, err := os.ReadFile("data/static/bases.json")
	if err == nil {
		if err := json.Unmarshal(basesData, &bases); err != nil {
			log.Printf("WARNING: failed to parse bases.json: %v", err)
		}
	}

	// Assemble CollectedData
	data := models.CollectedData{
		Aircraft:  aircraft,
		Vessels:   vessels,
		Events:    events,
		News:      news,
		Bases:     bases,
		Timestamp: time.Now().UTC(),
	}

	// Save collected data as JSON
	if err := saveJSON("data/aircraft.json", data.Aircraft); err != nil {
		log.Printf("WARNING: failed to save aircraft.json: %v", err)
	}
	if err := saveJSON("data/vessels.json", data.Vessels); err != nil {
		log.Printf("WARNING: failed to save vessels.json: %v", err)
	}
	if err := saveJSON("data/events.json", data.Events); err != nil {
		log.Printf("WARNING: failed to save events.json: %v", err)
	}
	if err := saveJSON("data/news.json", data.News); err != nil {
		log.Printf("WARNING: failed to save news.json: %v", err)
	}

	// ── Enrich phase ───────────────────────────────────────────────────
	fmt.Println("Starting enrichment phase...")

	// Initialize available AI providers (skip those without API keys)
	var members []providers.Completer

	if geminiKey != "" {
		g, err := providers.NewGemini(ctx, geminiKey, "gemini", "gemini-2.0-flash-lite")
		if err != nil {
			log.Printf("WARNING: failed to init Gemini provider: %v", err)
		} else {
			members = append(members, g)
			log.Println("Initialized provider: gemini")
		}
	}
	if groqKey != "" {
		members = append(members, providers.NewGroq(groqKey))
		log.Println("Initialized provider: groq")
	}
	if mistralKey != "" {
		members = append(members, providers.NewMistral(mistralKey))
		log.Println("Initialized provider: mistral")
	}
	if deepseekKey != "" {
		members = append(members, providers.NewDeepSeek(deepseekKey))
		log.Println("Initialized provider: deepseek")
	}
	if openrouterKey != "" {
		members = append(members, providers.NewOpenRouter(openrouterKey))
		log.Println("Initialized provider: openrouter")
	}
	if openaiKey != "" {
		members = append(members, providers.NewChatGPT(openaiKey))
		log.Println("Initialized provider: chatgpt")
	}
	if anthropicKey != "" {
		members = append(members, providers.NewClaude(anthropicKey))
		log.Println("Initialized provider: claude")
	}

	// Add local Ollama as a council member (always available on the runner)
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost != "" {
		members = append(members, providers.NewOllama())
		log.Println("Initialized provider: ollama (local)")
	}

	log.Printf("Total AI providers available: %d", len(members))

	chairman := "none"
	score := 0.0

	if len(members) > 0 {
		// Build a concise summary prompt instead of sending raw JSON
		userPrompt := buildDataSummary(data)
		systemPrompt := buildStructuredSystemPrompt()

		// Run council
		responses := enrichment.RunCouncil(ctx, members, systemPrompt, userPrompt)
		for _, r := range responses {
			if r.Err != nil {
				log.Printf("Council member %s FAILED (%v): %v", r.Provider, r.Latency, r.Err)
			} else {
				log.Printf("Council member %s OK (%v): %d chars", r.Provider, r.Latency, len(r.Response))
			}
		}
		successful := enrichment.SuccessfulResponses(responses)

		if len(successful) > 0 {
			// Load chairman scores
			scores, err := enrichment.LoadScores("config/chairman_scores.json")
			if err != nil {
				log.Printf("WARNING: failed to load chairman scores: %v", err)
				scores = make(enrichment.ChairmanScores)
			}

			// Select chairman
			chairman = enrichment.SelectChairman(scores)

			// If no chairman yet, use the first successful provider
			if chairman == "" {
				chairman = successful[0].Provider
			}

			// Find the chairman provider
			var chairmanProvider providers.Completer
			for _, m := range members {
				if m.Name() == chairman {
					chairmanProvider = m
					break
				}
			}

			// Try synthesis with fallback chain
			var parsed *councilJSON
			actualChairman := chairman
			synthesisPrompt := buildStructuredSynthesisPrompt(successful)
			sysPrompt := buildStructuredSystemPrompt()

			// Step 1: Try the selected chairman
			if chairmanProvider != nil {
				synthResp, synthErr := chairmanProvider.Complete(ctx, sysPrompt, synthesisPrompt)
				if synthErr != nil {
					log.Printf("WARNING: chairman %s synthesis failed: %v", chairman, synthErr)
				} else {
					p, parseErr := parseCouncilJSON(synthResp)
					if parseErr != nil {
						log.Printf("WARNING: chairman %s returned unparseable JSON: %v", chairman, parseErr)
					} else {
						parsed = p
					}
				}
			}

			// Step 2: Try other successful providers as fallback chairman
			if parsed == nil {
				successfulNames := make(map[string]bool, len(successful))
				for _, s := range successful {
					successfulNames[s.Provider] = true
				}
				for _, m := range members {
					if m.Name() == chairman || !successfulNames[m.Name()] {
						continue
					}
					log.Printf("Trying fallback chairman: %s", m.Name())
					synthResp, synthErr := m.Complete(ctx, sysPrompt, synthesisPrompt)
					if synthErr != nil {
						log.Printf("WARNING: fallback chairman %s failed: %v", m.Name(), synthErr)
						continue
					}
					p, parseErr := parseCouncilJSON(synthResp)
					if parseErr != nil {
						log.Printf("WARNING: fallback chairman %s returned unparseable JSON: %v", m.Name(), parseErr)
						continue
					}
					parsed = p
					actualChairman = m.Name()
					log.Printf("Fallback chairman %s succeeded", m.Name())
					break
				}
			}

			// Step 3: Try parsing individual council responses directly
			if parsed == nil {
				for _, resp := range successful {
					p, parseErr := parseCouncilJSON(resp.Response)
					if parseErr == nil {
						parsed = p
						actualChairman = resp.Provider
						log.Printf("Using direct council response from %s (no synthesis)", resp.Provider)
						break
					}
				}
			}

			// Apply enrichment if any fallback succeeded
			if parsed != nil {
				chairman = actualChairman
				mergeAircraftEnrichments(&data, parsed.AircraftEnrichments)
				addVesselDeployments(&data, parsed.VesselDeployments)
				data.Summary = parsed.IntelligenceSummary
				log.Printf("AI enrichment applied: %d aircraft enriched, %d vessel deployments added",
					len(parsed.AircraftEnrichments), len(parsed.VesselDeployments))
			} else {
				log.Printf("WARNING: all enrichment attempts failed, continuing with unenriched data")
			}

			// Use the chairman's score from the scores map
			if entry, ok := scores[chairman]; ok {
				score = entry.AvgScore
			}
		}

		fmt.Printf("Enrichment complete. Chairman: %s, Score: %.2f\n", chairman, score)
	} else {
		fmt.Println("No AI providers configured, skipping enrichment.")
	}

	// ── Generate phase ─────────────────────────────────────────────────
	fmt.Println("Generating KML...")

	// Ensure output directory exists
	if err := os.MkdirAll("output", 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := kml.Generate("output/tracker.kml", &data, chairman, score); err != nil {
		return fmt.Errorf("generate KML: %w", err)
	}

	// Print summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Aircraft: %d tracked\n", len(data.Aircraft))
	fmt.Printf("Vessels:  %d tracked\n", len(data.Vessels))
	fmt.Printf("Events:   %d tracked\n", len(data.Events))
	fmt.Printf("News:     %d items\n", len(data.News))
	fmt.Println("KML written to output/tracker.kml")

	return nil
}

// buildDataSummary creates a concise text summary of collected data for AI analysis.
// Sending raw JSON of hundreds of aircraft would exceed token limits, so we summarize.
func buildDataSummary(data models.CollectedData) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "=== MILITARY TRACKING DATA (%s UTC) ===\n\n", data.Timestamp.Format("2006-01-02 15:04"))

	// Aircraft: show up to 50 most interesting (those with callsigns or type info)
	fmt.Fprintf(&b, "AIRCRAFT (%d total tracked):\n", len(data.Aircraft))
	shown := 0
	for _, a := range data.Aircraft {
		if shown >= 50 {
			fmt.Fprintf(&b, "... and %d more aircraft\n", len(data.Aircraft)-50)
			break
		}
		callsign := a.Callsign
		if callsign == "" {
			callsign = "UNKNOWN"
		}
		fmt.Fprintf(&b, "- %s | Hex:%s | Alt:%dft | Speed:%dkts | Lat:%.2f Lon:%.2f",
			callsign, a.Hex, int(a.Altitude), int(a.Speed), a.Lat, a.Lon)
		if a.Type != "" {
			fmt.Fprintf(&b, " | Type:%s", a.Type)
		}
		b.WriteString("\n")
		shown++
	}

	// Vessels
	if len(data.Vessels) > 0 {
		fmt.Fprintf(&b, "\nVESSELS (%d tracked):\n", len(data.Vessels))
		for _, v := range data.Vessels {
			fmt.Fprintf(&b, "- %s (MMSI:%s) | Lat:%.2f Lon:%.2f | Speed:%.1fkts | Type:%s\n",
				v.Name, v.MMSI, v.Lat, v.Lon, v.Speed, v.Type)
		}
	}

	// Events
	if len(data.Events) > 0 {
		fmt.Fprintf(&b, "\nEVENTS (%d):\n", len(data.Events))
		for _, e := range data.Events {
			fmt.Fprintf(&b, "- [%s] %s | %s | Lat:%.2f Lon:%.2f\n",
				e.Source, e.Title, e.Type, e.Lat, e.Lon)
		}
	}

	// News — split into fleet-tagged and general items
	if len(data.News) > 0 {
		var fleetItems, generalItems []models.NewsItem
		for _, n := range data.News {
			if n.Tag == "fleet" {
				fleetItems = append(fleetItems, n)
			} else {
				generalItems = append(generalItems, n)
			}
		}

		if len(fleetItems) > 0 {
			fmt.Fprintf(&b, "\nFLEET/NAVAL INTELLIGENCE (%d items):\n", len(fleetItems))
			limit := len(fleetItems)
			if limit > 10 {
				limit = 10
			}
			for _, n := range fleetItems[:limit] {
				desc := n.Description
				if len(desc) > 500 {
					desc = desc[:500] + "..."
				}
				fmt.Fprintf(&b, "- %s (Source: %s)\n  %s\n", n.Title, n.Source, desc)
			}
		}

		if len(generalItems) > 0 {
			fmt.Fprintf(&b, "\nGENERAL MILITARY NEWS (%d items):\n", len(generalItems))
			limit := len(generalItems)
			if limit > 15 {
				limit = 15
			}
			for _, n := range generalItems[:limit] {
				desc := n.Description
				if len(desc) > 150 {
					desc = desc[:150] + "..."
				}
				fmt.Fprintf(&b, "- %s (Source: %s) %s\n", n.Title, n.Source, desc)
			}
		}
	}

	return b.String()
}

// councilJSON represents the structured JSON response expected from the AI council.
type councilJSON struct {
	AircraftEnrichments []aircraftEnrichment `json:"aircraft_enrichments"`
	VesselDeployments   []vesselDeployment   `json:"vessel_deployments"`
	IntelligenceSummary string               `json:"intelligence_summary"`
}

// aircraftEnrichment is the AI's assessment of a specific tracked aircraft.
type aircraftEnrichment struct {
	Hex        string `json:"hex"`
	Branch     string `json:"branch"`
	Mission    string `json:"mission"`
	Assessment string `json:"assessment"`
}

// vesselDeployment represents a known vessel deployment from AI analysis.
type vesselDeployment struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Status  string  `json:"status"`
	Details string  `json:"details"`
}

// buildStructuredSystemPrompt returns the system prompt that instructs the AI to
// return a structured JSON response with enrichments and deployments.
func buildStructuredSystemPrompt() string {
	return `You are a military intelligence analyst. Analyze the tracking data and return ONLY a valid JSON object with exactly this structure (no markdown, no code fences, no extra text):

{
  "aircraft_enrichments": [
    {
      "hex": "HEX_CODE",
      "branch": "USAF|USN|USMC|USA|USCG",
      "mission": "short mission type like transport, ISR, patrol, tanker, training, etc.",
      "assessment": "1-2 sentence assessment of this aircraft's likely activity"
    }
  ],
  "vessel_deployments": [
    {
      "name": "USS Ship Name (HULL-N)",
      "type": "carrier_strike_group|amphibious_ready_group|destroyer|cruiser|submarine|other",
      "lat": 0.0,
      "lon": 0.0,
      "status": "deployed|in-port|transit",
      "details": "Brief description of current deployment area and mission"
    }
  ],
  "intelligence_summary": "2-4 paragraph overall intelligence assessment covering key military activities, unusual patterns, and strategic implications observed in the data."
}

Rules:
- aircraft_enrichments: Only include the most notable/interesting aircraft (up to 20). Match by hex code from the data.
- vessel_deployments: Extract ALL deployed US Navy vessel positions from the FLEET/NAVAL INTELLIGENCE section: carrier strike groups (CVN), amphibious ready groups (LHD/LHA), forward-deployed destroyers (DDG), and cruisers (CG). Use ship names, hull numbers, and geographic descriptions from the news. Convert to approximate lat/lon. Do NOT guess positions — only include vessels explicitly mentioned in the provided data with location information. Include up to 20 vessels.
- intelligence_summary: Provide a concise but thorough assessment of overall military posture and activity patterns.
- Return ONLY the JSON object. No other text before or after.`
}

// buildStructuredSynthesisPrompt constructs the chairman synthesis prompt that
// includes all council analyses and asks for a structured JSON output.
func buildStructuredSynthesisPrompt(responses []enrichment.CouncilResponse) string {
	var b bytes.Buffer

	b.WriteString("You are the chairman synthesizer. Below are independent analyses from multiple AI council members. ")
	b.WriteString("Synthesize them into a single, authoritative assessment. ")
	b.WriteString("Resolve any contradictions by favoring the most detailed and well-reasoned analysis.\n\n")

	for i, r := range responses {
		fmt.Fprintf(&b, "=== Analysis %d ===\n%s\n\n", i+1, r.Response)
	}

	b.WriteString("Combine the above analyses into your final synthesized response. ")
	b.WriteString("Return ONLY a valid JSON object following the exact schema from your instructions. No markdown, no code fences.")

	return b.String()
}

// parseCouncilJSON attempts to parse the AI response as structured JSON.
// It handles common issues like markdown code fences wrapping the JSON.
func parseCouncilJSON(response string) (*councilJSON, error) {
	// Try to extract JSON from the response (handle markdown code fences)
	cleaned := strings.TrimSpace(response)

	// Strip markdown code fences if present
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)
	}

	// Try to find a JSON object in the response
	startIdx := strings.Index(cleaned, "{")
	endIdx := strings.LastIndex(cleaned, "}")
	if startIdx >= 0 && endIdx > startIdx {
		cleaned = cleaned[startIdx : endIdx+1]
	}

	var result councilJSON
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("unmarshal council JSON: %w (first 200 chars: %.200s)", err, cleaned)
	}

	return &result, nil
}

// mergeAircraftEnrichments applies AI-generated enrichments to matching aircraft
// in the collected data (matched by hex code).
func mergeAircraftEnrichments(data *models.CollectedData, enrichments []aircraftEnrichment) {
	if len(enrichments) == 0 {
		return
	}

	// Build a lookup map for fast matching
	enrichMap := make(map[string]aircraftEnrichment, len(enrichments))
	for _, e := range enrichments {
		enrichMap[strings.ToUpper(e.Hex)] = e
	}

	for i := range data.Aircraft {
		if e, ok := enrichMap[strings.ToUpper(data.Aircraft[i].Hex)]; ok {
			// Only overwrite if the AI provided a non-empty value and the original is empty
			if data.Aircraft[i].Branch == "" && e.Branch != "" {
				data.Aircraft[i].Branch = e.Branch
			}
			if data.Aircraft[i].Mission == "" && e.Mission != "" {
				data.Aircraft[i].Mission = e.Mission
			}
		}
	}
}

// addVesselDeployments converts AI-identified vessel deployments into Vessel
// entries and appends them to the collected data.
func addVesselDeployments(data *models.CollectedData, deployments []vesselDeployment) {
	for _, d := range deployments {
		if d.Name == "" {
			continue
		}
		status := d.Status
		if d.Details != "" {
			if status != "" {
				status += " — " + d.Details
			} else {
				status = d.Details
			}
		}
		vesselType := d.Type
		if vesselType == "" {
			vesselType = "warship"
		}
		vessel := models.Vessel{
			Name:   d.Name,
			Type:   vesselType,
			Lat:    d.Lat,
			Lon:    d.Lon,
			Source: "ai_intel",
			Branch: "USN",
			Class:  status,
		}
		data.Vessels = append(data.Vessels, vessel)
	}
}

// saveJSON marshals v as indented JSON and writes it to the given path.
func saveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
