// Package ndrivertaglesscorrelation is lint-ndriver-tagless-correlation (G67): an NDriver module whose messages
// carry no transaction tag while its comm stack keeps more than one request outstanding. NComm matches a
// response to a request by getTag(); with no override every message has the same tag, so when two requests are
// outstanding (the BCommConfig default is 32) a response can complete the wrong request. The fix is a getTag()
// override or getMaxOutstandingTransactions() returning 1 (types/driver-authoring.md § 1.7). The contract is the
// deferred G67 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-u5-c59-c62.md);
// toolbelt/lint-ndriver-tagless-correlation.sh is the wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a -rt part's src/, srcTest/, test/ and build/
// not entered, read as javasrc tokens (comments stripped, a literal one token). Superclass chains are followed
// through the classes of the root. One row per root, on the first NMessage subclass in walk order, when at least
// one exists, none declares getTag(), and no comm-config class declares a getMaxOutstandingTransactions() whose
// body is `return 1;`.
package ndrivertaglesscorrelation

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name>
// convention; the deferred spec's row wrote "ndriver-tagless-correlation").
const ID = "lint-ndriver-tagless-correlation"

// commConfigs are the NDriver comm configuration bases that own getMaxOutstandingTransactions().
var commConfigs = []string{"BCommConfig", "BTcpCommConfig", "BUdpCommConfig", "BSerialCommConfig", "BHttpCommConfig"}

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), WARN rows (exit 1 under
// --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-ndriver-tagless-correlation.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "test", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt part's src/ (wrong path: pass the module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var rtPart = regexp.MustCompile(`(^|/)[^/]+-rt/src/`)

type source struct {
	path    string
	toks    []javasrc.Token
	classes []javasrc.Class
}

// CheckTree gathers the root's classes and methods, then decides the one row.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var srcs []source
	var all []javasrc.Class
	for _, f := range files {
		if rtPart.MatchString(f.Path) {
			toks := javasrc.Tokens(f.Stripped())
			s := source{f.Path, toks, javasrc.Classes(toks)}
			srcs, all = append(srcs, s), append(all, s.classes...)
		}
	}
	if len(srcs) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	messages := javasrc.Descendants(all, "NMessage")
	configs := javasrc.Descendants(all, commConfigs...)
	var first *lint.Finding
	for _, s := range srcs {
		for _, c := range s.classes {
			if messages[c.Name] && first == nil {
				first = &lint.Finding{Path: s.path, Line: c.Line + 1,
					Msg: "no getTag() override and getMaxOutstandingTransactions() is not 1 -- a response can complete the wrong request"}
			}
		}
		for _, d := range javasrc.Decls(s.toks) {
			switch {
			case len(d.Params) > 0 || d.Anonymous:
			case d.Name == "getTag" && messages[d.Class]:
				return nil, nil // the module correlates by tag
			case d.Name == "getMaxOutstandingTransactions" && configs[d.Class] && returnsOne(s.toks[d.Body[0]:d.Body[1]]):
				return nil, nil // one request at a time
			}
		}
	}
	if first == nil {
		return nil, nil
	}
	return []lint.Finding{*first}, nil
}

// returnsOne reports a body that is `return 1;`, parentheses around the 1 allowed.
func returnsOne(body []javasrc.Token) bool {
	var ts []string
	for _, t := range body {
		if t.Text != "(" && t.Text != ")" {
			ts = append(ts, t.Text)
		}
	}
	return len(ts) == 3 && ts[0] == "return" && ts[1] == "1" && ts[2] == ";"
}
