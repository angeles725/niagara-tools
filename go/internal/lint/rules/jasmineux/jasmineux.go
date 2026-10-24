// Package jasmineux is the lint-jasmine-ux rule: a -ux part that ships a browser src/rc/ tree but no
// srcTest/rc/spec/*.js (the Jasmine/Karma test seam) WARNs once, at its root. Advisory: exit 0. Port of
// toolbelt/lint-jasmine-ux.sh (the wrapper keeps the user-facing contract and its header documents the rule
// and its mutation pins). A RootRule: it reads two directories of a module root, not the shared walk.
package jasmineux

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-jasmine-ux"

// Spec registers the rule with the argument contract of toolbelt/lint-jasmine-ux.sh: no flag is parsed and
// only the first argument is read (Raw); the row names the root as given, with no line.
func Spec() lint.Spec {
	return lint.Spec{
		Root:        rule{},
		Mode:        lint.Raw,
		Usage:       "usage: lint-jasmine-ux.sh <ux-module-root>",
		BarePathRow: true,
		// A module without src/rc is clean, exit 0 (tests/lint-empty-root.bats ER5): CheckRoot returns no
		// ErrEmptyRoot for it; EmptyOK records that documented exception.
		EmptyOK: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// CheckRoot WARNs when root/src/rc is a directory and root/srcTest/rc/spec holds no *.js (dot and *-worktrees
// sub-directories pruned, as every walk); a module without src/rc is not checked (no row, exit 0, the
// documented exit-0 exception of lint-empty-root ER5). A spec directory the walk cannot read is an error.
func (rule) CheckRoot(root string) ([]lint.Finding, error) {
	if !isDir(filepath.Join(root, "src", "rc")) {
		return nil, nil
	}
	spec := root + "/srcTest/rc/spec"
	if isDir(spec) {
		files, err := lint.Walk(spec, lint.Selection{Names: []string{"*.js"}})
		if err != nil {
			return nil, errors.New("cannot list every file under " + spec + ": " + err.Error())
		}
		if len(files) > 0 {
			return nil, nil
		}
	}
	return []lint.Finding{{Path: root, Msg: "ux ships src/rc but no srcTest/rc/spec/*.js"}}, nil
}

func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }
