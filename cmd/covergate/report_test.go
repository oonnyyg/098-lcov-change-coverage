package main

import "testing"

func TestThresholdPassExact(t *testing.T) {
	// 1/3 = 0.3333... must not pass a 3334 (33.34%) threshold,
	// but must pass 3333 exactly-comparable thresholds.
	if thresholdPass(1, 3, 3334) {
		t.Error("1/3 should not pass threshold 3334")
	}
	if !thresholdPass(1, 3, 3333) {
		t.Error("1/3 should pass threshold 3333")
	}
	if !thresholdPass(3, 4, 7500) {
		t.Error("3/4 should pass threshold 7500 (equal passes)")
	}
	if !thresholdPass(0, 0, 10000) {
		t.Error("zero denominator should always pass")
	}
}

func TestBuildReport(t *testing.T) {
	cov := Coverage{
		"src/a.go": {1: 2, 2: 0, 3: 1},
		"src/b.go": {10: 0},
	}
	changes := Changes{
		"src/a.go": {1: true, 2: true, 99: true},
		"src/b.go": {10: true},
		"src/c.go": {1: true},
	}
	cfg := &Config{OverallThreshold: 5000, ChangeThreshold: 5000}
	rep := buildReport(cov, changes, cfg)

	if rep.Overall.Covered != 2 || rep.Overall.Total != 4 {
		t.Errorf("overall = %d/%d, want 2/4", rep.Overall.Covered, rep.Overall.Total)
	}
	if rep.Overall.Ratio == nil || *rep.Overall.Ratio != 0.5 {
		t.Errorf("overall ratio = %v, want 0.5", rep.Overall.Ratio)
	}
	// Changed executable lines: a.go{1,2}, b.go{10} -> covered 1 of 3.
	if rep.Change.Covered != 1 || rep.Change.Total != 3 {
		t.Errorf("change = %d/%d, want 1/3", rep.Change.Covered, rep.Change.Total)
	}
	if len(rep.MissingFiles) != 1 || rep.MissingFiles[0] != "src/c.go" {
		t.Errorf("missing files = %v, want [src/c.go]", rep.MissingFiles)
	}
	if rep.Pass {
		t.Error("report should fail: missing file and change threshold not met")
	}

	var a *FileReport
	for i := range rep.Files {
		if rep.Files[i].Path == "src/a.go" {
			a = &rep.Files[i]
		}
	}
	if a == nil {
		t.Fatal("missing per-file report for src/a.go")
	}
	if len(a.UncoveredChangedLines) != 1 || a.UncoveredChangedLines[0] != 2 {
		t.Errorf("uncovered changed lines = %v, want [2]", a.UncoveredChangedLines)
	}
	if len(a.NonExecutableChangeLines) != 1 || a.NonExecutableChangeLines[0] != 99 {
		t.Errorf("non-executable changed lines = %v, want [99]", a.NonExecutableChangeLines)
	}
}

func TestBuildReportZeroDenominators(t *testing.T) {
	rep := buildReport(Coverage{}, Changes{}, &Config{OverallThreshold: 10000, ChangeThreshold: 10000})
	if !rep.Pass {
		t.Error("empty coverage and changes should pass (null ratios)")
	}
	if rep.Overall.Ratio != nil || rep.Change.Ratio != nil {
		t.Error("ratios should be null with zero denominators")
	}
}

func TestMissingFileFailsEvenWithZeroDenominator(t *testing.T) {
	rep := buildReport(Coverage{}, Changes{"x.go": {1: true}}, &Config{})
	if rep.Pass {
		t.Error("missing coverage data must fail the gate")
	}
	if rep.Change.Total != 0 || rep.Change.Ratio != nil {
		t.Error("missing file contributes no denominator")
	}
}
