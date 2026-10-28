package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDirnameBasename: expected values from GNU coreutils; the walks up to a .git sentinel (lint-structure,
// lint-write-path) step with Dirname and keep its spelling (a relative root stops at ".").
func TestDirnameBasename(t *testing.T) {
	for in, want := range map[string]string{
		"a/b": "a", "a/b/": "a", "a//b": "a", "a": ".", "a/": ".", ".": ".", "./": ".", "..": ".",
		"/": "/", "//": "/", "/a": "/", "/a/b/c": "/a/b", "a/b//": "a",
	} {
		if got := Dirname(in); got != want {
			t.Errorf("Dirname(%q) = %q, want %q", in, got, want)
		}
	}
	for _, tc := range []struct{ in, dir, base string }{
		{"a/b", "a", "b"}, {"a/b/", "a", "b"}, {"a", ".", "a"}, {".", ".", "."}, {"/", "/", "/"},
		{"/a", "/", "a"}, {"//a//b//", "//a", "b"}, {"a//b", "a", "b"},
	} {
		if d, b := Dirname(tc.in), Basename(tc.in); d != tc.dir || b != tc.base {
			t.Errorf("%q: dirname %q basename %q, want %q and %q", tc.in, d, b, tc.dir, tc.base)
		}
	}
}

func TestIsFileIsDir(t *testing.T) {
	dir := t.TempDir()
	f, link := filepath.Join(dir, "f"), filepath.Join(dir, "l")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if !IsFile(f) || IsFile(dir) || IsFile(dir+"/none") || !IsDir(dir) || !IsDir(link) || IsDir(f) {
		t.Error("test -f / test -d mismatch")
	}
}

// TestMapHeader: the module map's header is the lines before its first table row, outside ``` fences; the first
// modularity: line decides, a trailing # comment is dropped, and legacy needs a non-empty legacy-reason: line.
func TestMapHeader(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		mode       Modularity
		line       int
		problem    string
	}{
		{"strict", "# MODULE-MAP — M\n\nmodularity: strict\n\n| Concern |\n", ModularityStrict, 3, ""},
		{"strict with a trailing comment", "modularity: strict   # strict | legacy\n", ModularityStrict, 1, ""},
		{"legacy with a reason", "modularity: legacy\nlegacy-reason: predates the map\n", ModularityLegacy, 1, ""},
		{"legacy reason before the mode", "legacy-reason: old\nmodularity: legacy\n", ModularityLegacy, 2, ""},
		{"legacy without a reason", "modularity: legacy\n", ModularityNone, 1, "modularity: legacy needs a non-empty legacy-reason: line"},
		{"legacy with an empty reason", "modularity: legacy\nlegacy-reason:   \n", ModularityNone, 1, "modularity: legacy needs a non-empty legacy-reason: line"},
		{"unknown value", "modularity: loose\n", ModularityNone, 1, "modularity: loose is not strict or legacy"},
		{"no header", "# MODULE-MAP\n| Concern |\n", ModularityNone, 0, "no modularity: line before the table (want modularity: strict or modularity: legacy)"},
		{"a header after the table is not read", "| Concern |\nmodularity: strict\n", ModularityNone, 0, "no modularity: line before the table (want modularity: strict or modularity: legacy)"},
		{"a fenced example is not read", "```\nmodularity: strict\n```\nmodularity: legacy\nlegacy-reason: x\n", ModularityLegacy, 4, ""},
		{"the first modularity line wins", "modularity: strict\nmodularity: legacy\n", ModularityStrict, 1, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mode, line, problem := MapHeader(strings.Split(tc.text, "\n"))
			if mode != tc.mode || line != tc.line || problem != tc.problem {
				t.Errorf("MapHeader = %v, %d, %q; want %v, %d, %q", mode, line, problem, tc.mode, tc.line, tc.problem)
			}
		})
	}
}

// TestModuleMode: the walk goes up from a file or directory to the first directory holding MODULE-MAP.md and
// returns its header's mode; it stops at a directory holding .git (a directory or a worktree's .git file),
// after reading a map in that same directory. No map, or a map with an invalid header, is ModularityNone.
func TestModuleMode(t *testing.T) {
	const (
		strict = "modularity: strict\n"
		legacy = "modularity: legacy\nlegacy-reason: predates the map\n"
	)
	for _, tc := range []struct {
		name  string
		tree  map[string]string // path -> content; a path ending in / is a directory
		start string
		want  Modularity
	}{
		{"strict map at the module root, deep file", map[string]string{".git/": "", "M/MODULE-MAP.md": strict, "M/M-rt/src/a/A.java": ""}, "M/M-rt/src/a/A.java", ModularityStrict},
		{"legacy map", map[string]string{".git/": "", "M/MODULE-MAP.md": legacy, "M/M-rt/src/A.java": ""}, "M/M-rt/src/A.java", ModularityLegacy},
		{"start is a directory", map[string]string{".git/": "", "M/MODULE-MAP.md": strict, "M/M-rt/src/": ""}, "M/M-rt/src", ModularityStrict},
		{"start is the map's own directory", map[string]string{".git/": "", "M/MODULE-MAP.md": legacy}, "M", ModularityLegacy},
		{"no map up to .git", map[string]string{".git/": "", "M/M-rt/src/A.java": ""}, "M/M-rt/src/A.java", ModularityNone},
		{"a map above the .git boundary is not read", map[string]string{"MODULE-MAP.md": strict, "repo/.git/": "", "repo/M/M-rt/src/A.java": ""}, "repo/M/M-rt/src/A.java", ModularityNone},
		{"a worktree .git file stops the walk too", map[string]string{"MODULE-MAP.md": strict, "repo/.git": "gitdir: x\n", "repo/M/A.java": ""}, "repo/M/A.java", ModularityNone},
		{"a map beside .git is read", map[string]string{"repo/.git/": "", "repo/MODULE-MAP.md": strict, "repo/M-rt/src/A.java": ""}, "repo/M-rt/src/A.java", ModularityStrict},
		{"the nearest map wins", map[string]string{".git/": "", "MODULE-MAP.md": strict, "M/MODULE-MAP.md": legacy, "M/M-rt/A.java": ""}, "M/M-rt/A.java", ModularityLegacy},
		{"an invalid header is none", map[string]string{".git/": "", "M/MODULE-MAP.md": "modularity: legacy\n", "M/A.java": ""}, "M/A.java", ModularityNone},
		{"a directory named MODULE-MAP.md is not a map", map[string]string{".git/": "", "M/MODULE-MAP.md/": "", "M/A.java": ""}, "M/A.java", ModularityNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for p, body := range tc.tree {
				full := filepath.Join(root, p)
				if strings.HasSuffix(p, "/") {
					if err := os.MkdirAll(full, 0o755); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := ModuleMode(filepath.Join(root, tc.start)); got != tc.want {
				t.Errorf("ModuleMode(%s) = %v, want %v", tc.start, got, tc.want)
			}
		})
	}
}

func TestModularityString(t *testing.T) {
	for m, want := range map[Modularity]string{ModularityNone: "none", ModularityLegacy: "legacy", ModularityStrict: "strict"} {
		if got := m.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", m, got, want)
		}
	}
}
