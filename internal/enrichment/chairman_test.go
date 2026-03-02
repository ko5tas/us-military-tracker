package enrichment

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSelectChairman(t *testing.T) {
	scores := ChairmanScores{
		"groq":    {AvgScore: 7.5, Runs: 3, Last5: []float64{7.0, 7.5, 8.0}},
		"mistral": {AvgScore: 8.2, Runs: 5, Last5: []float64{8.0, 8.5, 8.0, 8.2, 8.3}},
		"deepseek": {AvgScore: 6.0, Runs: 2, Last5: []float64{5.5, 6.5}},
	}

	got := SelectChairman(scores)
	if got != "mistral" {
		t.Errorf("SelectChairman: got %q, want %q", got, "mistral")
	}
}

func TestSelectChairmanEmpty(t *testing.T) {
	got := SelectChairman(ChairmanScores{})
	if got != "" {
		t.Errorf("SelectChairman(empty): got %q, want %q", got, "")
	}
}

func TestUpdateScore(t *testing.T) {
	scores := ChairmanScores{
		"groq": {AvgScore: 7.0, Runs: 2, Last5: []float64{6.0, 8.0}},
	}

	// Add a third score.
	UpdateScore(scores, "groq", 9.0)

	entry := scores["groq"]
	if entry.Runs != 3 {
		t.Errorf("Runs: got %d, want 3", entry.Runs)
	}
	if len(entry.Last5) != 3 {
		t.Errorf("Last5 length: got %d, want 3", len(entry.Last5))
	}

	// AvgScore should be average of Last5: (6+8+9)/3 = 7.666...
	wantAvg := (6.0 + 8.0 + 9.0) / 3.0
	if math.Abs(entry.AvgScore-wantAvg) > 0.001 {
		t.Errorf("AvgScore: got %f, want %f", entry.AvgScore, wantAvg)
	}

	// Fill up to 5 entries.
	UpdateScore(scores, "groq", 7.0)
	UpdateScore(scores, "groq", 8.0)
	entry = scores["groq"]
	if len(entry.Last5) != 5 {
		t.Errorf("Last5 length after 5 scores: got %d, want 5", len(entry.Last5))
	}
	if entry.Runs != 5 {
		t.Errorf("Runs after 5 scores: got %d, want 5", entry.Runs)
	}

	// Add a 6th score — oldest should be dropped, cap at 5.
	UpdateScore(scores, "groq", 10.0)
	entry = scores["groq"]
	if len(entry.Last5) != 5 {
		t.Errorf("Last5 length after 6 scores: got %d, want 5", len(entry.Last5))
	}
	if entry.Runs != 6 {
		t.Errorf("Runs after 6 scores: got %d, want 6", entry.Runs)
	}

	// Last5 should be [8, 9, 7, 8, 10] (dropped the original 6.0).
	wantLast5 := []float64{8.0, 9.0, 7.0, 8.0, 10.0}
	for i, v := range wantLast5 {
		if math.Abs(entry.Last5[i]-v) > 0.001 {
			t.Errorf("Last5[%d]: got %f, want %f", i, entry.Last5[i], v)
		}
	}

	// AvgScore should be average of new Last5.
	wantAvg = (8.0 + 9.0 + 7.0 + 8.0 + 10.0) / 5.0
	if math.Abs(entry.AvgScore-wantAvg) > 0.001 {
		t.Errorf("AvgScore after cap: got %f, want %f", entry.AvgScore, wantAvg)
	}

	// Update a new provider that doesn't exist yet.
	UpdateScore(scores, "newprovider", 5.0)
	entry = scores["newprovider"]
	if entry.Runs != 1 {
		t.Errorf("new provider Runs: got %d, want 1", entry.Runs)
	}
	if len(entry.Last5) != 1 {
		t.Errorf("new provider Last5 length: got %d, want 1", len(entry.Last5))
	}
	if entry.AvgScore != 5.0 {
		t.Errorf("new provider AvgScore: got %f, want 5.0", entry.AvgScore)
	}
}

func TestLoadSaveScores(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scores.json")

	// Loading a non-existent file should return an empty map, no error.
	scores, err := LoadScores(path)
	if err != nil {
		t.Fatalf("LoadScores non-existent: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("LoadScores non-existent: got %d entries, want 0", len(scores))
	}

	// Populate and save.
	scores["groq"] = ScoreEntry{AvgScore: 7.5, Runs: 3, Last5: []float64{7.0, 7.5, 8.0}}
	scores["mistral"] = ScoreEntry{AvgScore: 8.0, Runs: 2, Last5: []float64{7.5, 8.5}}

	if err := SaveScores(path, scores); err != nil {
		t.Fatalf("SaveScores: %v", err)
	}

	// Verify the file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SaveScores did not create the file")
	}

	// Load it back.
	loaded, err := LoadScores(path)
	if err != nil {
		t.Fatalf("LoadScores after save: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("loaded scores count: got %d, want 2", len(loaded))
	}

	groq := loaded["groq"]
	if groq.Runs != 3 {
		t.Errorf("loaded groq.Runs: got %d, want 3", groq.Runs)
	}
	if math.Abs(groq.AvgScore-7.5) > 0.001 {
		t.Errorf("loaded groq.AvgScore: got %f, want 7.5", groq.AvgScore)
	}
	if len(groq.Last5) != 3 {
		t.Errorf("loaded groq.Last5 length: got %d, want 3", len(groq.Last5))
	}

	mistral := loaded["mistral"]
	if mistral.Runs != 2 {
		t.Errorf("loaded mistral.Runs: got %d, want 2", mistral.Runs)
	}
	if math.Abs(mistral.AvgScore-8.0) > 0.001 {
		t.Errorf("loaded mistral.AvgScore: got %f, want 8.0", mistral.AvgScore)
	}
}

func TestBuildSynthesisPrompt(t *testing.T) {
	responses := []CouncilResponse{
		{
			Provider: "groq",
			Response: "The aircraft is a C-17 Globemaster III.",
			Err:      nil,
			Latency:  2 * time.Second,
		},
		{
			Provider: "mistral",
			Response: "This appears to be a military transport aircraft, likely a C-17.",
			Err:      nil,
			Latency:  3 * time.Second,
		},
	}

	prompt := BuildSynthesisPrompt(responses)

	if prompt == "" {
		t.Fatal("BuildSynthesisPrompt returned empty string")
	}

	// Should contain numbered analyses.
	if !strings.Contains(prompt, "Analysis 1") {
		t.Error("prompt should contain 'Analysis 1'")
	}
	if !strings.Contains(prompt, "Analysis 2") {
		t.Error("prompt should contain 'Analysis 2'")
	}

	// Should contain the response content.
	if !strings.Contains(prompt, "C-17 Globemaster III") {
		t.Error("prompt should contain first response content")
	}
	if !strings.Contains(prompt, "military transport aircraft") {
		t.Error("prompt should contain second response content")
	}

	// Should NOT contain provider names (anonymized).
	if strings.Contains(prompt, "groq") {
		t.Error("prompt should not contain provider name 'groq'")
	}
	if strings.Contains(prompt, "mistral") {
		t.Error("prompt should not contain provider name 'mistral'")
	}
}
