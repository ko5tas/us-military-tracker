package platform

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RunnerProfile captures the hardware profile of the GitHub Actions runner
// (or local machine) at a point in time.
type RunnerProfile struct {
	CPUs      int       `json:"cpus"`
	MemoryMB  int       `json:"memory_mb"`
	DiskGB    int       `json:"disk_gb"`
	Timestamp time.Time `json:"timestamp"`
}

// DetectRunnerProfile reads runtime.NumCPU() and parses /proc/meminfo for
// MemTotal. On non-Linux systems, MemoryMB will be 0.
func DetectRunnerProfile() RunnerProfile {
	profile := RunnerProfile{
		CPUs:      runtime.NumCPU(),
		Timestamp: time.Now().UTC(),
	}

	profile.MemoryMB = readMemoryMB()

	return profile
}

// readMemoryMB attempts to read MemTotal from /proc/meminfo.
// Returns 0 on non-Linux systems or on any parse error.
func readMemoryMB() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.Atoi(fields[1])
				if err != nil {
					return 0
				}
				return kb / 1024
			}
		}
	}

	return 0
}

// CompareProfiles returns a list of change descriptions between a stored and
// current runner profile. CPU must differ. Memory must differ by >512MB.
// Disk must differ.
func CompareProfiles(stored, current RunnerProfile) []string {
	var changes []string

	if stored.CPUs != current.CPUs {
		changes = append(changes, fmt.Sprintf("CPUs changed: %d -> %d", stored.CPUs, current.CPUs))
	}

	memDiff := current.MemoryMB - stored.MemoryMB
	if memDiff < 0 {
		memDiff = -memDiff
	}
	if memDiff > 512 {
		changes = append(changes, fmt.Sprintf("Memory changed: %dMB -> %dMB", stored.MemoryMB, current.MemoryMB))
	}

	if stored.DiskGB != current.DiskGB {
		changes = append(changes, fmt.Sprintf("Disk changed: %dGB -> %dGB", stored.DiskGB, current.DiskGB))
	}

	return changes
}

// LoadProfile reads a RunnerProfile from a JSON file.
func LoadProfile(path string) (RunnerProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunnerProfile{}, fmt.Errorf("read profile file: %w", err)
	}

	var profile RunnerProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return RunnerProfile{}, fmt.Errorf("unmarshal profile: %w", err)
	}

	return profile, nil
}

// SaveProfile writes a RunnerProfile to a JSON file with indentation.
func SaveProfile(path string, profile RunnerProfile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write profile file: %w", err)
	}

	return nil
}
