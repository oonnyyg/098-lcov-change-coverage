package main

type Metric struct {
	Covered   int      `json:"covered"`
	Total     int      `json:"total"`
	Ratio     *float64 `json:"ratio"`
	Threshold int      `json:"threshold"`
	Pass      bool     `json:"pass"`
}

type FileReport struct {
	Path                     string   `json:"path"`
	Covered                  int      `json:"covered"`
	Total                    int      `json:"total"`
	Ratio                    *float64 `json:"ratio"`
	ChangedCovered           int      `json:"changed_covered"`
	ChangedTotal             int      `json:"changed_total"`
	ChangedRatio             *float64 `json:"changed_ratio"`
	UncoveredChangedLines    []int    `json:"uncovered_changed_lines,omitempty"`
	NonExecutableChangeLines []int    `json:"non_executable_changed_lines,omitempty"`
}

type Report struct {
	Pass         bool         `json:"pass"`
	Overall      Metric       `json:"overall"`
	Change       Metric       `json:"change"`
	Files        []FileReport `json:"files"`
	MissingFiles []string     `json:"missing_files"`
}

func ratioPtr(covered, total int) *float64 {
	if total == 0 {
		return nil
	}
	r := float64(covered) / float64(total)
	return &r
}

// thresholdPass compares covered/total >= threshold/10000 exactly using
// integer arithmetic. A zero denominator always passes.
func thresholdPass(covered, total, threshold int) bool {
	if total == 0 {
		return true
	}
	return covered*10000 >= threshold*total
}

func buildReport(cov Coverage, changes Changes, cfg *Config) *Report {
	rep := &Report{Pass: true}

	ovCovered, ovTotal := 0, 0
	for _, path := range sortedKeys(cov) {
		for _, cnt := range cov[path] {
			ovTotal++
			if cnt > 0 {
				ovCovered++
			}
		}
	}
	rep.Overall = Metric{
		Covered:   ovCovered,
		Total:     ovTotal,
		Ratio:     ratioPtr(ovCovered, ovTotal),
		Threshold: cfg.OverallThreshold,
		Pass:      thresholdPass(ovCovered, ovTotal, cfg.OverallThreshold),
	}

	chCovered, chTotal := 0, 0
	missing := []string{}
	fileReports := map[string]*FileReport{}
	for _, path := range sortedKeys(changes) {
		lines, tracked := cov[path]
		if !tracked {
			missing = append(missing, path)
			continue
		}
		fr := &FileReport{Path: path}
		for _, ln := range sortedLines(changes[path]) {
			cnt, executable := lines[ln]
			if !executable {
				fr.NonExecutableChangeLines = append(fr.NonExecutableChangeLines, ln)
				continue
			}
			chTotal++
			fr.ChangedTotal++
			if cnt > 0 {
				chCovered++
				fr.ChangedCovered++
			} else {
				fr.UncoveredChangedLines = append(fr.UncoveredChangedLines, ln)
			}
		}
		fr.ChangedRatio = ratioPtr(fr.ChangedCovered, fr.ChangedTotal)
		fileReports[path] = fr
	}
	rep.Change = Metric{
		Covered:   chCovered,
		Total:     chTotal,
		Ratio:     ratioPtr(chCovered, chTotal),
		Threshold: cfg.ChangeThreshold,
		Pass:      thresholdPass(chCovered, chTotal, cfg.ChangeThreshold),
	}
	rep.MissingFiles = missing

	paths := map[string]bool{}
	for p := range cov {
		paths[p] = true
	}
	for p := range changes {
		paths[p] = true
	}
	for _, p := range sortedKeys(paths) {
		fr, ok := fileReports[p]
		if !ok {
			fr = &FileReport{Path: p}
		}
		for _, cnt := range cov[p] {
			fr.Total++
			if cnt > 0 {
				fr.Covered++
			}
		}
		fr.Ratio = ratioPtr(fr.Covered, fr.Total)
		rep.Files = append(rep.Files, *fr)
	}

	rep.Pass = rep.Overall.Pass && rep.Change.Pass && len(rep.MissingFiles) == 0
	return rep
}
