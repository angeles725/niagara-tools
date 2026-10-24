// Package statusparity is the lint-status-parity rule (stub: the port lands in the next commit).
package statusparity

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the rule's lint name.
const ID = "lint-status-parity"

// Spec registers the rule.
func Spec() lint.Spec {
	return lint.Spec{Rule: rule{}, Mode: lint.First, Usage: "usage: lint-status-parity.sh [--strict] <java-src-dir>",
		Select: lint.Selection{Names: []string{"*.java"}}}
}

type rule struct{}

func (rule) ID() string                                  { return ID }
func (rule) Severity() lint.Severity                     { return lint.Warn }
func (rule) Check(*javasrc.File) ([]lint.Finding, error) { return nil, nil }
