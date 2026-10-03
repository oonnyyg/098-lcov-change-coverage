package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunMergesShardsAndEvaluates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.info", "SF:src/a.go\nDA:1,1\nDA:2,0\nend_of_record\n")
	writeFile(t, dir, "b.info", "SF:src/a.go\nDA:2,3\nDA:3,0\nend_of_record\nSF:src/b.go\nDA:1,0\nend_of_record\n")
	writeFile(t, dir, "changes.json", `{"src/a.go":[1,2,2,9],"src/b.go":[1],"src/missing.go":[5]}`)
	cfgPath := writeFile(t, dir, "task.json", `{
  "lcov_files": ["a.info", "b.info"],
  "changes_file": "changes.json",
  "overall_threshold": 5000,
  "change_threshold": 5000,
  "output": "report.json"
}`)
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := LoadChanges(cfg.ChangesFile)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Run(cfg, changes)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Error("expected gate to fail due to missing file")
	}
	if len(rep.MissingFiles) != 1 || rep.MissingFiles[0] != "src/missing.go" {
		t.Errorf("missing files: %v", rep.MissingFiles)
	}
	// overall: 4 executable lines (a:1,2,3 b:1), covered a:1,a:2 => 2/4
	if rep.Overall.Total != 4 || rep.Overall.Covered != 2 {
		t.Errorf("overall: %+v", rep.Overall)
	}
	// change coverage: src/a.go lines 1(cov),2(cov),9(non-exec); src/b.go line 1 uncovered
	if rep.Overall.ChangeTotal != 3 || rep.Overall.ChangeCovered != 2 {
		t.Errorf("change totals: %+v", rep.Overall)
	}
	var a *FileReport
	for i := range rep.Files {
		if rep.Files[i].Path == "src/a.go" {
			a = &rep.Files[i]
		}
	}
	if a == nil {
		t.Fatal("no report for src/a.go")
	}
	if len(a.NonExecutableLines) != 1 || a.NonExecutableLines[0] != 9 {
		t.Errorf("non-executable lines: %v", a.NonExecutableLines)
	}
	if len(a.UncoveredChangeLines) != 0 {
		t.Errorf("uncovered change lines: %v", a.UncoveredChangeLines)
	}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["overall"].(map[string]any)["ratio"] != "2/4" {
		t.Errorf("overall ratio encoding: %v", decoded["overall"])
	}
}

func TestThresholdsExactAndNullRatio(t *testing.T) {
	cfg := &Config{OverallThreshold: 3333, ChangeThreshold: 0}
	changes := Changes{}
	hits := map[string]map[int]int64{"a.go": {1: 1, 2: 0, 3: 0}}
	rep := evaluate(cfg, changes, hits)
	// 1/3 = 3333.33... >= 3333/10000 passes exactly.
	if !rep.Passed {
		t.Error("1/3 should meet 3333/10000")
	}
	cfg.OverallThreshold = 3334
	rep = evaluate(cfg, changes, hits)
	if rep.Passed {
		t.Error("1/3 should not meet 3334/10000")
	}
	// Zero denominator: ratio null, threshold passes.
	rep = evaluate(&Config{OverallThreshold: 0, ChangeThreshold: 10000}, changes,
		map[string]map[int]int64{"a.go": {1: 0}})
	if !rep.Passed {
		t.Error("zero change denominator should pass")
	}
	if rep.Overall.ChangeRatio.Total != 0 {
		t.Errorf("change ratio: %+v", rep.Overall.ChangeRatio)
	}
}

func TestLoadConfigValidation(t *testing.T) {
	dir := t.TempDir()
	bad := writeFile(t, dir, "bad.json", `{"changes_file":"c.json","output":"o.json","overall_threshold":10001}`)
	if _, err := LoadConfig(bad); err == nil {
		t.Error("expected threshold validation error")
	}
	missing := writeFile(t, dir, "missing.json", `{"output":"o.json"}`)
	if _, err := LoadConfig(missing); err == nil {
		t.Error("expected changes_file required error")
	}
}

func TestLoadChangesValidation(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "c.json", `{"a.go":[0]}`)
	if _, err := LoadChanges(p); err == nil {
		t.Error("expected invalid line number error")
	}
	p = writeFile(t, dir, "c2.json", `{"/abs.go":[1]}`)
	if _, err := LoadChanges(p); err == nil {
		t.Error("expected invalid path error")
	}
}
