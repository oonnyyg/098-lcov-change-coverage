// Command covergate merges LCOV coverage shards and checks overall and
// per-change coverage thresholds.
//
// Usage: go run ./cmd/covergate --config 任务.json
//
// Exit codes: 0 gate passed, 1 thresholds failed, 2 invalid input.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/lcov-change-coverage/internal/gate"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "", "path to the JSON task config")
	flag.Parse()
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "covergate: --config is required")
		return 2
	}
	cfg, err := gate.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		return 2
	}
	changes, err := gate.LoadChanges(cfg.ChangesFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		return 2
	}
	rep, err := gate.Run(cfg, changes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		return 2
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "covergate: marshal report:", err)
		return 2
	}
	data = append(data, '\n')
	if err := os.WriteFile(cfg.Output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "covergate: write report:", err)
		return 2
	}
	if !rep.Passed {
		return 1
	}
	return 0
}
