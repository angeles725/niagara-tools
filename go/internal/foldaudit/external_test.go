package foldaudit

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fakeGH answers issue listings and views; viewState maps an issue number to its state.
type fakeGH struct {
	labeled   string
	viewState map[string]string
	err       error
	calls     []string
}

func (g *fakeGH) run(args []string, _ string) (string, error) {
	line := strings.Join(args, " ")
	g.calls = append(g.calls, line)
	if g.err != nil {
		return "", g.err
	}
	switch {
	case strings.HasPrefix(line, "issue list"):
		if g.labeled == "" {
			return "[]", nil
		}
		return g.labeled, nil
	case strings.HasPrefix(line, "issue view "):
		return `{"state":"` + g.viewState[args[2]] + `"}`, nil
	}
	return "", nil
}

func runGH(g *fakeGH, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(string) string { return "" }}
	code = Command(g.run)(args, s)
	return code, o.String(), e.String()
}

// extRetro: one retro outside the kit with two kit-area deltas and one that is not.
const extRetro = "## Proposed kit deltas\n| # | Proposed change | Target | Evidence |\n|---|---|---|---|\n" +
	"| B1 | n4_foo drops a key | tool | seen |\n" +
	"| B2 | the lint misses a case | toolbelt/lint-x.sh | seen |\n" +
	"| B3 | something about the research loop | research-sdd | seen |\n"

// extKit is a kit root with one folded retro, a corpus file and an outside directory holding extRetro as
// 2026-03-01-outsidestem.md plus a retro that names neither area.
func extKit(t *testing.T, corpus string) (root, ext string) {
	root = kit(t, map[string]string{
		"retros/INDEX.md": "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n| 2026-01-01-ownstem.md | kit | 2026-01-01 | folded | 1 |\n",
		"core.md":         "[ev: retro ownstem Δ1]\n" + corpus,
	})
	ext = t.TempDir()
	for name, text := range map[string]string{
		"2026-03-01-outsidestem.md": extRetro,
		"2026-03-02-unrelated.md":   "## Proposed deltas\n| # | Delta | Target |\n|---|---|---|\n| 1 | nothing relevant | somewhere else |\n",
	} {
		if err := os.WriteFile(filepath.Join(ext, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, ext
}

func TestExternalRootReportsUncitedDeltasOfRetrosThatNameTheKit(t *testing.T) {
	root, ext := extKit(t, "")
	g := &fakeGH{}
	code, out, errs := runGH(g, "--strict", "--external-root", ext, "--alias", "outsidestem=ALPHA", root+"/retros/INDEX.md", root)
	want := w + "external ALPHA ΔB1 not cited\n" + w + "external ALPHA ΔB2 not cited\n" +
		"fold-audit: external: 1 retros, 2 deltas checked, 2 not cited\n"
	if code != ExitStrict || errs != "" || !strings.Contains(out, want) {
		t.Errorf("exit %d, stderr %q, out:\n%s\nwant it to contain:\n%s", code, errs, out, want)
	}
	if strings.Contains(out, "outsidestem") || strings.Contains(out, "unrelated") || strings.Contains(out, "ΔB3") || strings.Contains(out, ext) {
		t.Errorf("a stem, a path, an unrelated retro or a research-sdd delta leaked into the output:\n%s", out)
	}
}

func TestExternalDeltaCitedByAliasIsNotReported(t *testing.T) {
	root, ext := extKit(t, "Rule [ev: retro ALPHA ΔB1] and out of repo [deferred: retro ALPHA ΔB2].\n")
	code, out, _ := runGH(&fakeGH{}, "--strict", "--external-root", ext, "--alias", "outsidestem=ALPHA", root+"/retros/INDEX.md", root)
	if code != ExitOK || strings.Contains(out, "external ALPHA") || !strings.Contains(out, "fold-audit: external: 1 retros, 2 deltas checked, 0 not cited") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestExternalDeltaWithoutAliasNeverPrintsTheStem(t *testing.T) {
	root, ext := extKit(t, "")
	_, out, errs := runGH(&fakeGH{}, "--external-root", ext, root+"/retros/INDEX.md", root)
	if strings.Contains(out+errs, "outsidestem") {
		t.Errorf("a stem with no alias was printed:\n%s%s", out, errs)
	}
	if !strings.Contains(out, w+"external root1-retro1 ΔB1 not cited") || !strings.Contains(out, "no --alias for 1 retro(s)") {
		t.Errorf("without an alias the retro is named by its position and the alias is asked for:\n%s", out)
	}
}

func TestExternalClosedIssueMarkerCreditsADelta(t *testing.T) {
	root, ext := extKit(t, "")
	g := &fakeGH{labeled: `[{"number":9,"title":"t","state":"CLOSED","body":"<!-- retro-delta: ALPHA ΔB1 -->","labels":[],"comments":[]},
		{"number":10,"title":"t","state":"OPEN","body":"<!-- retro-delta: ALPHA ΔB2 -->","labels":[],"comments":[]}]`}
	_, out, _ := runGH(g, "--external-root", ext, "--alias", "outsidestem=ALPHA", root+"/retros/INDEX.md", root)
	if strings.Contains(out, "ΔB1 not cited") || !strings.Contains(out, "external ALPHA ΔB2 not cited") || !strings.Contains(out, "2 deltas checked, 1 not cited") {
		t.Errorf("a closed marker credits, an open one does not:\n%s", out)
	}
}

func TestExternalOfflineIsUnverifiedNeverSilent(t *testing.T) {
	root, ext := extKit(t, "")
	g := &fakeGH{err: errors.New("gh: not logged in")}
	code, out, _ := runGH(g, "--strict", "--external-root", ext, "--alias", "outsidestem=ALPHA", root+"/retros/INDEX.md", root)
	if code != ExitStrict || !strings.Contains(out, "fold-audit: UNVERIFIED external ALPHA ΔB1 (closed-issue lookup unavailable: gh: not logged in)") ||
		!strings.Contains(out, "fold-audit: issues: 2 unverified") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestExternalRootsFromTheConfigFile(t *testing.T) {
	root, ext := extKit(t, "")
	conf := "# local only\nroot " + ext + "\nalias outsidestem=BETA\n"
	if err := os.WriteFile(filepath.Join(root, "fold-roots.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runGH(&fakeGH{}, root+"/retros/INDEX.md", root)
	if !strings.Contains(out, w+"external BETA ΔB1 not cited") {
		t.Errorf("the config file's root and alias were not used:\n%s", out)
	}
	if err := os.WriteFile(filepath.Join(root, "fold-roots.conf"), []byte("bogus line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runGH(&fakeGH{}, root+"/retros/INDEX.md", root); code != ExitEnv || !strings.HasPrefix(errs, "fold-audit: fold-roots.conf line 1") {
		t.Errorf("a malformed config must be an environment error: exit %d, stderr %q", code, errs)
	}
}

func TestExternalRootErrors(t *testing.T) {
	root, _ := extKit(t, "")
	code, _, errs := runGH(&fakeGH{}, "--external-root", filepath.Join(t.TempDir(), "none"), root+"/retros/INDEX.md", root)
	if code != ExitEnv || !strings.HasPrefix(errs, "fold-audit: external root #1 not found") || strings.Contains(errs, "none") {
		t.Errorf("exit %d, stderr %q (the path must not be printed)", code, errs)
	}
	if code, _, errs := runGH(&fakeGH{}, "--external-root"); code != ExitEnv || !strings.Contains(errs, "--external-root needs a directory") {
		t.Errorf("exit %d, stderr %q", code, errs)
	}
}

func TestNoExternalRootChangesNothing(t *testing.T) {
	root, _ := extKit(t, "")
	g := &fakeGH{}
	code, out, _ := runGH(g, "--strict", root+"/retros/INDEX.md", root)
	if code != ExitOK || strings.Contains(out, "external") || len(g.calls) != 0 {
		t.Errorf("without a root nothing external runs and gh is not called: exit %d, calls %q, out:\n%s", code, g.calls, out)
	}
}

// internal rows that carry #N: the issue is the durable record of the delta.
func issueKit(t *testing.T, row string) string {
	return kit(t, map[string]string{
		"retros/INDEX.md":              "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n| 2026-10-01-tracked.md | kit | 2026-10-01 | folded | 1 |\n",
		"retros/2026-10-01-tracked.md": "## Proposed kit deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n" + row,
		"core.md":                      "[ev: retro tracked]\n",
	})
}

func TestRowCarryingAClosedIssueCountsAsCited(t *testing.T) {
	root := issueKit(t, "| Δ1 | do the thing #55 | x | y |\n")
	args := []string{"--strict", "--deltas-since", "2026-09-24", root + "/retros/INDEX.md", root}
	closed := &fakeGH{viewState: map[string]string{"55": "CLOSED"}}
	if code, out, _ := runGH(closed, args...); code != ExitOK || strings.Contains(out, "Δ1 not cited") || !strings.Contains(out, "deltas: 1 checked, 0 not cited") {
		t.Errorf("closed #55: exit %d\n%s", code, out)
	}
	open := &fakeGH{viewState: map[string]string{"55": "OPEN"}}
	if code, out, _ := runGH(open, args...); code != ExitStrict || !strings.Contains(out, "2026-10-01-tracked.md Δ1 not cited") {
		t.Errorf("open #55 must stay uncited: exit %d\n%s", code, out)
	}
	offline := &fakeGH{err: errors.New("gh: no network")}
	code, out, _ := runGH(offline, args...)
	if code != ExitStrict || !strings.Contains(out, "2026-10-01-tracked.md Δ1 not cited") ||
		!strings.Contains(out, "fold-audit: UNVERIFIED 2026-10-01-tracked.md Δ1 (issue #55 state unavailable: gh: no network)") || !strings.Contains(out, "fold-audit: issues: 1 unverified") {
		t.Errorf("offline: exit %d\n%s", code, out)
	}
}

func TestRowWithoutAnIssueNeverCallsGH(t *testing.T) {
	root := issueKit(t, "| Δ1 | do the thing | x | y |\n")
	g := &fakeGH{}
	code, out, _ := runGH(g, "--strict", "--deltas-since", "2026-09-24", root+"/retros/INDEX.md", root)
	if code != ExitStrict || len(g.calls) != 0 || !strings.Contains(out, "Δ1 not cited") {
		t.Errorf("exit %d, calls %q\n%s", code, g.calls, out)
	}
}
