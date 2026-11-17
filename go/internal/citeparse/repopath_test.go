package citeparse

import (
	"reflect"
	"testing"
)

// TestCardRepoPaths pins retro 2026-11-13-remove-dead-bash-lint-libs Δ1: a backticked repo-relative path under
// toolbelt/, scripts/, go/ or tests/ in prose is a RepoPath pointer, so --strict reports one that was deleted.
func TestCardRepoPaths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []brief
	}{
		{"toolbelt script", "reuse `toolbelt/lib/awk-lint.sh` here", []brief{{1, RepoPath, "toolbelt/lib/awk-lint.sh", ""}}},
		{"the four roots", "`scripts/ci-local.sh`, `go/internal/lint/`, `tests/kit.bats`",
			[]brief{{1, RepoPath, "scripts/ci-local.sh", ""}, {1, RepoPath, "go/internal/lint/", ""}, {1, RepoPath, "tests/kit.bats", ""}}},
		{"other roots are not repo paths", "`lib/scan-files.sh` and `types/logic.md`", nil},
		{"a command line is not a path", "`scripts/ci-local.sh --changed`", nil},
		{"a glob or an ellipsis is not a path", "`toolbelt/lint-*.sh`, `go/...`, `tests/…`", nil},
		{"a path with lines stays a path pointer", "`tests/kit.bats:12`", []brief{{1, Path, "tests/kit.bats", ":12"}}},
		{"a tag is not prose", "[ev: tests/awk.bats AWB1]", []brief{{1, Path, "tests/awk.bats", "AWB1"}}},
		{"a fenced block is skipped", "```\n`toolbelt/x.sh`\n```", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := briefs(Card(tt.text)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// TestEvidenceTagFlag pins Δ2's input: a pointer read from an [ev: …] tag carries Evidence; one from a
// [deferred: …] tag or from prose does not.
func TestEvidenceTagFlag(t *testing.T) {
	t.Parallel()
	ps := Card("[ev: ~/a.md] [deferred: ~/b.md] `tests/c.md:1`")
	if len(ps) != 3 || !ps[0].Evidence || ps[1].Evidence || ps[2].Evidence {
		t.Fatalf("got %+v", ps)
	}
}
