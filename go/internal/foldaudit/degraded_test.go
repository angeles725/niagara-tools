package foldaudit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const degradedLine = "fold-audit: DEGRADED external-roots-unconfigured: niagara-research/retros has 1 retro(s) with 2 pending kit delta(s); " +
	"not audited — pass --external-root <dir> or add `root <dir>` to fold-roots.conf\n"

// runHome is runGH with HOME set, so $HOME/niagara-research/retros is found.
func runHome(home string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}}
	code = Command((&fakeGH{}).run)(args, s)
	return code, o.String(), e.String()
}

// researchHome is a home dir holding niagara-research/retros with extRetro (two kit deltas) and an unrelated retro.
func researchHome(t *testing.T) string {
	home := t.TempDir()
	dir := filepath.Join(home, "niagara-research", "retros")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2026-03-01-outsidestem.md"), []byte(extRetro), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestUnconfiguredExternalRootsWithPendingResearchRetrosIsDegradedNotSilent(t *testing.T) {
	root, _ := extKit(t, "")
	home := researchHome(t)
	code, out, errs := runHome(home, "--strict", root+"/retros/INDEX.md", root)
	if code != ExitStrict || errs != "" || !strings.Contains(out, degradedLine) {
		t.Errorf("exit %d, stderr %q, want %d and the line\n%swithin:\n%s", code, errs, ExitStrict, degradedLine, out)
	}
	if strings.Contains(out, "outsidestem") || strings.Contains(out, home) {
		t.Errorf("a stem or a path leaked:\n%s", out)
	}
}

func TestDegradedExternalStateIsAbsentWhenARootIsConfiguredOrNothingPends(t *testing.T) {
	root, ext := extKit(t, "")
	home := researchHome(t)
	_, out, _ := runHome(home, "--external-root", ext, root+"/retros/INDEX.md", root)
	if strings.Contains(out, "DEGRADED") {
		t.Errorf("a configured root is audited, not degraded:\n%s", out)
	}
	if code, out, _ := runHome(t.TempDir(), "--strict", root+"/retros/INDEX.md", root); code != ExitOK || strings.Contains(out, "DEGRADED") {
		t.Errorf("no research retros: exit %d, out:\n%s", code, out)
	}
}

func TestNoResearchProbeSkipsOnlyTheDegradedProbe(t *testing.T) {
	root, _ := extKit(t, "")
	home := researchHome(t)
	idx := root + "/retros/INDEX.md"
	code, out, errs := runHome(home, "--strict", "--no-research-probe", idx, root)
	if code != ExitOK || errs != "" || strings.Contains(out, "DEGRADED") {
		t.Errorf("with the option: exit %d, stderr %q, want %d and no DEGRADED line; out:\n%s", code, errs, ExitOK, out)
	}
	code, out, _ = runHome(home, "--strict", idx, root)
	if code != ExitStrict || !strings.Contains(out, degradedLine) {
		t.Errorf("without the option the line stays: exit %d, out:\n%s", code, out)
	}
}

func TestNoResearchProbeKeepsTheAuditOfAConfiguredRoot(t *testing.T) {
	root, ext := extKit(t, "")
	home := researchHome(t)
	idx := root + "/retros/INDEX.md"
	_, withOpt, _ := runHome(home, "--no-research-probe", "--external-root", ext, idx, root)
	_, without, _ := runHome(home, "--external-root", ext, idx, root)
	if withOpt != without {
		t.Errorf("the option changed the report of a configured root:\nwith:\n%s\nwithout:\n%s", withOpt, without)
	}
}

func TestParseArgsNoResearchProbe(t *testing.T) {
	o, msg := parseArgs([]string{"--no-research-probe", "I", "K"})
	if msg != "" || !o.noResearchProbe || o.index != "I" || o.root != "K" {
		t.Errorf("got %+v, %q", o, msg)
	}
	if o, _ := parseArgs([]string{"I", "K"}); o.noResearchProbe {
		t.Error("the probe is skipped by default")
	}
}
