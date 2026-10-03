package gate

import "sort"

func evaluate(cfg *Config, changes Changes, hits map[string]map[int]int64) *Report {
	rep := &Report{Passed: true, Files: []FileReport{}, MissingFiles: []string{}}
	rep.Overall.UncoveredChangeLines = []int{}
	rep.Overall.NonExecutableLines = []int{}
	paths := make([]string, 0, len(hits))
	for p := range hits {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	overall := &rep.Overall
	overall.Path = ""
	for _, p := range paths {
		fr := FileReport{Path: p, UncoveredChangeLines: []int{}, NonExecutableLines: []int{}}
		for _, count := range hits[p] {
			fr.Total++
			if count > 0 {
				fr.Covered++
			}
		}
		if set, ok := changes[p]; ok {
			for line := range set {
				count, executable := hits[p][line]
				if !executable {
					fr.NonExecutableLines = append(fr.NonExecutableLines, line)
					continue
				}
				fr.ChangeTotal++
				if count > 0 {
					fr.ChangeCovered++
				} else {
					fr.UncoveredChangeLines = append(fr.UncoveredChangeLines, line)
				}
			}
			sort.Ints(fr.UncoveredChangeLines)
			sort.Ints(fr.NonExecutableLines)
		}
		fr.Ratio = Ratio{fr.Covered, fr.Total}
		fr.ChangeRatio = Ratio{fr.ChangeCovered, fr.ChangeTotal}
		rep.Files = append(rep.Files, fr)
		overall.Covered += fr.Covered
		overall.Total += fr.Total
		overall.ChangeCovered += fr.ChangeCovered
		overall.ChangeTotal += fr.ChangeTotal
	}
	overall.Ratio = Ratio{overall.Covered, overall.Total}
	overall.ChangeRatio = Ratio{overall.ChangeCovered, overall.ChangeTotal}

	for p := range changes {
		if _, ok := hits[p]; !ok {
			rep.MissingFiles = append(rep.MissingFiles, p)
		}
	}
	sort.Strings(rep.MissingFiles)

	if len(rep.MissingFiles) > 0 {
		rep.Passed = false
	}
	if !overall.Ratio.meets(cfg.OverallThreshold) || !overall.ChangeRatio.meets(cfg.ChangeThreshold) {
		rep.Passed = false
	}
	return rep
}
