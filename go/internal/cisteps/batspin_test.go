package cisteps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodSHA = "a1a9f7875aa4b6a9480ca384d5865f1ccf1b0b1faead6b47aa47d79709a5c5fd"

// installStep is a workflow whose "Install bats" step runs the given script lines.
func installStep(lines ...string) string {
	return head + "      - name: Install bats-core\n        run: |\n          " + strings.Join(lines, "\n          ") + "\n" +
		"      - name: after\n        run: echo BATS_VER=9.9.9\n"
}

func runBatsPin(t *testing.T, yml string) (int, string, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "w.yml")
	if err := os.WriteFile(p, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runMain("bats-pin", p)
	return code, out, strings.ReplaceAll(errs, p, "w.yml")
}

func TestBatsPinPrintsVersionAndSha(t *testing.T) {
	t.Parallel()
	yml := installStep("BATS_VER=1.10.0", "BATS_SHA256="+goodSHA, "PAR_VER=20240222", "echo \"${BATS_SHA256}  x\"")
	if code, out, errs := runBatsPin(t, yml); code != 0 || out != "1.10.0 "+goodSHA+"\n" || errs != "" {
		t.Errorf("bats-pin = %d, %q, %q", code, out, errs)
	}
}

func TestBatsPinReadsIndentedLinesWithTrailingSpace(t *testing.T) {
	t.Parallel()
	yml := installStep("if true; then", "  BATS_VER=1.10.0  ", "  BATS_SHA256="+goodSHA+" ", "fi")
	if code, out, _ := runBatsPin(t, yml); code != 0 || out != "1.10.0 "+goodSHA+"\n" {
		t.Errorf("bats-pin = %d, %q", code, out)
	}
}

func TestBatsPinWithoutInstallStepIsEmptySuccess(t *testing.T) {
	t.Parallel()
	yml := head + "      - name: bats run\n        run: bats tests\n"
	if code, out, errs := runBatsPin(t, yml); code != 0 || out != "" || errs != "" {
		t.Errorf("bats-pin = %d, %q, %q; want 0, empty, empty", code, out, errs)
	}
}

func TestBatsPinFailsClosed(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, yml, err string }{
		{"missing BATS_VER", installStep("BATS_SHA256=" + goodSHA),
			"ci-steps bats-pin: step \"Install bats-core\" has no BATS_VER=<version> line\n"},
		{"missing BATS_SHA256", installStep("BATS_VER=1.10.0"),
			"ci-steps bats-pin: step \"Install bats-core\" has no BATS_SHA256=<64 hex> line\n"},
		{"duplicated BATS_VER", installStep("BATS_VER=1.10.0", "BATS_SHA256="+goodSHA, "BATS_VER=1.11.0"),
			"ci-steps bats-pin: step \"Install bats-core\" sets BATS_VER= more than once (lines 1 and 3 of the step)\n"},
		{"duplicated BATS_SHA256", installStep("BATS_VER=1.10.0", "BATS_SHA256="+goodSHA, "x=1", "BATS_SHA256="+goodSHA),
			"ci-steps bats-pin: step \"Install bats-core\" sets BATS_SHA256= more than once (lines 2 and 4 of the step)\n"},
		{"malformed sha (63 hex)", installStep("BATS_VER=1.10.0", "BATS_SHA256="+goodSHA[1:]),
			"ci-steps bats-pin: step \"Install bats-core\" has no BATS_SHA256=<64 hex> line\n"},
		{"malformed version", installStep("BATS_VER=v1.10", "BATS_SHA256="+goodSHA),
			"ci-steps bats-pin: step \"Install bats-core\" has no BATS_VER=<version> line\n"},
	}
	for _, c := range cases {
		if code, out, errs := runBatsPin(t, c.yml); code != 3 || out != "" || errs != c.err {
			t.Errorf("%s: bats-pin = %d, %q, %q; want 3, \"\", %q", c.name, code, out, errs, c.err)
		}
	}
}

func TestBatsPinUsageAndUnreadable(t *testing.T) {
	t.Parallel()
	if code, out, errs := runMain("bats-pin"); code != 2 || out != "" || errs != "usage: ci-steps bats-pin <workflow.yml>\n" {
		t.Errorf("no file: %d, %q, %q", code, out, errs)
	}
	missing := filepath.Join(t.TempDir(), "nope.yml")
	if code, out, errs := runMain("bats-pin", missing); code != 2 || out != "" || errs != "[Errno 2] No such file or directory: '"+missing+"'\n" {
		t.Errorf("missing: %d, %q, %q", code, out, errs)
	}
	bad := head + "      - run: >\n          x\n"
	if code, out, _ := runBatsPin(t, bad); code != 2 || out != "" {
		t.Errorf("unparseable workflow: %d, %q; want 2", code, out)
	}
}
