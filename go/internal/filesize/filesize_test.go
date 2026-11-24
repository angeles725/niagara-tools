package filesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const why = "a reason long enough"

// writeLines writes rel under root with n lines.
func writeLines(t *testing.T, root, rel string, n int) {
	t.Helper()
	writeText(t, root, rel, strings.Repeat("x\n", n))
}

func writeText(t *testing.T, root, rel, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// run checks a tree holding one file a/f.go of n lines against debt (the entry lines, no header) and returns the
// violations.
func run(t *testing.T, n int, debt string) []string {
	t.Helper()
	root := t.TempDir()
	writeLines(t, root, "a/f.go", n)
	return runTree(t, root, debt)
}

func runTree(t *testing.T, root, debt string) []string {
	t.Helper()
	debtPath := filepath.Join(root, DebtFileName)
	if err := os.WriteFile(debtPath, []byte("# header\n"+debt), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Check(root, debtPath)
	if err != nil {
		t.Fatal(err)
	}
	return r.Violations
}

// want fails unless got is exactly the messages in exp.
func want(t *testing.T, got []string, exp ...string) {
	t.Helper()
	if len(got) != len(exp) {
		t.Fatalf("violations = %q, want %q", got, exp)
	}
	for i := range exp {
		if got[i] != exp[i] {
			t.Fatalf("violation %d =\n%s\nwant\n%s", i, got[i], exp[i])
		}
	}
}

const dbt = "go/internal/filesize/modularity-debt.txt"

func TestA200LineFileNeedsNoEntry(t *testing.T) {
	want(t, run(t, 200, ""))
}

func TestAnUnlistedFileOver200FailsWithTheLineToAdd(t *testing.T) {
	want(t, run(t, 201, ""),
		"a/f.go: 201 lines, over 200 and not listed: split it by responsibility, or add this line to "+dbt+":\n"+
			"a/f.go 201 accepted <reason>")
}

func TestAnUnlistedFileOver400CannotBeListed(t *testing.T) {
	want(t, run(t, 401, ""),
		"a/f.go: 401 lines, over 400: a file over 400 lines cannot be listed, split it by responsibility. "+
			"If it is a listed file that moved, change the path on its line instead (same count and kind): that is not a new entry.")
}

func TestAnAcceptedFileMayStayOrShrinkFreely(t *testing.T) {
	want(t, run(t, 300, "a/f.go 300 accepted "+why+"\n"))
	want(t, run(t, 250, "a/f.go 300 accepted "+why+"\n"))
}

func TestAnAcceptedFileMayGrowOnlyWithItsLineRaised(t *testing.T) {
	want(t, run(t, 301, "a/f.go 300 accepted "+why+"\n"),
		"a/f.go: grew from 300 to 301 lines: split it by responsibility, or raise its line in "+dbt+" to:\n"+
			"a/f.go 301 accepted "+why)
	want(t, run(t, 301, "a/f.go 301 accepted "+why+"\n"))
	want(t, run(t, 400, "a/f.go 400 accepted "+why+"\n"))
}

func TestAnAcceptedFileNeverPassesTheHardLimit(t *testing.T) {
	want(t, run(t, 401, "a/f.go 401 accepted "+why+"\n"),
		"modularity-debt.txt:2: an accepted entry may not be above 400 lines (a/f.go 401): split it by responsibility.")
	want(t, run(t, 401, "a/f.go 400 accepted "+why+"\n"),
		"a/f.go: grew from 400 to 401 lines and is over 400: a file over 400 lines may not grow, split it by responsibility first.")
}

func TestALegacyCountOver400IsExact(t *testing.T) {
	leg := "a/f.go 500 legacy " + why + "\n"
	want(t, run(t, 500, leg))
	want(t, run(t, 501, leg),
		"a/f.go: grew from 500 to 501 lines and is over 400: a file over 400 lines may not grow, split it by responsibility first.")
	want(t, run(t, 480, leg),
		"a/f.go: 480 lines, below its recorded 500: a count over 400 is exact, so lower its line in "+dbt+" to:\n"+
			"a/f.go 480 legacy "+why)
	want(t, run(t, 390, leg),
		"a/f.go: 390 lines, below its recorded 500: a count over 400 is exact, so lower its line in "+dbt+" to:\n"+
			"a/f.go 390 legacy "+why)
}

func TestALegacyCountOf400OrLessIsACeilingLikeAccepted(t *testing.T) {
	leg := "a/f.go 300 legacy " + why + "\n"
	want(t, run(t, 250, leg))
	want(t, run(t, 300, leg))
	want(t, run(t, 350, leg),
		"a/f.go: grew from 300 to 350 lines: split it by responsibility, or raise its line in "+dbt+" to:\n"+
			"a/f.go 350 legacy "+why)
	want(t, run(t, 401, leg),
		"a/f.go: grew from 300 to 401 lines and is over 400: a file over 400 lines may not grow, split it by responsibility first.")
}

func TestAListedFileAt200OrLessMustDeleteItsLine(t *testing.T) {
	line := "a/f.go 300 accepted " + why
	want(t, run(t, 200, line+"\n"),
		"a/f.go: 200 lines, no longer over 200: delete this line from "+dbt+":\n"+line)
	want(t, run(t, 10, line+"\n"),
		"a/f.go: 10 lines, no longer over 200: delete this line from "+dbt+":\n"+line)
}

func TestAListedPathThatIsNoProductionFileMustBeDeletedOrMoved(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/ok.go", 10)
	line := "a/gone.go 300 accepted " + why
	want(t, runTree(t, root, line+"\n"),
		"a/gone.go: listed but not a production Go file in this tree. Delete this line from "+dbt+
			", or, if the file moved, change the path on it (same count and kind, list kept sorted):\n"+line)
}

func TestABadLineIsReportedAndNeverTakenForAnUnlistedFile(t *testing.T) {
	const form = `; expected "<path> <lines> <legacy|accepted> <reason of at least 10 characters>", entries sorted by path`
	for _, c := range []struct{ line, problem string }{
		{"a/f.go 300 accepted", "malformed line, fewer than four fields"},
		{"a/f.go many accepted " + why, `"many" is not a positive line count`},
		{"a/f.go 0 accepted " + why, `"0" is not a positive line count`},
		{"a/f.go 300 tolerated " + why, `unknown kind "tolerated"`},
		{"a/f.go 300 accepted short", "reason is shorter than 10 characters"},
		{"a/f.go 300 accepted <reason>", "replace <reason> with the reason this file is accepted"},
	} {
		want(t, run(t, 250, c.line+"\n"), "modularity-debt.txt:2: "+c.problem+form)
	}
}

func TestUnsortedAndDuplicateLinesFail(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/b.go", 250)
	writeLines(t, root, "a/c.go", 250)
	const form = `; expected "<path> <lines> <legacy|accepted> <reason of at least 10 characters>", entries sorted by path`
	got := runTree(t, root, "a/c.go 250 accepted "+why+"\na/b.go 250 accepted "+why+"\n")
	want(t, got, "modularity-debt.txt:3: a/b.go is not after a/c.go (unsorted)"+form)
	got = runTree(t, root, "a/b.go 250 accepted "+why+"\na/b.go 250 accepted "+why+"\na/c.go 250 accepted "+why+"\n")
	want(t, got, "modularity-debt.txt:3: duplicate path a/b.go"+form)
}

func TestSortingIsByBytesNotByLocale(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "B/x.go", 250)
	writeLines(t, root, "a/x.go", 250)
	want(t, runTree(t, root, "B/x.go 250 accepted "+why+"\na/x.go 250 accepted "+why+"\n"))
}

func TestAGeneratedFileIsExemptButOnlyBeforeThePackageClause(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("// x\n", 300)
	writeText(t, root, "a/gen.go", "// Code generated by tool. DO NOT EDIT.\n\npackage a\n"+body)
	writeText(t, root, "a/late.go", "package a\n// Code generated by tool. DO NOT EDIT.\n"+body)
	writeLines(t, root, "a/ok.go", 10)
	want(t, runTree(t, root, ""),
		"a/late.go: 302 lines, over 200 and not listed: split it by responsibility, or add this line to "+dbt+":\n"+
			"a/late.go 302 accepted <reason>")
}

func TestTestsTestdataAndIgnoredPathsAreNotMeasured(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/big_test.go", 900)
	writeLines(t, root, "a/testdata/big.go", 900)
	writeLines(t, root, "bin/big.go", 900)
	writeLines(t, root, "a/_draft/big.go", 900)
	writeLines(t, root, "a/.hidden/big.go", 900)
	writeLines(t, root, "a/_big.go", 900)
	writeLines(t, root, "a/notgo.txt", 900)
	writeLines(t, root, "a/ok.go", 10)
	want(t, runTree(t, root, ""))
}

func TestAMissingDebtFileOrAnEmptyTreeIsAnErrorNotAPass(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/ok.go", 10)
	if _, err := Check(root, filepath.Join(root, "nope.txt")); err == nil {
		t.Fatal("a missing debt file passed")
	}
	empty := t.TempDir()
	debt := filepath.Join(empty, DebtFileName)
	if err := os.WriteFile(debt, []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(empty, debt); err == nil {
		t.Fatal("a tree with no production Go file passed")
	}
}

func TestTheResultSaysWhatWasLookedAt(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/ok.go", 10)
	writeLines(t, root, "a/big.go", 250)
	writeText(t, root, "a/gen.go", "// Code generated by x. DO NOT EDIT.\npackage a\n")
	debt := filepath.Join(root, DebtFileName)
	if err := os.WriteFile(debt, []byte("a/big.go 250 accepted "+why+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Check(root, debt)
	if err != nil {
		t.Fatal(err)
	}
	if r.Scanned != 2 || r.Generated != 1 || r.Over != 1 || r.Entries != 1 || len(r.Violations) != 0 {
		t.Fatalf("result = %+v, want scanned 2, generated 1, over 1, entries 1, no violation", r)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdateLowersCountsAndDeletesLinesOfFilesThatAreGoneOrSmall(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/exact.go", 500)
	writeLines(t, root, "a/shrunk.go", 480)
	writeLines(t, root, "a/ceiling.go", 250)
	writeLines(t, root, "a/small.go", 200)
	writeLines(t, root, "a/grew.go", 350)
	debt := filepath.Join(root, DebtFileName)
	in := "# header\n\n" +
		"a/ceiling.go 300 accepted " + why + "\n" +
		"a/exact.go 500 legacy " + why + "\n" +
		"a/gone.go 300 accepted " + why + "\n" +
		"a/grew.go 300 accepted " + why + "\n" +
		"a/shrunk.go 500 legacy " + why + "\n" +
		"a/small.go 300 accepted " + why + "\n" +
		"a/bad.go 5 nonsense " + why + "\n"
	if err := os.WriteFile(debt, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := Update(root, debt)
	if err != nil || !changed {
		t.Fatalf("Update = %v, %v, want a rewrite", changed, err)
	}
	out := "# header\n\n" +
		"a/ceiling.go 250 accepted " + why + "\n" +
		"a/exact.go 500 legacy " + why + "\n" +
		"a/grew.go 300 accepted " + why + "\n" +
		"a/shrunk.go 480 legacy " + why + "\n" +
		"a/bad.go 5 nonsense " + why + "\n"
	if got := readFile(t, debt); got != out {
		t.Fatalf("debt after update =\n%s\nwant\n%s", got, out)
	}
	if changed, err := Update(root, debt); err != nil || changed {
		t.Fatalf("a second Update = %v, %v, want no rewrite", changed, err)
	}
}

func TestUpdateOnlyRunsWhenTheRequestIsExactly1(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/f.go", 250)
	debt := filepath.Join(root, DebtFileName)
	in := "a/f.go 300 accepted " + why + "\n"
	if err := os.WriteFile(debt, []byte(in), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, req := range []string{"", "0", "true", "yes", "11"} {
		if changed, err := UpdateIfRequested(req, root, debt); err != nil || changed {
			t.Fatalf("request %q: %v, %v, want nothing written", req, changed, err)
		}
	}
	if readFile(t, debt) != in {
		t.Fatal("the debt file changed without a request of exactly 1")
	}
	if changed, err := UpdateIfRequested("1", root, debt); err != nil || !changed {
		t.Fatalf("request 1: %v, %v, want a rewrite", changed, err)
	}
	if got := readFile(t, debt); got != "a/f.go 250 accepted "+why+"\n" {
		t.Fatalf("debt = %q", got)
	}
}

// TestTheGoTreeKeepsItsFilesSmall is the gate: it measures this repository's go/ tree against modularity-debt.txt.
// With NT_FILESIZE_UPDATE=1 it first lowers or deletes stale lines (never raising a count or adding an entry).
func TestTheGoTreeKeepsItsFilesSmall(t *testing.T) {
	root := filepath.Join("..", "..")
	debt := DebtFileName
	changed, err := UpdateIfRequested(os.Getenv(UpdateEnv), root, debt)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Logf("update mode: %s was rewritten", DebtFileName)
	}
	r, err := Check(root, debt)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range r.Violations {
		t.Error(v)
	}
	if len(r.Violations) > 0 {
		t.Log("the rule and how to fix a finding: go/internal/filesize/filesize.go and go/README.md")
	}
	if r.Scanned == 0 {
		t.Fatal("scanned no file")
	}
}
