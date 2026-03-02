package enrichment

import (
	"math"
	"testing"

	"github.com/ko5tas/us-military-tracker/internal/models"
)

func TestEvalDataFidelity(t *testing.T) {
	t.Run("full match", func(t *testing.T) {
		raw := []models.Aircraft{
			{Hex: "AE1234"},
			{Hex: "AE5678"},
		}
		output := `{"assets":[{"id":"AE1234"},{"id":"AE5678"}]}`
		got := EvalDataFidelity(raw, output)
		if got != 1.0 {
			t.Errorf("EvalDataFidelity full match: got %f, want 1.0", got)
		}
	})

	t.Run("partial match", func(t *testing.T) {
		raw := []models.Aircraft{
			{Hex: "AE1234"},
			{Hex: "AE5678"},
			{Hex: "AE9999"},
			{Hex: "AE0000"},
		}
		output := `{"assets":[{"id":"AE1234"},{"id":"AE5678"}]}`
		got := EvalDataFidelity(raw, output)
		want := 0.5
		if math.Abs(got-want) > 0.001 {
			t.Errorf("EvalDataFidelity partial match: got %f, want %f", got, want)
		}
	})

	t.Run("empty raw", func(t *testing.T) {
		got := EvalDataFidelity(nil, `{"some":"output"}`)
		if got != 1.0 {
			t.Errorf("EvalDataFidelity empty raw: got %f, want 1.0", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		raw := []models.Aircraft{
			{Hex: "AE1234"},
			{Hex: "AE5678"},
		}
		output := `{"assets":[]}`
		got := EvalDataFidelity(raw, output)
		if got != 0.0 {
			t.Errorf("EvalDataFidelity no match: got %f, want 0.0", got)
		}
	})
}

func TestEvalHallucination(t *testing.T) {
	raw := []models.Aircraft{{Hex: "AE1234"}}
	got := EvalHallucination(raw, "some output")
	if got != 1.0 {
		t.Errorf("EvalHallucination baseline: got %f, want 1.0", got)
	}
}

func TestEvalFormatCorrectness(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		got := EvalFormatCorrectness(`{"assets":[{"id":"AE1234"}],"summary":"test"}`)
		if got != 1.0 {
			t.Errorf("EvalFormatCorrectness valid: got %f, want 1.0", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		got := EvalFormatCorrectness(`{not valid json`)
		if got != 0.0 {
			t.Errorf("EvalFormatCorrectness invalid: got %f, want 0.0", got)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		got := EvalFormatCorrectness("")
		if got != 0.0 {
			t.Errorf("EvalFormatCorrectness empty: got %f, want 0.0", got)
		}
	})

	t.Run("valid JSON array", func(t *testing.T) {
		got := EvalFormatCorrectness(`[1, 2, 3]`)
		if got != 1.0 {
			t.Errorf("EvalFormatCorrectness array: got %f, want 1.0", got)
		}
	})
}

func TestCompositeScore(t *testing.T) {
	// Perfect scores: 1.0*0.5 + 1.0*0.3 + 1.0*0.2 = 1.0
	got := CompositeScore(1.0, 1.0, 1.0)
	if math.Abs(got-1.0) > 0.001 {
		t.Errorf("CompositeScore perfect: got %f, want 1.0", got)
	}

	// Mixed scores: 0.8*0.5 + 1.0*0.3 + 0.5*0.2 = 0.4 + 0.3 + 0.1 = 0.8
	got = CompositeScore(0.8, 1.0, 0.5)
	want := 0.8
	if math.Abs(got-want) > 0.001 {
		t.Errorf("CompositeScore mixed: got %f, want %f", got, want)
	}

	// Zero scores: 0.0*0.5 + 0.0*0.3 + 0.0*0.2 = 0.0
	got = CompositeScore(0.0, 0.0, 0.0)
	if math.Abs(got-0.0) > 0.001 {
		t.Errorf("CompositeScore zero: got %f, want 0.0", got)
	}
}
