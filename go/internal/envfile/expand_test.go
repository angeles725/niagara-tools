package envfile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// outer is the lookup of these tests: the variables the reader's caller knows, outside the file.
func outer(name string) (string, bool) {
	v, ok := map[string]string{"HOME": "/hm", "X": "val", "EMPTY": "", "SECRET": "hunter2"}[name]
	return v, ok
}

// parsed is Parse over text with the outer lookup, flattened; it fails the test on an error.
func parsed(t *testing.T, text string) []string {
	t.Helper()
	got, err := Parse("f", text, outer)
	if err != nil {
		t.Errorf("%q: %v", text, err)
	}
	return pairs(got)
}

func TestAVariableExpandsInBareText(t *testing.T) {
	want := []string{"A=val", "B=val/x", "C=valvalvalz", "D=a.val", "E="}
	if got := parsed(t, "A=$X\nB=${X}/x\nC=$X$X${X}z\nD=a.$X\nE=$EMPTY\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAVariableExpandsInsideDoubleQuotes(t *testing.T) {
	want := []string{"A=val_val-", "B=a val b", "C=val"}
	if got := parsed(t, `A="${X}_$X-"`+"\n"+`B="a $X b"`+"\n"+`C=""$X''`+"\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAVariableIsTextInsideSingleQuotes(t *testing.T) {
	if got := parsed(t, "A='$X ${X} $(x)'\n"); !reflect.DeepEqual(got, []string{"A=$X ${X} $(x)"}) {
		t.Errorf("got %q", got)
	}
}

func TestAnEscapedDollarIsText(t *testing.T) {
	if got := parsed(t, `A=\$X`+"\n"+`B="\$X \${X}"`+"\n"); !reflect.DeepEqual(got, []string{"A=$X", "B=$X ${X}"}) {
		t.Errorf("got %q", got)
	}
}

// A "$" that starts no expansion is text, as under bash.
func TestADollarThatStartsNothingIsText(t *testing.T) {
	want := []string{"A=a$", "B=$/x", "C=$", "D=a$ b", "E=$%", "F=$=", "G=$'x"}
	if got := parsed(t, "A=a$\nB=$/x\nC=\"$\"\nD=\"a$ b\"\nE=$%\nF=$=\nG=\"$'x\"\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAnEarlierAssignmentOfTheFileWinsOverTheEnvironment(t *testing.T) {
	want := []string{"X=file", "A=file", "B=file/b", "Q=1", "R=1"}
	if got := parsed(t, "X=file\nA=$X\nB=\"${X}/b\"\nQ=1 R=$Q\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	// bash expands every argument of an export before it assigns any: a name set on the same export line
	// still has its earlier value
	if got := parsed(t, "export X=file A=$X\nB=$X\n"); !reflect.DeepEqual(got, []string{"X=file", "A=val", "B=file"}) {
		t.Errorf("export line: got %q", got)
	}
}

// The bash tool sourced the file under set -u: an unset variable stopped the deploy.
func TestAnUnsetVariableIsRefusedNamingIt(t *testing.T) {
	for _, text := range []string{"A=1\nB=$UNSET_ONE\n", "A=1\nB=\"x${UNSET_ONE}\"\n", "A=1\nexport C=2 B=$X$UNSET_ONE\n"} {
		_, err := Parse("some.env", text, outer)
		var se *SyntaxError
		if !errors.As(err, &se) || se.Line != 2 || !strings.Contains(se.Reason, "UNSET_ONE") {
			t.Errorf("%q: %v", text, err)
		}
	}
}

func TestAnErrorNeverCarriesAnExpandedValue(t *testing.T) {
	_, err := Parse("some.env", "A=$SECRET$(x)\nB=$SECRET $SECRET\n", outer)
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("error %v", err)
	}
	_, err = Parse("some.env", "$SECRET\n", outer)
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("error %v", err)
	}
}

func TestALeadingTildeIsTheHomeDirectory(t *testing.T) {
	want := []string{"A=/hm", "B=/hm/x", "C=/a:/hm/b:/hm", "D=/hm:x", "E=/hm/val"}
	if got := parsed(t, "A=~\nB=~/x\nC=/a:~/b:~\nexport D=~:x\nE=~/$X\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := parsed(t, "HOME=/other\nA=~/x\n"); !reflect.DeepEqual(got, []string{"HOME=/other", "A=/other/x"}) {
		t.Errorf("HOME set by the file: got %q", got)
	}
}

// Where bash leaves a tilde alone: quoted, escaped, not at the start of the value or after a colon, or
// with a quote, an escape or an expansion right after it.
func TestATildeElsewhereIsText(t *testing.T) {
	want := []string{"A=~/x", "B=~/x", "C=a~/b", "D=~/q", "E=~val", "F=~/x", "G=x=~/y"}
	if got := parsed(t, "A=\"~/x\"\nB='~/x'\nC=a~/b\nD=~\"/q\"\nE=~$X\nF=~\\/x\nG=x=~/y\n"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEveryOtherExpansionIsStillRefused(t *testing.T) {
	for name, line := range map[string]string{
		"command substitution": "A=$(date)", "substitution in quotes": `A="$(date)"`, "backquotes": "A=`date`",
		"arithmetic": "A=$((1+1))", "old arithmetic": "A=$[1+1]",
		"default value": "A=${X:-d}", "alternate": "A=${X-d}", "prefix strip": "A=${X#v}", "suffix strip": "A=${X%l}",
		"replace": "A=${X/a/b}", "length": "A=${#X}", "indirect": "A=${!X}", "unclosed brace": "A=${X", "empty brace": "A=${}",
		"positional": "A=$1", "positional in braces": "A=${1}", "all arguments": "A=$@", "star": "A=$*", "status": "A=$?",
		"process id": "A=$$", "count": "A=$#", "flags": "A=$-", "background id": "A=$!", "last argument": "A=$_",
		"ansi-c quote": "A=$'x'", "locale quote": `A=$"x"`, "expansion in quotes": `A="${X:-d}"`,
		"a user's home": "A=~root/x", "tilde plus": "A=~+", "tilde after colon": "A=/a:~root",
		"an expanded key": "$X=1", "an expanded command": "$X", "operator": "A=1;B=2",
	} {
		got, err := Parse("some.env", "OK=1\n"+line+"\n", outer)
		var se *SyntaxError
		if !errors.As(err, &se) || se.Line != 2 || got != nil {
			t.Errorf("%s (%s): got %q, %v; want a SyntaxError at line 2", name, line, pairs(got), err)
		}
	}
}
