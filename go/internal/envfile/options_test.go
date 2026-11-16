package envfile

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestTrimCRStripsOneCarriageReturnFromTheEndOfEachLine(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       []string
		lines      []int
	}{
		{"a CRLF file", "A=1\r\n# c\r\n\r\nexport B='x y'\r\nC=\"$A\"\r\n", []string{"A=1", "B=x y", "C=1"}, []int{1, 2, 3, 4, 5}},
		{"only some lines", "A=1\nB=2\r\nC=3\n", []string{"A=1", "B=2", "C=3"}, []int{2}},
		{"no final newline", "A=1\r\nB=2\r", []string{"A=1", "B=2"}, []int{1, 2}},
		{"an LF file is read as without the option", "A=1\n\nB=2\n", []string{"A=1", "B=2"}, nil},
		{"only the last one of a line", "A=1\r\r\n", []string{"A=1\r"}, []int{1}},
		{"one inside a value stays", "A=a\rb\n", []string{"A=a\rb"}, nil},
		{"one inside quotes stays", "A='a\r'\n", []string{"A=a\r"}, nil},
	} {
		var lines []int
		got, err := Parse("f", tc.text, nothing, TrimCR(&lines))
		if err != nil || !reflect.DeepEqual(pairs(got), tc.want) || !reflect.DeepEqual(lines, tc.lines) {
			t.Errorf("%s: got %q, lines %v, %v; want %q, lines %v", tc.name, pairs(got), lines, err, tc.want, tc.lines)
		}
	}
}

func TestTrimCRKeepsTheLineNumbersAndTheRefusals(t *testing.T) {
	var lines []int
	got, err := Parse("f", "# one\r\nA=1\r\n\r\nB=2\r\n", nothing, TrimCR(&lines))
	if err != nil || len(got) != 2 || got[0].Line != 2 || got[1].Line != 4 {
		t.Errorf("got %+v, %v", got, err)
	}
	_, err = Parse("some.env", "A=1\r\necho x\r\n", nothing, TrimCR(&lines))
	var se *SyntaxError
	if !errors.As(err, &se) || se.Line != 2 {
		t.Errorf("a command in a CRLF file: %v", err)
	}
}

// Without the option a reader gets what it always got: the carriage return stays in the value, and a line
// holding only one is refused (TestParseReadsPlainAssignments and TestParseRefusesEveryOtherConstruct pin
// the same through the plain call).
func TestWithoutTrimCRTheCarriageReturnStays(t *testing.T) {
	got, err := Parse("f", "A=1\r\nB=2\r\n", nothing)
	if err != nil || !reflect.DeepEqual(pairs(got), []string{"A=1\r", "B=2\r"}) {
		t.Errorf("got %q, %v", pairs(got), err)
	}
	if _, err := Parse("f", "A=1\r\n\r\n", nothing); err == nil {
		t.Error("a blank CRLF line was accepted without TrimCR")
	}
}

func TestLoadPassesItsOptionsOn(t *testing.T) {
	path := t.TempDir() + "/e"
	if err := os.WriteFile(path, []byte("A=1\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var lines []int
	if as, found, err := Load(path, nothing, TrimCR(&lines)); !found || err != nil || !reflect.DeepEqual(pairs(as), []string{"A=1"}) || !reflect.DeepEqual(lines, []int{1}) {
		t.Errorf("got %v, %v, %v, lines %v", as, found, err, lines)
	}
	if as, _, err := Load(path, nothing); err != nil || !reflect.DeepEqual(pairs(as), []string{"A=1\r"}) {
		t.Errorf("without the option: got %q, %v", pairs(as), err)
	}
}
