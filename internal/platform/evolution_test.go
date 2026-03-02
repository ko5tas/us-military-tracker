package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSaveProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")

	// Loading a non-existent file should return a default config, no error.
	cfg, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatalf("LoadProviderConfig non-existent: %v", err)
	}
	if cfg.SchemaVersion != 1 {
		t.Errorf("default SchemaVersion: got %d, want 1", cfg.SchemaVersion)
	}

	// Populate and save.
	cfg.Council.Members = []MemberConfig{
		{
			ID:           "gemini-1",
			Type:         "gemini",
			Endpoint:     "https://api.google.com",
			Model:        "gemini-2.0-flash",
			DailyLimit:   1500,
			QualityScore: 8.5,
			Status:       "active",
		},
	}
	cfg.Council.Candidates = []MemberConfig{
		{
			ID:           "groq-1",
			Type:         "openai-compatible",
			Endpoint:     "https://api.groq.com/openai/v1",
			Model:        "llama-3.3-70b",
			DailyLimit:   14400,
			QualityScore: 7.0,
			Status:       "shadow",
			ShadowWeeks:  2,
		},
	}
	cfg.LastEvolved = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	if err := SaveProviderConfig(path, cfg); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}

	// Verify the file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SaveProviderConfig did not create the file")
	}

	// Load it back.
	loaded, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatalf("LoadProviderConfig after save: %v", err)
	}

	if loaded.SchemaVersion != 1 {
		t.Errorf("loaded SchemaVersion: got %d, want 1", loaded.SchemaVersion)
	}
	if len(loaded.Council.Members) != 1 {
		t.Fatalf("loaded Members count: got %d, want 1", len(loaded.Council.Members))
	}
	if loaded.Council.Members[0].ID != "gemini-1" {
		t.Errorf("loaded Member ID: got %q, want %q", loaded.Council.Members[0].ID, "gemini-1")
	}
	if len(loaded.Council.Candidates) != 1 {
		t.Fatalf("loaded Candidates count: got %d, want 1", len(loaded.Council.Candidates))
	}
	if loaded.Council.Candidates[0].ShadowWeeks != 2 {
		t.Errorf("loaded Candidate ShadowWeeks: got %d, want 2", loaded.Council.Candidates[0].ShadowWeeks)
	}
}

func TestPromoteCandidate(t *testing.T) {
	t.Run("candidate beats weakest and gets promoted", func(t *testing.T) {
		cfg := ProviderConfig{
			SchemaVersion: 1,
			Council: CouncilConfig{
				Members: []MemberConfig{
					{ID: "strong", QualityScore: 9.0, Status: "active"},
					{ID: "weak", QualityScore: 5.0, Status: "active"},
				},
				Candidates: []MemberConfig{
					{ID: "challenger", QualityScore: 7.0, Status: "shadow", ShadowWeeks: 4},
				},
			},
		}

		promoted := TryPromoteCandidates(&cfg)
		if !promoted {
			t.Fatal("TryPromoteCandidates: expected promotion")
		}

		// The challenger should now be in Members.
		found := false
		for _, m := range cfg.Council.Members {
			if m.ID == "challenger" {
				found = true
				if m.Status != "active" {
					t.Errorf("promoted member Status: got %q, want %q", m.Status, "active")
				}
			}
		}
		if !found {
			t.Error("challenger was not found in Members after promotion")
		}

		// The weak member should be gone from Members.
		for _, m := range cfg.Council.Members {
			if m.ID == "weak" {
				t.Error("weak member should have been replaced")
			}
		}

		// Candidates list should be empty.
		if len(cfg.Council.Candidates) != 0 {
			t.Errorf("Candidates count after promotion: got %d, want 0", len(cfg.Council.Candidates))
		}
	})

	t.Run("candidate not ready yet", func(t *testing.T) {
		cfg := ProviderConfig{
			SchemaVersion: 1,
			Council: CouncilConfig{
				Members: []MemberConfig{
					{ID: "member1", QualityScore: 5.0, Status: "active"},
				},
				Candidates: []MemberConfig{
					{ID: "challenger", QualityScore: 7.0, Status: "shadow", ShadowWeeks: 2},
				},
			},
		}

		promoted := TryPromoteCandidates(&cfg)
		if promoted {
			t.Error("TryPromoteCandidates: expected no promotion (shadow weeks < 3)")
		}
	})

	t.Run("candidate does not outperform weakest", func(t *testing.T) {
		cfg := ProviderConfig{
			SchemaVersion: 1,
			Council: CouncilConfig{
				Members: []MemberConfig{
					{ID: "member1", QualityScore: 8.0, Status: "active"},
				},
				Candidates: []MemberConfig{
					{ID: "challenger", QualityScore: 6.0, Status: "shadow", ShadowWeeks: 4},
				},
			},
		}

		promoted := TryPromoteCandidates(&cfg)
		if promoted {
			t.Error("TryPromoteCandidates: expected no promotion (candidate worse than weakest)")
		}
	})

	t.Run("no candidates", func(t *testing.T) {
		cfg := ProviderConfig{
			SchemaVersion: 1,
			Council: CouncilConfig{
				Members: []MemberConfig{
					{ID: "member1", QualityScore: 8.0, Status: "active"},
				},
				Candidates: nil,
			},
		}

		promoted := TryPromoteCandidates(&cfg)
		if promoted {
			t.Error("TryPromoteCandidates: expected no promotion (no candidates)")
		}
	})
}
