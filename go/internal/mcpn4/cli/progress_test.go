package cli

import "testing"

// Go twin of test_bql.TestInventoryTool.test_server_flag_sets_the_progress_file
// (go_server.go_twin("cli", "injection")); the context gets both paths expanded as server.py
// expands them (os.path.expanduser).
func TestServerFlagSetsTheProgressFile(t *testing.T) {
	t.Setenv("HOME", "/home/op/")
	cfg, _, err := Parse([]string{"--progress-file", "/x/p.jsonl"})
	if err != nil || cfg.ProgressFile != "/x/p.jsonl" {
		t.Fatalf("progress file %q, %v", cfg.ProgressFile, err)
	}
	if ctx := NewContext(cfg); ctx.ProgressPath != "/x/p.jsonl" || ctx.StateDir != "/home/op/.local/state/mcp-n4" {
		t.Errorf("progress %q, state dir %q", ctx.ProgressPath, ctx.StateDir)
	}
	cfg, _, _ = Parse([]string{"--progress-file", "~/p.jsonl", "--state-dir", "~"})
	if ctx := NewContext(cfg); ctx.ProgressPath != "/home/op/p.jsonl" || ctx.StateDir != "/home/op" {
		t.Errorf("progress %q, state dir %q", ctx.ProgressPath, ctx.StateDir)
	}
	if ctx := NewContext(Config{}); ctx.ProgressPath != "" {
		t.Errorf("no flag: progress %q", ctx.ProgressPath)
	}
}
