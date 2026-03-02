package enrichment

import (
	"encoding/json"
	"strings"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

// EvalDataFidelity checks what fraction of raw aircraft hex codes appear in the
// output string. Returns found/total. Returns 1.0 if raw is empty.
func EvalDataFidelity(raw []models.Aircraft, output string) float64 {
	if len(raw) == 0 {
		return 1.0
	}

	found := 0
	for _, a := range raw {
		if strings.Contains(output, a.Hex) {
			found++
		}
	}

	return float64(found) / float64(len(raw))
}

// EvalHallucination returns 1.0 as a baseline score. Hallucination detection is
// complex and will be enhanced later via AI evaluation.
func EvalHallucination(_ []models.Aircraft, _ string) float64 {
	return 1.0
}

// EvalFormatCorrectness returns 1.0 if output is valid JSON, 0.0 otherwise.
func EvalFormatCorrectness(output string) float64 {
	if json.Valid([]byte(output)) {
		return 1.0
	}
	return 0.0
}

// CompositeScore computes a weighted average of the three evaluation dimensions:
// fidelity*0.5 + hallucination*0.3 + format*0.2.
func CompositeScore(fidelity, hallucination, format float64) float64 {
	return fidelity*0.5 + hallucination*0.3 + format*0.2
}
