// Package writepath is lint-write-path (toolbelt/lint-write-path.sh): every @NiagaraProperty with
// Flags.OPERATOR (or "o") under a module root needs a row in docs/write-path-matrix.md (FAIL), every
// matrix row names a source slot (STALE) or, marked [concept], none (DRIFT). It is a lint.OptionRule: it
// reads its module root itself, walks up to the matrix and, with --matrix, reads the given one instead.
package writepath

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

const id = "lint-write-path"

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Fail }

func (r rule) CheckRoot(root string) ([]lint.Finding, error) { return r.CheckRootOptions(root, nil) }

// CheckRootOptions prints the bash lint's rows in its order: FAIL per profile, then STALE/DRIFT per
// matrix line. Its env errors are stdout rows (lint.RowError).
func (rule) CheckRootOptions(root string, values map[string]string) ([]lint.Finding, error) {
	s := &scan{root: root, files: map[string][]string{}}
	m, err := s.matrix(values["--matrix"])
	if err != nil {
		return nil, err
	}
	matrixJava, err := s.list(lint.Dirname(lint.Dirname(m.path)), "build")
	if err != nil {
		return nil, err
	}
	covered, err := s.names(matrixJava)
	if err != nil {
		return nil, err
	}
	profiles, srcs := s.targets()
	if len(srcs) == 0 {
		return nil, s.errorRow("no src found")
	}
	rows := m.slots()
	var out []lint.Finding
	for i, src := range srcs {
		files, err := s.list(src, "")
		if err != nil {
			return out, err
		}
		slots, err := s.operatorSlots(files)
		if err != nil {
			return out, err
		}
		for _, slot := range slots {
			if !rows[slot] {
				out = append(out, lint.Finding{Path: profiles[i], Msg: "slot " + slot + ": no matrix row"})
			}
		}
	}
	return append(out, m.stale(covered)...), nil
}

// Spec is lint-write-path's contract: <module-root> [--bog <config.bog>] [--matrix <path>] [--strict] in any
// order; FAIL rows "<profile>  slot <name>: no matrix row", STALE/DRIFT "<matrix>:<line>  slot <name>: ...";
// exit 1 on a FAIL (or a STALE/DRIFT under --strict), 3 on a usage or env error.
func Spec() lint.Spec {
	return lint.Spec{Root: rule{}, Mode: lint.Options, ValueFlags: []string{"--bog", "--matrix"}, BarePathRow: true,
		Usage: "usage: lint-write-path.sh <module-root> [--bog <config.bog>] [--matrix <path>] [--strict]"}
}

// scan is one run over a module root; files caches each Java file's lines, read once (the matrix-root
// walk and a profile walk list the same files).
type scan struct {
	root  string
	files map[string][]string
}

func (s *scan) errorRow(reason string) error {
	return &lint.RowError{Row: id + "  ERROR  " + s.root + "  " + reason}
}

// list is list_java: every *.java under dir (dot, *-worktrees and prune sub-directories pruned), or the
// ERROR row of a walk that could not read a directory (fail closed: a skipped file hides its slots).
func (s *scan) list(dir, prune string) ([]string, error) {
	sel := lint.Selection{Names: []string{"*.java"}}
	if prune != "" {
		sel.Prune = []string{prune}
	}
	files, err := lint.Walk(dir, sel)
	if err != nil {
		return nil, s.errorRow(fmt.Sprintf("cannot list every file under %s: %s", dir, err))
	}
	return files, nil
}

// lines reads p once. An unreadable file fails closed (the bash awk and grep skipped it in silence).
func (s *scan) lines(p string) ([]string, error) {
	if ls, ok := s.files[p]; ok {
		return ls, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("cannot scan %s: %s", p, strerror.Text(err))
	}
	ls, _ := javasrc.Lines(strings.NewReader(string(data))) // a strings.Reader never fails
	s.files[p] = ls
	return ls, nil
}

// targets are the profiles scanned: <root>/src alone, named after the root, or else every *-rt/-ux/-wb/-se
// sub-directory holding a src (symlinks below the root not followed, as find -type d), in byte order.
func (s *scan) targets() (names, srcs []string) {
	if lint.IsDir(s.root + "/src") {
		return []string{lint.Basename(s.root)}, []string{s.root + "/src"}
	}
	entries, _ := os.ReadDir(s.root) // an unreadable root lists no profile, as find 2>/dev/null did
	prefix := strings.TrimSuffix(s.root, "/") + "/"
	for _, e := range entries {
		n := e.Name()
		if e.Type().IsDir() && profile(n) && lint.IsDir(prefix+n+"/src") {
			names, srcs = append(names, n), append(srcs, prefix+n+"/src")
		}
	}
	return names, srcs
}

func profile(name string) bool {
	for _, sfx := range []string{"-rt", "-ux", "-wb", "-se"} {
		if strings.HasSuffix(name, sfx) {
			return true
		}
	}
	return false
}

// sorted is sort -u of a set.
func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
