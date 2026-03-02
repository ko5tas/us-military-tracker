package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectRunnerProfile(t *testing.T) {
	profile := DetectRunnerProfile()

	if profile.CPUs <= 0 {
		t.Errorf("DetectRunnerProfile CPUs: got %d, want > 0", profile.CPUs)
	}

	// MemoryMB can be 0 on non-Linux, so we only check it's non-negative.
	if profile.MemoryMB < 0 {
		t.Errorf("DetectRunnerProfile MemoryMB: got %d, want >= 0", profile.MemoryMB)
	}

	if profile.Timestamp.IsZero() {
		t.Error("DetectRunnerProfile Timestamp should not be zero")
	}
}

func TestCompareProfiles(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		stored := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}
		current := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}

		changes := CompareProfiles(stored, current)
		if len(changes) != 0 {
			t.Errorf("CompareProfiles no changes: got %d changes, want 0: %v", len(changes), changes)
		}
	})

	t.Run("CPU change", func(t *testing.T) {
		stored := RunnerProfile{CPUs: 2, MemoryMB: 8192, DiskGB: 100}
		current := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}

		changes := CompareProfiles(stored, current)
		if len(changes) == 0 {
			t.Error("CompareProfiles CPU change: expected at least one change")
		}
	})

	t.Run("memory change above threshold", func(t *testing.T) {
		stored := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}
		current := RunnerProfile{CPUs: 4, MemoryMB: 9000, DiskGB: 100}

		changes := CompareProfiles(stored, current)
		if len(changes) == 0 {
			t.Error("CompareProfiles memory change: expected at least one change")
		}
	})

	t.Run("memory change below threshold", func(t *testing.T) {
		stored := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}
		current := RunnerProfile{CPUs: 4, MemoryMB: 8400, DiskGB: 100}

		changes := CompareProfiles(stored, current)
		if len(changes) != 0 {
			t.Errorf("CompareProfiles small memory change: got %d changes, want 0: %v", len(changes), changes)
		}
	})

	t.Run("disk change", func(t *testing.T) {
		stored := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 100}
		current := RunnerProfile{CPUs: 4, MemoryMB: 8192, DiskGB: 200}

		changes := CompareProfiles(stored, current)
		if len(changes) == 0 {
			t.Error("CompareProfiles disk change: expected at least one change")
		}
	})
}

func TestLoadSaveProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.json")

	// Loading a non-existent file should return error.
	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("LoadProfile non-existent: expected error")
	}

	// Save a profile.
	profile := RunnerProfile{
		CPUs:      4,
		MemoryMB:  8192,
		DiskGB:    100,
		Timestamp: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}
	if err := SaveProfile(path, profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	// Verify the file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SaveProfile did not create the file")
	}

	// Load it back.
	loaded, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile after save: %v", err)
	}

	if loaded.CPUs != profile.CPUs {
		t.Errorf("loaded CPUs: got %d, want %d", loaded.CPUs, profile.CPUs)
	}
	if loaded.MemoryMB != profile.MemoryMB {
		t.Errorf("loaded MemoryMB: got %d, want %d", loaded.MemoryMB, profile.MemoryMB)
	}
	if loaded.DiskGB != profile.DiskGB {
		t.Errorf("loaded DiskGB: got %d, want %d", loaded.DiskGB, profile.DiskGB)
	}
	if !loaded.Timestamp.Equal(profile.Timestamp) {
		t.Errorf("loaded Timestamp: got %v, want %v", loaded.Timestamp, profile.Timestamp)
	}
}
