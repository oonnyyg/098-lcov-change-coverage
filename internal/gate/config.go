// Package gate merges LCOV coverage with a changed-lines manifest and
// evaluates overall and per-change coverage thresholds.
package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"example.com/lcov-change-coverage/internal/lcov"
)

// Config is the JSON task file consumed by covergate.
type Config struct {
	LCOVFiles        []string `json:"lcov_files"`
	ChangesFile      string   `json:"changes_file"`
	OverallThreshold int      `json:"overall_threshold"`
	ChangeThreshold  int      `json:"change_threshold"`
	Output           string   `json:"output"`
}

// LoadConfig reads and validates the task configuration. Relative paths
// are resolved against the directory containing the config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.ChangesFile == "" {
		return nil, fmt.Errorf("config: changes_file is required")
	}
	if cfg.Output == "" {
		return nil, fmt.Errorf("config: output is required")
	}
	for _, name := range []string{"overall_threshold", "change_threshold"} {
		v := cfg.OverallThreshold
		if name == "change_threshold" {
			v = cfg.ChangeThreshold
		}
		if v < 0 || v > 10000 {
			return nil, fmt.Errorf("config: %s must be between 0 and 10000, got %d", name, v)
		}
	}
	base := filepath.Dir(path)
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(base, p)
	}
	cfg.ChangesFile = resolve(cfg.ChangesFile)
	cfg.Output = resolve(cfg.Output)
	for i, f := range cfg.LCOVFiles {
		cfg.LCOVFiles[i] = resolve(f)
	}
	return &cfg, nil
}

// Changes maps source paths to the set of changed line numbers in the
// new revision of each file.
type Changes map[string]map[int]bool

// LoadChanges reads the JSON changed-lines manifest. Duplicate lines
// are deduplicated.
func LoadChanges(path string) (Changes, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read changes: %w", err)
	}
	var raw map[string][]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse changes: %w", err)
	}
	changes := Changes{}
	for p, lines := range raw {
		if err := lcov.ValidatePath(p); err != nil {
			return nil, fmt.Errorf("changes: %w", err)
		}
		set := map[int]bool{}
		for _, n := range lines {
			if n <= 0 {
				return nil, fmt.Errorf("changes: file %q has invalid line number %d", p, n)
			}
			set[n] = true
		}
		changes[p] = set
	}
	return changes, nil
}
