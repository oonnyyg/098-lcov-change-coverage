package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Config is the covergate task configuration (see FORMAT.md).
type Config struct {
	LCOVFiles        []string `json:"lcov_files"`
	ChangesFile      string   `json:"changes_file"`
	OutputFile       string   `json:"output_file"`
	OverallThreshold int      `json:"overall_threshold"`
	ChangeThreshold  int      `json:"change_threshold"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %v", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %v", path, err)
	}
	if len(cfg.LCOVFiles) == 0 {
		return nil, fmt.Errorf("config: lcov_files must be a non-empty list")
	}
	if cfg.ChangesFile == "" {
		return nil, fmt.Errorf("config: changes_file is required")
	}
	if cfg.OutputFile == "" {
		return nil, fmt.Errorf("config: output_file is required")
	}
	for _, name := range []struct {
		field string
		v     int
	}{{"overall_threshold", cfg.OverallThreshold}, {"change_threshold", cfg.ChangeThreshold}} {
		if name.v < 0 || name.v > 10000 {
			return nil, fmt.Errorf("config: %s must be an integer between 0 and 10000, got %d", name.field, name.v)
		}
	}
	// Resolve relative paths against the config file's directory.
	dir := filepath.Dir(path)
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(dir, p)
	}
	for i, f := range cfg.LCOVFiles {
		cfg.LCOVFiles[i] = resolve(f)
	}
	cfg.ChangesFile = resolve(cfg.ChangesFile)
	cfg.OutputFile = resolve(cfg.OutputFile)
	return &cfg, nil
}

// Changes maps file path -> set of changed line numbers (deduplicated).
type Changes map[string]map[int]bool

func loadChanges(path string) (Changes, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read changes file: %v", err)
	}
	var raw map[string][]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse changes file %s: %v", path, err)
	}
	changes := Changes{}
	for p, lines := range raw {
		if err := validatePath(p); err != nil {
			return nil, fmt.Errorf("changes file %s: invalid path: %v", path, err)
		}
		set := map[int]bool{}
		for _, ln := range lines {
			if ln <= 0 {
				return nil, fmt.Errorf("changes file %s: file %q has invalid line number %d (must be a positive integer)", path, p, ln)
			}
			set[ln] = true
		}
		changes[p] = set
	}
	return changes, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedLines(set map[int]bool) []int {
	lines := make([]int, 0, len(set))
	for ln := range set {
		lines = append(lines, ln)
	}
	sort.Ints(lines)
	return lines
}
