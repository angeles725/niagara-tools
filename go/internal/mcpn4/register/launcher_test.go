package register

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/envfile"
)

func TestRenderLauncherSourcesTheEnvFileAndExecsTheBinary(t *testing.T) {
	got := renderLauncher("lab", "/h/.config/mcp-n4/lab.env", "/h/.local/bin/mcp-n4",
		[]string{"--station", "LAB=https://127.0.0.1:18443", "--allow-writes", "--load-wait", "10"})
	want := "#!/bin/bash\n" +
		"# mcp-n4 launcher for station lab, written by 'mcp-n4 register'.\n" +
		"# Credentials come only from the env file sourced below (keep it chmod 600); this file holds none.\n" +
		"set -a; . '/h/.config/mcp-n4/lab.env'; set +a\n" +
		"exec '/h/.local/bin/mcp-n4' \\\n" +
		"  '--station' 'LAB=https://127.0.0.1:18443' \\\n" +
		"  '--allow-writes' \\\n" +
		"  '--load-wait' '10'\n"
	if got != want {
		t.Fatalf("launcher =\n%s\nwant\n%s", got, want)
	}
}

func TestSecretInArgsNamesOnlyTheKeysWhoseValueAFlagCarries(t *testing.T) {
	env := []envfile.Assignment{{Key: "P_USER", Value: "API"}, {Key: "P_PASSWORD", Value: "s3cr3t-pw"}, {Key: "P_EMPTY"}}
	args := []string{"--station", "LAB=https://h", "--write-scope", "station:|slot:/s3cr3t-pw", "--x", "API"}
	if got := secretInArgs(args, env); strings.Join(got, ",") != "P_PASSWORD,P_USER" {
		t.Fatalf("secretInArgs = %v, want both keys", got)
	}
	if got := secretInArgs([]string{"--station", "APIX=https://h"}, env); len(got) != 0 {
		t.Fatalf("a short value inside a flag was flagged: %v", got)
	}
}

func TestShellQuoteSurvivesASingleQuote(t *testing.T) {
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("shellQuote = %s", got)
	}
}

func TestFirstDiffLine(t *testing.T) {
	if got := firstDiffLine("a\nb\nc\n", "a\nx\nc\n"); got != 2 {
		t.Fatalf("firstDiffLine = %d, want 2", got)
	}
	if got := firstDiffLine("a\n", "a\nb\n"); got != 2 {
		t.Fatalf("firstDiffLine on a longer text = %d, want 2", got)
	}
}

func TestGitTreeOfFindsADotGitFileOrDirectoryUpTheTree(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "wt")
	if err := os.MkdirAll(filepath.Join(wt, "go", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: /x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := gitTreeOf(filepath.Join(wt, "go", "bin", "mcp-n4")); got != filepath.Join(wt, ".git") {
		t.Fatalf("gitTreeOf = %q, want the worktree .git file", got)
	}
	link := filepath.Join(root, "bin", "mcp-n4")
	_ = os.MkdirAll(filepath.Dir(link), 0o755)
	target := filepath.Join(wt, "go", "bin", "mcp-n4")
	_ = os.WriteFile(target, []byte("x"), 0o755)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if got := gitTreeOf(link); got != filepath.Join(wt, ".git") {
		t.Fatalf("gitTreeOf(symlink) = %q, want the resolved worktree", got)
	}
	if got := gitTreeOf(filepath.Join(root, "plain", "mcp-n4")); got != "" {
		t.Fatalf("gitTreeOf outside a checkout = %q", got)
	}
}
