package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Coverage maps file path -> line number -> hit count.
type Coverage map[string]map[int]int64

func (c Coverage) add(path string, line int, count int64) {
	lines, ok := c[path]
	if !ok {
		lines = map[int]int64{}
		c[path] = lines
	}
	lines[line] += count
}

// parseLCOV parses an LCOV stream containing only SF, DA and end_of_record
// records (TN, LF, LH are ignored). Any other record is an error.
func parseLCOV(r io.Reader, name string) (Coverage, error) {
	cov := Coverage{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lineNo := 0
	inSection := false
	var curPath string
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "SF:"):
			if inSection {
				return nil, fmt.Errorf("%s:%d: nested SF section for %q (previous %q not closed)", name, lineNo, line[3:], curPath)
			}
			path := strings.TrimSpace(line[3:])
			if err := validatePath(path); err != nil {
				return nil, fmt.Errorf("%s:%d: invalid SF path: %v", name, lineNo, err)
			}
			inSection = true
			curPath = path
		case strings.HasPrefix(line, "DA:"):
			if !inSection {
				return nil, fmt.Errorf("%s:%d: DA record outside of SF section", name, lineNo)
			}
			ln, cnt, err := parseDA(line[3:])
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", name, lineNo, err)
			}
			cov.add(curPath, ln, cnt)
		case line == "end_of_record":
			if !inSection {
				return nil, fmt.Errorf("%s:%d: end_of_record without open SF section", name, lineNo)
			}
			inSection = false
			curPath = ""
		case strings.HasPrefix(line, "TN:") || strings.HasPrefix(line, "LF:") || strings.HasPrefix(line, "LH:"):
			// ignored records
		default:
			return nil, fmt.Errorf("%s:%d: unsupported record %q", name, lineNo, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: read error: %v", name, err)
	}
	if inSection {
		return nil, fmt.Errorf("%s: unterminated SF section for %q (missing end_of_record)", name, curPath)
	}
	return cov, nil
}

func parseDA(body string) (int, int64, error) {
	parts := strings.Split(body, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("malformed DA record %q (expected DA:<line>,<count>)", body)
	}
	ln, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || ln <= 0 {
		return 0, 0, fmt.Errorf("invalid DA line number %q (must be a positive integer)", parts[0])
	}
	cnt, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || cnt < 0 {
		return 0, 0, fmt.Errorf("invalid DA hit count %q (must be a non-negative integer)", parts[1])
	}
	return ln, cnt, nil
}

// validatePath requires a forward-slash relative path with no empty or dot
// segments. Paths are case sensitive and kept verbatim.
func validatePath(p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}
	if strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
		return fmt.Errorf("path %q must be a relative forward-slash path", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return fmt.Errorf("path %q contains an empty segment", p)
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("path %q contains a dot segment", p)
		}
	}
	return nil
}

// merge adds all counts from other into c.
func (c Coverage) merge(other Coverage) {
	for path, lines := range other {
		for ln, cnt := range lines {
			c.add(path, ln, cnt)
		}
	}
}
