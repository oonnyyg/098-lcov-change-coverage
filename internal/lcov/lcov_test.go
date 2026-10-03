package lcov

import (
	"strings"
	"testing"
)

func TestParseMergesDuplicateLines(t *testing.T) {
	in := `TN:
SF:src/a.go
DA:1,2
DA:1,3
DA:5,0
LF:2
LH:1
end_of_record
SF:src/b.go
DA:10,1
end_of_record
`
	files, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].Hits[1] != 5 || files[0].Hits[5] != 0 {
		t.Fatalf("unexpected hits: %v", files[0].Hits)
	}
	if files[1].Path != "src/b.go" || files[1].Hits[10] != 1 {
		t.Fatalf("unexpected second file: %+v", files[1])
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"unclosed section":  "SF:a.go\nDA:1,1\n",
		"nested section":    "SF:a.go\nSF:b.go\nend_of_record\n",
		"da outside":        "DA:1,1\n",
		"end outside":       "end_of_record\n",
		"negative count":    "SF:a.go\nDA:1,-2\nend_of_record\n",
		"non-integer count": "SF:a.go\nDA:1,x\nend_of_record\n",
		"zero line":         "SF:a.go\nDA:0,1\nend_of_record\n",
		"checksum field":    "SF:a.go\nDA:1,1,abc\nend_of_record\n",
		"unknown record":    "SF:a.go\nFN:1,f\nend_of_record\n",
		"absolute path":     "SF:/a.go\nend_of_record\n",
		"dot segment":       "SF:a/../b.go\nend_of_record\n",
		"empty segment":     "SF:a//b.go\nend_of_record\n",
		"backslash":         `SF:a\b.go` + "\nend_of_record\n",
	}
	for name, in := range cases {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestValidatePath(t *testing.T) {
	for _, ok := range []string{"a.go", "src/deep/file.go", "Case/Sensitive.GO"} {
		if err := ValidatePath(ok); err != nil {
			t.Errorf("ValidatePath(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "/a.go", "a/./b", "a//b", "./a.go", "C:/a.go", `a\b`} {
		if err := ValidatePath(bad); err == nil {
			t.Errorf("ValidatePath(%q) = nil, want error", bad)
		}
	}
}
