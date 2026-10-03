// Package lcov parses LCOV tracefiles, supporting only SF, DA and
// end_of_record records. TN, LF and LH records are ignored; any other
// record is an error.
package lcov

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// File holds the merged line hit counts for one source path.
type File struct {
	Path string
	Hits map[int]int64
}

// ParseError describes a malformed record with its location.
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// Parse reads an LCOV stream and returns one File per SF section, in
// order of appearance.
func Parse(r io.Reader) ([]File, error) {
	var files []File
	var cur *File
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "SF:"):
			if cur != nil {
				return nil, &ParseError{lineNo, "SF record inside unclosed section"}
			}
			path := strings.TrimPrefix(line, "SF:")
			if err := ValidatePath(path); err != nil {
				return nil, &ParseError{lineNo, err.Error()}
			}
			cur = &File{Path: path, Hits: map[int]int64{}}
		case strings.HasPrefix(line, "DA:"):
			if cur == nil {
				return nil, &ParseError{lineNo, "DA record outside of a section"}
			}
			lineNum, count, err := parseDA(strings.TrimPrefix(line, "DA:"))
			if err != nil {
				return nil, &ParseError{lineNo, err.Error()}
			}
			cur.Hits[lineNum] += count
		case line == "end_of_record":
			if cur == nil {
				return nil, &ParseError{lineNo, "end_of_record outside of a section"}
			}
			files = append(files, *cur)
			cur = nil
		case strings.HasPrefix(line, "TN:"),
			strings.HasPrefix(line, "LF:"),
			strings.HasPrefix(line, "LH:"):
			// ignored
		default:
			return nil, &ParseError{lineNo, fmt.Sprintf("unsupported record %q", line)}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if cur != nil {
		return nil, &ParseError{lineNo, "section not closed by end_of_record"}
	}
	return files, nil
}

func parseDA(body string) (int, int64, error) {
	parts := strings.Split(body, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("DA record must have exactly line and count fields: %q", body)
	}
	lineNum, err := strconv.Atoi(parts[0])
	if err != nil || lineNum <= 0 {
		return 0, 0, fmt.Errorf("DA record has invalid line number %q", parts[0])
	}
	count, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || count < 0 {
		return 0, 0, fmt.Errorf("DA record has invalid hit count %q", parts[1])
	}
	return lineNum, count, nil
}

// ValidatePath checks that a source path is a case-sensitive relative
// path using forward slashes, without empty or dot segments.
func ValidatePath(p string) error {
	if p == "" {
		return fmt.Errorf("empty source path")
	}
	if strings.HasPrefix(p, "/") || strings.Contains(p, `\`) || strings.Contains(p, ":") {
		return fmt.Errorf("source path must be a relative path with forward slashes: %q", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			return fmt.Errorf("source path has an empty segment: %q", p)
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("source path has a dot segment: %q", p)
		}
	}
	return nil
}
