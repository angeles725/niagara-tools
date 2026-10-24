package golden

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// toy is the bash tool of the recorder tests; toyMain is its Go port.
const toy = `#!/usr/bin/env bash
printf '%s\n' "$1" > made.txt
rm gone.txt
chmod 755 kept.sh
printf 'hello %s' "$1"
printf 'warn %s\n' "$HOME" >&2
exit 3
`

func toyMain(_, args []string, s cli.Stdio) int {
	os.WriteFile("made.txt", []byte(args[0]+"\n"), 0o644)
	os.Remove("gone.txt")
	os.Chmod("kept.sh", 0o755)
	fmt.Fprintf(s.Out, "hello %s", args[0])
	fmt.Fprintf(s.Err, "warn %s\n", s.Env("HOME"))
	return 3
}

// recordToy writes the toy script (with body) and a one-case set pinned to pin ("" = the script's sha256),
// runs record.sh into a fresh out dir and returns that dir, the output and the error.
func recordToy(t *testing.T, body, pin string) (string, string, error) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	work := t.TempDir()
	script, cases, out := filepath.Join(work, "toy.sh"), filepath.Join(work, "cases"), filepath.Join(work, "out")
	if pin == "" {
		pin = fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	}
	for p, s := range map[string]string{
		script: body, filepath.Join(cases, "record.env"): "SOURCE=toy.sh\nCOMMIT=abc\nSHA256=" + pin + "\nCHDIR=1\n",
		filepath.Join(cases, "c.txt"): "-- args --\n@DIR@/x\n-- want --\n-- stderr --\n-- gone.txt --\nbye\n-- kept.sh --\n:\n",
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := exec.Command("bash", "record/record.sh", script, cases, out).CombinedOutput()
	return out, string(b), err
}

// TestRecordReplays records the toy and replays the recording through its Go port: output without a final
// newline, the default HOME, a written, a removed and a re-moded file all round-trip.
func TestRecordReplays(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("record.sh refuses root")
	}
	out, msg, err := recordToy(t, toy, "")
	if err != nil {
		t.Fatalf("record.sh: %v\n%s", err, msg)
	}
	b, _ := os.ReadFile(filepath.Join(out, "c.txt"))
	for _, want := range []string{"-- want (noeol) --\nexit 3\nhello @DIR@/x\n", "-- stderr --\nwarn @DIR@\n",
		"-- @after kept.sh (mode 755) --\n:\n", "-- @after made.txt --\n@DIR@/x\n", "-- @gone gone.txt --\n", "\ntools: GNU bash"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("recording lacks %q:\n%s", want, b)
		}
	}
	t.Run("replay", func(t *testing.T) { Runner{Dir: out, Cases: 1, Main: toyMain}.Run(t) })
}

// TestRecordRefuses pins the refusals: a script whose sha256 is not the pinned one, a script that runs a Go
// tool (the recording would compare the port with itself).
func TestRecordRefuses(t *testing.T) {
	for _, c := range []struct{ name, body, pin, msg string }{
		{"checksum", toy, strings.Repeat("0", 64), "pinned " + strings.Repeat("0", 64)},
		{"go tool", toy + "BIN=\"$(go_tool_bin go x)\"\n", "", "runs a Go tool"},
	} {
		if _, msg, err := recordToy(t, c.body, c.pin); err == nil || !strings.Contains(msg, c.msg) {
			t.Errorf("%s: err %v, output %q, want a refusal naming %q", c.name, err, msg, c.msg)
		}
	}
}
