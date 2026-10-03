package gate

import (
	"fmt"
	"os"

	"example.com/lcov-change-coverage/internal/lcov"
)

// Ratio is an exact fraction covered/total, rendered as a JSON string;
// it is null when the denominator is zero.
type Ratio struct {
	Covered int
	Total   int
}

func (r Ratio) MarshalJSON() ([]byte, error) {
	if r.Total == 0 {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("%q", fmt.Sprintf("%d/%d", r.Covered, r.Total))), nil
}

// meets reports whether the ratio satisfies the threshold given in
// ten-thousandths (0..10000). A zero denominator always passes.
func (r Ratio) meets(threshold int) bool {
	if r.Total == 0 {
		return true
	}
	return r.Covered*10000 >= threshold*r.Total
}

// FileReport holds per-file coverage and change-coverage counts.
type FileReport struct {
	Path                 string `json:"path"`
	Covered              int    `json:"covered"`
	Total                int    `json:"total"`
	Ratio                Ratio  `json:"ratio"`
	ChangeCovered        int    `json:"change_covered"`
	ChangeTotal          int    `json:"change_total"`
	ChangeRatio          Ratio  `json:"change_ratio"`
	UncoveredChangeLines []int  `json:"uncovered_change_lines"`
	NonExecutableLines   []int  `json:"non_executable_lines"`
}

// Report is the JSON document written to the configured output file.
type Report struct {
	Passed       bool         `json:"passed"`
	Overall      FileReport   `json:"overall"`
	Files        []FileReport `json:"files"`
	MissingFiles []string     `json:"missing_files"`
}

// Run merges all LCOV inputs, evaluates the thresholds and returns the
// report. Missing coverage data for a changed file fails the gate.
func Run(cfg *Config, changes Changes) (*Report, error) {
	hits := map[string]map[int]int64{}
	for _, path := range cfg.LCOVFiles {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open lcov file: %w", err)
		}
		files, err := lcov.Parse(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, lf := range files {
			m := hits[lf.Path]
			if m == nil {
				m = map[int]int64{}
				hits[lf.Path] = m
			}
			for line, count := range lf.Hits {
				m[line] += count
			}
		}
	}
	return evaluate(cfg, changes, hits), nil
}
