package enrichment

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ScoreEntry tracks performance metrics for a single AI provider acting as chairman.
type ScoreEntry struct {
	AvgScore float64   `json:"avg_score"`
	Runs     int       `json:"runs"`
	Last5    []float64 `json:"last5"`
}

// ChairmanScores maps provider names to their scoring history.
type ChairmanScores map[string]ScoreEntry

// SelectChairman returns the provider name with the highest AvgScore.
// Returns "" for an empty map.
func SelectChairman(scores ChairmanScores) string {
	if len(scores) == 0 {
		return ""
	}

	var best string
	var bestScore float64
	first := true

	for name, entry := range scores {
		if first || entry.AvgScore > bestScore {
			best = name
			bestScore = entry.AvgScore
			first = false
		}
	}

	return best
}

// UpdateScore adds a new score for the named provider, recalculates the rolling
// average over the last 5 scores, and increments the run counter. If the
// provider does not exist in the map it is created.
func UpdateScore(scores ChairmanScores, name string, score float64) {
	entry := scores[name]

	entry.Last5 = append(entry.Last5, score)
	if len(entry.Last5) > 5 {
		entry.Last5 = entry.Last5[len(entry.Last5)-5:]
	}

	entry.Runs++

	var sum float64
	for _, v := range entry.Last5 {
		sum += v
	}
	entry.AvgScore = sum / float64(len(entry.Last5))

	scores[name] = entry
}

// LoadScores reads chairman scores from a JSON file. Returns an empty map
// (not an error) when the file does not exist.
func LoadScores(path string) (ChairmanScores, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(ChairmanScores), nil
		}
		return nil, fmt.Errorf("read scores file: %w", err)
	}

	var scores ChairmanScores
	if err := json.Unmarshal(data, &scores); err != nil {
		return nil, fmt.Errorf("unmarshal scores: %w", err)
	}

	return scores, nil
}

// SaveScores writes chairman scores to a JSON file with indentation for
// human readability.
func SaveScores(path string, scores ChairmanScores) error {
	data, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scores: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write scores file: %w", err)
	}

	return nil
}

// BuildSynthesisPrompt constructs a chairman prompt that includes all council
// analyses in anonymized, numbered form. Provider names are stripped so the
// chairman evaluates content without identity bias.
func BuildSynthesisPrompt(responses []CouncilResponse) string {
	var b strings.Builder

	b.WriteString("You are the chairman synthesizer. Below are independent analyses from multiple AI council members. ")
	b.WriteString("Synthesize them into a single, authoritative assessment. ")
	b.WriteString("Resolve any contradictions by favoring the most detailed and well-reasoned analysis.\n\n")

	for i, r := range responses {
		fmt.Fprintf(&b, "=== Analysis %d ===\n%s\n\n", i+1, r.Response)
	}

	b.WriteString("Provide your synthesized assessment:")

	return b.String()
}
