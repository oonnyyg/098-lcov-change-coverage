package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("covergate", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to the task configuration JSON file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "covergate: --config is required")
		return 2
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "covergate: %v\n", err)
		return 2
	}

	cov := Coverage{}
	for _, f := range cfg.LCOVFiles {
		fh, err := os.Open(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "covergate: open lcov file: %v\n", err)
			return 2
		}
		part, err := parseLCOV(fh, f)
		fh.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "covergate: %v\n", err)
			return 2
		}
		cov.merge(part)
	}

	changes, err := loadChanges(cfg.ChangesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "covergate: %v\n", err)
		return 2
	}

	rep := buildReport(cov, changes, cfg)

	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "covergate: encode report: %v\n", err)
		return 2
	}
	data = append(data, '\n')
	if err := os.WriteFile(cfg.OutputFile, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "covergate: write report: %v\n", err)
		return 2
	}

	if !rep.Pass {
		fmt.Fprintln(os.Stderr, "covergate: coverage gate failed")
		return 1
	}
	return 0
}
