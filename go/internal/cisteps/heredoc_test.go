package cisteps

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// form675 is the #675 shape: a fixture written through a here-document whose lines start with the
// test keyword. bats 1.10 reads them as cases of the file and refuses it (Duplicate test name(s)).
const form675 = `@test "SH2-neg: fixture" {
  cat > "$TMPDIR_T/bad.bats" <<'SH'
SCRIPT="$REPO/scripts/install-skill.sh"
@test "a" { run "$SCRIPT" --dry-run; }
  @test "b" {
}
SH
  run lint "$TMPDIR_T/bad.bats"
}
`

func TestHeredocLineStartingWithTheTestKeywordIsFlagged(t *testing.T) {
	got := HeredocTestLines([]byte(form675))
	if want := []int{4, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines %v, want %v", got, want)
	}
}

func TestHeredocLintLeavesRealCasesAndThePlaceholderAlone(t *testing.T) {
	clean := `@test "one" {
  sed 's/^%test /@test /' > "$f" <<'SH'
%test "a" { :; }
SH
  x=$((1<<2))
  # cat <<EOF  (a comment opens nothing)
  grep -q x <<< "@test not a heredoc"
  printf '#!/bin/sh\ncat <<"TAP"\n' > stub   # a quoted << writes a here-document, it opens none here
  printf 'cat > g <<'"'"'EOF'"'"'\n' > other
}
@test "two" { :; }
`
	if got := HeredocTestLines([]byte(clean)); len(got) != 0 {
		t.Errorf("flagged %v in a clean file", got)
	}
}

func TestHeredocLintFollowsEveryDelimiterForm(t *testing.T) {
	src := "@test \"x\" {\n" +
		"\tcat <<-\"END\" > a\n\t@test \"tab\" {\n\tEND\n" + // <<- strips leading tabs from the delimiter line
		"  cat <<EOF > b; cat <<\\TWO > c\n@test \"first\" {\nEOF\n@test \"second\" {\nTWO\n" + // two on one line
		"}\n@test \"after\" { :; }\n"
	want := []int{3, 6, 8}
	if got := HeredocTestLines([]byte(src)); !reflect.DeepEqual(got, want) {
		t.Errorf("lines %v, want %v", got, want)
	}
}

func TestLintBatsHeredocCommandRowsAndExits(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "tests", "bad.bats")
	if err := os.MkdirAll(filepath.Join(dir, "tests", "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(form675), 0o644); err != nil {
		t.Fatal(err)
	}
	// a fixture tree is pruned: its files are data of a test, not cases bats loads
	if err := os.WriteFile(filepath.Join(dir, "tests", "fixtures", "x.bats"), []byte(form675), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Main([]string{"lint-bats-heredoc", filepath.Join(dir, "tests")}, cli.Stdio{Out: &out, Err: &errb})
	if code != 1 {
		t.Fatalf("exit %d, want 1\n%s%s", code, out.String(), errb.String())
	}
	rows := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(rows) != 2 || !strings.HasPrefix(rows[0], "FAIL  bats-heredoc-test  "+bad+":4  ") {
		t.Errorf("rows:\n%s", out.String())
	}
	// a directory with no *.bats is an empty input set: exit 3, as for every kit lint
	if code := Main([]string{"lint-bats-heredoc", t.TempDir()}, cli.Stdio{Out: &out, Err: &errb}); code != 3 {
		t.Errorf("empty root: exit %d, want 3", code)
	}
	if code := Main([]string{"lint-bats-heredoc", filepath.Join(dir, "missing.bats")}, cli.Stdio{Out: &out, Err: &errb}); code != 3 {
		t.Errorf("unreadable input: exit %d, want 3", code)
	}
	clean := filepath.Join(dir, "clean.bats")
	if err := os.WriteFile(clean, []byte("@test \"a\" { :; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := Main([]string{"lint-bats-heredoc", clean}, cli.Stdio{Out: &out, Err: &errb}); code != 0 || out.Len() != 0 {
		t.Errorf("clean file: exit %d, out %q", code, out.String())
	}
}
