// Package ntplmanifest is lint-ntpl-manifest-deps (G92): the template archives a module ships. A .ntpl is a zip
// holding template.bog and template-manifest.xml, whose <dependencies> list the modules the template needs
// (types/logic-authoring.md § Shipping a template module); a station template, the kind Workbench lists
// from newStations/, carries isStation="true" on its manifest root (types/distribution.md § 8). For every
// *.ntpl under a module root the lint FAILs an archive without one of the two entries, a manifest that is not
// a well-formed template manifest, a <dependency> naming a module that neither a build file of the module
// declares nor the Niagara installation ships, and a template under newStations/ whose manifest is not a
// station manifest. toolbelt/lint-ntpl-manifest-deps.sh is the wrapper and its header is the user-facing
// contract.
//
// It is a lint.RootRule: it lists the root itself (lint.WalkFollow, which reads through symlinks and fails
// closed) and opens each archive within fixed bounds, never through the engine's whole-file text read. One
// file per concern: messages.go (names, bounds and every text), archive.go (the bounded zip read),
// manifest.go (the XML read), declared.go (the module's declared dependencies), installed.go (the modules
// an installation ships), rows.go (the rows of one archive), links.go (the rows of the links the walk did
// not follow out of the module); this file composes them.
package ntplmanifest

import (
	"errors"
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rowpath"
)

// Spec registers the rule: [--niagara-home <dir>] <module-root>. A root with neither a build file nor a
// template is the ERROR row (exit 3); its root is a module root, so n4lint --all leaves it out.
func Spec() lint.Spec { return specWith(os.Getenv) }

// specWith is Spec reading the environment through getenv.
func specWith(getenv func(string) string) lint.Spec {
	return lint.Spec{Root: rule{getenv: getenv}, Mode: lint.Valued, Options: []string{homeOption}, Usage: usage, EmptyError: lint.EmptyRow{Msg: emptyRoot}}
}

// rule is the lint; option is the --niagara-home value of a run that gave one.
type rule struct {
	option string
	getenv func(string) string
}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Configure keeps the --niagara-home value (the last one given); the installation is read when a root is
// checked. An empty value names no installation and is a usage error: it never stands for "no
// installation", which would turn the ntpl-deps rows into WARN.
func (r rule) Configure(a lint.Args) (lint.Configured, error) {
	if vals := a.Values[homeOption]; len(vals) > 0 {
		r.option = vals[len(vals)-1]
		if r.option == "" {
			// Mutation: NTPL-empty-home -- reading an empty value as "no installation" drops every ntpl-deps row to WARN
			return nil, errors.New(emptyHome)
		}
	}
	return r, nil
}

// CheckRoot checks every *.ntpl under root, in path order. A module that ships none is one PASS row. An
// installation path that gives no module list, a directory the walk cannot enter, a template link that
// leads nowhere, an archive that cannot be opened and a build file that cannot be read are errors (exit 3),
// after the rows of what was read; "nothing to check" is never printed over something that was skipped.
func (r rule) CheckRoot(root string) ([]lint.Finding, error) {
	h := homeOf(r.option, r.getenv)
	stock, err := h.shipped()
	if err != nil {
		return nil, err
	}
	templates, outside, walkErr := lint.WalkFollow(root, lint.Selection{Names: []string{templateNames}, PrunePaths: lint.PartOutputs()})
	mod, err := declaredIn(root)
	if err == nil && walkErr != nil {
		// Mutation: NTPL-walk -- dropping this error prints the rows of the templates that were found and exits as if every directory had been read
		err = fmt.Errorf(partialWalk, rowpath.Display(root), walkErr)
	}
	links := linkRows(outside, mod.outside)
	if len(templates) == 0 {
		switch {
		case err != nil || len(links) > 0:
			// "nothing to check" is never printed over a link that was not followed
			return links, err
		case len(mod.files) == 0:
			return nil, lint.ErrEmptyRoot
		}
		return []lint.Finding{{Severity: lint.Pass, Path: rowpath.Display(root), Msg: noTemplates}}, nil
	}
	c := checker{root: rowpath.Display(root), mod: mod, home: h, stock: stock, out: links}
	for _, path := range templates {
		a, openErr := readArchive(path)
		if openErr != nil {
			if err == nil {
				err = fmt.Errorf(scanFailure, rowpath.Display(path), openErr)
			}
			continue
		}
		c.check(rowpath.Display(path), underStationDir(root, path), a)
	}
	return c.rows(), err
}
