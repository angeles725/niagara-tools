// Package wsplacement is the lint-ws-placement rule (stub: the port lands in the next commit).
package wsplacement

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the rule's lint name.
const ID = "lint-ws-placement"

// Spec registers the rule.
func Spec() lint.Spec {
	return lint.Spec{Rule: rule{}, Mode: lint.Each, Usage: "usage: lint-ws-placement.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}}}
}

type rule struct{}

func (rule) ID() string                                  { return ID }
func (rule) Severity() lint.Severity                     { return lint.Warn }
func (rule) Check(*javasrc.File) ([]lint.Finding, error) { return nil, nil }
