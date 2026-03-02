package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// ProviderConfig holds the full provider configuration including council
// members and candidate models under evaluation.
type ProviderConfig struct {
	SchemaVersion int          `json:"schema_version"`
	LastEvolved   time.Time    `json:"last_evolved"`
	Council       CouncilConfig `json:"council"`
}

// CouncilConfig holds the active members and shadow candidates.
type CouncilConfig struct {
	Members    []MemberConfig `json:"members"`
	Candidates []MemberConfig `json:"candidates"`
}

// MemberConfig describes a single AI provider member or candidate.
type MemberConfig struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Endpoint     string  `json:"endpoint"`
	Model        string  `json:"model"`
	DailyLimit   int     `json:"daily_limit"`
	QualityScore float64 `json:"quality_score"`
	Status       string  `json:"status"`
	ShadowWeeks  int     `json:"shadow_weeks"`
}

// LoadProviderConfig reads a ProviderConfig from a JSON file. Returns a default
// config with SchemaVersion 1 if the file does not exist.
func LoadProviderConfig(path string) (ProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProviderConfig{
				SchemaVersion: 1,
				LastEvolved:   time.Now().UTC(),
				Council: CouncilConfig{
					Members:    []MemberConfig{},
					Candidates: []MemberConfig{},
				},
			}, nil
		}
		return ProviderConfig{}, fmt.Errorf("read provider config: %w", err)
	}

	var cfg ProviderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ProviderConfig{}, fmt.Errorf("unmarshal provider config: %w", err)
	}

	return cfg, nil
}

// SaveProviderConfig writes a ProviderConfig to a JSON file with indentation.
func SaveProviderConfig(path string, cfg ProviderConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provider config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write provider config: %w", err)
	}

	return nil
}

// TryPromoteCandidates checks candidates with ShadowWeeks >= 3 that outperform
// the weakest active member. If found, the candidate replaces the weakest
// member. Returns true if any promotion occurred.
func TryPromoteCandidates(cfg *ProviderConfig) bool {
	if len(cfg.Council.Candidates) == 0 || len(cfg.Council.Members) == 0 {
		return false
	}

	// Find the weakest active member.
	weakestIdx := 0
	for i, m := range cfg.Council.Members {
		if m.QualityScore < cfg.Council.Members[weakestIdx].QualityScore {
			weakestIdx = i
		}
	}
	weakestScore := cfg.Council.Members[weakestIdx].QualityScore

	promoted := false
	var remaining []MemberConfig

	for _, candidate := range cfg.Council.Candidates {
		if candidate.ShadowWeeks >= 3 && candidate.QualityScore > weakestScore {
			// Promote: replace the weakest member with this candidate.
			candidate.Status = "active"
			cfg.Council.Members[weakestIdx] = candidate
			promoted = true

			// Recalculate weakest after replacement.
			weakestIdx = 0
			for i, m := range cfg.Council.Members {
				if m.QualityScore < cfg.Council.Members[weakestIdx].QualityScore {
					weakestIdx = i
				}
			}
			weakestScore = cfg.Council.Members[weakestIdx].QualityScore
		} else {
			remaining = append(remaining, candidate)
		}
	}

	if remaining == nil {
		remaining = []MemberConfig{}
	}
	cfg.Council.Candidates = remaining

	return promoted
}
