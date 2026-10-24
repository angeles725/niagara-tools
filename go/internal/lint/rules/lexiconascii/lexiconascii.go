// Package lexiconascii is lint-lexicon-ascii: a non-ASCII byte in a *.lexicon file. Niagara reads lexicons as
// Latin-1 and serves them as UTF-8, so a raw UTF-8 "Presión" arrives as mojibake; accented text must be a
// \uNNNN escape. Port of toolbelt/lint-lexicon-ascii.sh (a bash read loop forking od per line); the wrapper
// keeps the user-facing contract and its header documents the rule and its mutation pins.
//
// It is a per-file lint.Rule over raw bytes: a .lexicon is a java.util.Properties file, but the rule reads no
// key or value (a non-ASCII byte is a defect anywhere, a comment included), so it does not use internal/props.
package lexiconascii

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-lexicon-ascii"

// Spec registers the rule with the argument contract of toolbelt/lint-lexicon-ascii.sh: the first argument is the
// root (no flag is parsed), rows name the file relative to the root, and a root with no *.lexicon is clean (LA5).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:     rule{},
		Mode:     lint.Raw,
		Usage:    "usage: lint-lexicon-ascii.sh <module-root>",
		Select:   lint.Selection{Names: []string{"*.lexicon"}},
		RelPaths: true,
		// Reason: LA5, "no *.lexicon files exits 0": report-module runs the lint on every module root.
		EmptyOK: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Check is the bash read loop: one row per line holding a byte above 0x7f, the bytes in order. Every line is
// read, the last one too when it has no final newline (truth fix 2026-10-24: the bash `while read` skipped it).
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i, ln := range f.Raw {
		var bad []string
		for j := 0; j < len(ln); j++ {
			if ln[j] > 0x7f {
				bad = append(bad, fmt.Sprintf("0x%02x", ln[j]))
			}
		}
		if bad != nil {
			out = append(out, lint.Finding{Line: i + 1, Msg: "non-ASCII: " + strings.Join(bad, " ")})
		}
	}
	return out, nil
}
