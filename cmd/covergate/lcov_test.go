package main

import (
	"strings"
	"testing"
)

func TestParseLCOVBasic(t *testing.T) {
	in := `TN:
SF:src/a.go
DA:1,2
DA:2,0
LF:2
LH:1
end_of_record
SF:src/a.go
DA:1,3
end_of_record
SF:src/b.go
DA:5,1
end_of_record
`
	cov, err := parseLCOV(strings.NewReader(in), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cov["src/a.go"][1]; got != 5 {
		t.Errorf("src/a.go:1 count = %d, want 5 (summed across sections)", got)
	}
	if got := cov["src/a.go"][2]; got != 0 {
		t.Errorf("src/a.go:2 count = %d, want 0", got)
	}
	if got := cov["src/b.go"][5]; got != 1 {
		t.Errorf("src/b.go:5 count = %d, want 1", got)
	}
}

func TestParseLCOVErrors(t *testing.T) {
	cases := map[string]string{
		"unterminated section": "SF:src/a.go\nDA:1,1\n",
		"nested section":       "SF:src/a.go\nSF:src/b.go\nend_of_record\n",
		"DA outside section":   "DA:1,1\n",
		"end without section":  "end_of_record\n",
		"unsupported record":   "SF:src/a.go\nBRDA:1,0,0,1\nend_of_record\n",
		"zero line":            "SF:src/a.go\nDA:0,1\nend_of_record\n",
		"negative line":        "SF:src/a.go\nDA:-2,1\nend_of_record\n",
		"negative count":       "SF:src/a.go\nDA:1,-1\nend_of_record\n",
		"non-integer count":    "SF:src/a.go\nDA:1,x\nend_of_record\n",
		"checksum in DA":       "SF:src/a.go\nDA:1,2,abc\nend_of_record\n",
		"empty path segment":   "SF:src//a.go\nend_of_record\n",
		"dot segment":          "SF:src/../a.go\nend_of_record\n",
		"absolute path":        "SF:/src/a.go\nend_of_record\n",
		"backslash path":       "SF:src\\a.go\nend_of_record\n",
	}
	for name, in := range cases {
		if _, err := parseLCOV(strings.NewReader(in), "test"); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		} else if !strings.Contains(err.Error(), "test:") && !strings.Contains(err.Error(), "test") {
			t.Errorf("%s: error %q should locate the input", name, err)
		}
	}
}

func TestMerge(t *testing.T) {
	a := Coverage{"f.go": {1: 1, 2: 0}}
	b := Coverage{"f.go": {1: 2, 3: 4}, "g.go": {1: 1}}
	a.merge(b)
	if a["f.go"][1] != 3 || a["f.go"][2] != 0 || a["f.go"][3] != 4 || a["g.go"][1] != 1 {
		t.Errorf("merge result wrong: %v", a)
	}
}
