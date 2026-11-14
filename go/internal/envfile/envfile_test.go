package envfile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// nothing is the lookup of a reader with no variable set.
func nothing(string) (string, bool) { return "", false }

// pairs flattens assignments to "KEY=value" for a compact comparison.
func pairs(as []Assignment) []string {
	var out []string
	for _, a := range as {
		out = append(out, a.Key+"="+a.Value)
	}
	return out
}

func TestParseReadsPlainAssignments(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       []string
	}{
		{"bare", "A=1\nB=two\n", []string{"A=1", "B=two"}},
		{"export", "export A=1\n", []string{"A=1"}},
		{"comments and blank lines", "# c\n\n  \t# indented\nA=1 # trailing\n", []string{"A=1"}},
		{"hash inside a word is text", "A=a#b\n", []string{"A=a#b"}},
		{"single quotes are literal", `A='x $HOME \n "q"'` + "\n", []string{`A=x $HOME \n "q"`}},
		{"double quotes keep blanks", `A="b 42"` + "\n", []string{"A=b 42"}},
		{"double quote escapes", `A="a\"b\\c\d\$e"` + "\n", []string{`A=a"b\c\d$e`}},
		{"unquoted backslash escapes one byte", `A=un\quo\ te` + "\n", []string{"A=unquo te"}},
		{"quoted and bare parts join", `A=a'b c'"d e"f` + "\n", []string{"A=ab cd ef"}},
		{"empty values", "A=\nB=''\nC=\"\"\n", []string{"A=", "B=", "C="}},
		{"several on a line", "A=1 B=2\nexport C=3 D=4\n", []string{"A=1", "B=2", "C=3", "D=4"}},
		{"no final newline", "A=1\nB=last", []string{"A=1", "B=last"}},
		{"a later line wins by order", "A=1\nA=2\n", []string{"A=1", "A=2"}},
		{"a carriage return is part of the value", "A=1\r\n", []string{"A=1\r"}},
		{"equals and globs in a value", "A=k=v*?[x]\n", []string{"A=k=v*?[x]"}},
	} {
		got, err := Parse("f", tc.text, nothing)
		if err != nil || !reflect.DeepEqual(pairs(got), tc.want) {
			t.Errorf("%s: got %q, %v; want %q", tc.name, pairs(got), err, tc.want)
		}
	}
}

func TestParseRecordsTheLineOfEachAssignment(t *testing.T) {
	got, err := Parse("f", "# one\nA=1\n\nB=2\n", nothing)
	if err != nil || len(got) != 2 || got[0].Line != 2 || got[1].Line != 4 {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestParseRefusesEveryOtherConstruct(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		line       int
	}{
		{"command substitution", "A=1\nB=$(date)\n", 2},
		{"backquotes", "B=`date`\n", 1},
		{"backquotes in double quotes", "B=\"`x`\"\n", 1},
		{"a command", "echo hello\n", 1},
		{"a command after an assignment", "A=1 make\n", 1},
		{"a function", "f() { :; }\n", 1},
		{"a pipeline or list", "A=1; B=2\n", 1},
		{"a redirection", "A=1 > x\n", 1},
		{"an array", "A=(1 2)\n", 1},
		{"append", "A+=1\n", 1},
		{"export without a value", "export A\n", 1},
		{"export alone", "export\n", 1},
		{"export option", "export -n A=1\n", 1},
		{"a quoted key", "\"A\"=1\n", 1},
		{"a key starting with a digit", "1A=1\n", 1},
		{"brace expansion", "A={a,b}\n", 1},
		{"unterminated single quote", "A='x\nB=2\n", 1},
		{"unterminated double quote", "A=\"x\n", 1},
		{"line continuation", "A=x\\\nB=2\n", 1},
		{"a blank line of a CRLF file", "A=1\n\r\n", 2},
	} {
		got, err := Parse("some.env", tc.text, nothing)
		var se *SyntaxError
		if !errors.As(err, &se) || se.Line != tc.line || se.Path != "some.env" || got != nil {
			t.Errorf("%s: got %q, %v; want a SyntaxError at line %d", tc.name, pairs(got), err, tc.line)
		}
	}
}

func TestSyntaxErrorNamesFileAndLineAndNeverTheValue(t *testing.T) {
	_, err := Parse("/p/some.env", "TOKEN=hunter2$(x)\n", nothing)
	if err == nil || !strings.HasPrefix(err.Error(), "/p/some.env: line 1: ") || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("error %v", err)
	}
}

func TestLoadToleratesAMissingFileAndADirectory(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{dir + "/absent", dir, dir + "/absent/below"} {
		if as, found, err := Load(path, nothing); as != nil || found || err != nil {
			t.Errorf("%s: got %v, %v, %v", path, as, found, err)
		}
	}
}

func TestLoadReadsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e")
	if err := os.WriteFile(path, []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if as, found, err := Load(path, nothing); !found || err != nil || !reflect.DeepEqual(pairs(as), []string{"A=1"}) {
		t.Errorf("got %v, %v, %v", as, found, err)
	}
}

func TestLoadRefusesAFileItCannotCheckOrRead(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "locked", "e")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("A=1\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path, nothing); err == nil {
		t.Error("an unreadable file loaded")
	}
	if err := os.Chmod(filepath.Dir(path), 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(path), 0o700)
	if _, found, err := Load(path, nothing); err == nil || found {
		t.Errorf("a file that cannot be checked: found %v, err %v", found, err)
	}
}
