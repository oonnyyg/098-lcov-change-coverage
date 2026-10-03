package main

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

func runMain(t *testing.T, configPath string) int {
	t.Helper()
	old := os.Args
	os.Args = []string{"covergate", "--config", configPath}
	defer func() { os.Args = old }()
	return run()
}

func TestEndToEndPass(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shard1.info", "SF:src/a.go\nDA:1,1\nDA:2,0\nend_of_record\n")
	writeFile(t, dir, "shard2.info", "SF:src/a.go\nDA:2,5\nend_of_record\nSF:src/b.go\nDA:1,1\nend_of_record\n")
	writeFile(t, dir, "changes.json", `{"src/a.go":[2,2,3],"src/b.go":[1]}`)
	cfg := `{
	  "lcov_files": ["shard1.info", "shard2.info"],
	  "changes_file": "changes.json",
	  "output_file": "report.json",
	  "overall_threshold": 10000,
	  "change_threshold": 10000
	}`
	cfgPath := writeFile(t, dir, "task.json", cfg)

	if code := runMain(t, cfgPath); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatalf("report not written: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Pass || rep.Overall.Covered != 3 || rep.Change.Covered != 2 {
		t.Errorf("unexpected report: pass=%v overall=%d/%d change=%d/%d",
			rep.Pass, rep.Overall.Covered, rep.Overall.Total, rep.Change.Covered, rep.Change.Total)
	}
}

func TestEndToEndGateFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.info", "SF:src/a.go\nDA:1,0\nend_of_record\n")
	writeFile(t, dir, "changes.json", `{"src/a.go":[1]}`)
	cfgPath := writeFile(t, dir, "task.json", `{
	  "lcov_files": ["a.info"],
	  "changes_file": "changes.json",
	  "output_file": "report.json",
	  "overall_threshold": 0,
	  "change_threshold": 1
	}`)
	if code := runMain(t, cfgPath); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "report.json")); err != nil {
		t.Error("report should still be written on gate failure")
	}
}

func TestEndToEndInvalidInput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.info", "SF:src/a.go\nDA:1,-3\nend_of_record\n")
	writeFile(t, dir, "changes.json", `{}`)
	cfgPath := writeFile(t, dir, "task.json", `{
	  "lcov_files": ["bad.info"],
	  "changes_file": "changes.json",
	  "output_file": "report.json",
	  "overall_threshold": 0,
	  "change_threshold": 0
	}`)
	if code := runMain(t, cfgPath); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "report.json")); !os.IsNotExist(err) {
		t.Error("report must not be written on invalid input")
	}
}
