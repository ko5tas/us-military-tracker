package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
		g, err := providers.NewGemini(ctx, geminiKey, "gemini", "gemini-2.0-flash")
		if err != nil {
			log.Printf("WARNING: failed to init Gemini provider: %v", err)
		} else {
			members = append(members, g)
		}
	}
	if groqKey != "" {
		members = append(members, providers.NewGroq(groqKey))
	}
	if mistralKey != "" {
		members = append(members, providers.NewMistral(mistralKey))
	}
	if deepseekKey != "" {
		members = append(members, providers.NewDeepSeek(deepseekKey))
	}
	if openrouterKey != "" {
		members = append(members, providers.NewOpenRouter(openrouterKey))
	}

	chairman := "none"
	score := 0.0

	if len(members) > 0 {
		// Build prompt from collected data
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal collected data for enrichment: %w", err)
		}

		systemPrompt := "You are a military intelligence analyst. Analyze the following tracking data and provide insights about military activities, deployments, and potential missions."
		userPrompt := string(dataJSON)

		// Run council
		responses := enrichment.RunCouncil(ctx, members, systemPrompt, userPrompt)
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

			if chairmanProvider != nil {
				// Run chairman synthesis
				synthesisPrompt := enrichment.BuildSynthesisPrompt(successful)
				_, synthErr := chairmanProvider.Complete(ctx, "You are the chairman synthesizer.", synthesisPrompt)
				if synthErr != nil {
					log.Printf("WARNING: chairman synthesis failed: %v", synthErr)
				}
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
