// Package nomd5credential is the lint-no-md5-credential-digest rule: MessageDigest.getInstance("MD5") in a
// file that also holds a credential context (MD5 must not hash credentials). Port of the check in
// toolbelt/lint-no-md5-credential-digest.sh (the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins).
package nomd5credential

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-no-md5-credential-digest"

// Spec registers the rule with the argument contract of toolbelt/lint-no-md5-credential-digest.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-no-md5-credential-digest.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_strip_run (awk_lint_list), which scans the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

// md5 is the call, lower-cased: the line is matched with ASCII case folding (grep -i under LC_ALL=C).
const md5 = `messagedigest.getinstance("md5")`

// credential is every credential-context text, lower-cased (BPassword and BCredentials contain two of them).
var credential = []string{"password", "credential", "pin"}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check, over the code before each line's first "//" folded to ASCII lower case: a file that calls
// MessageDigest.getInstance("MD5") and holds a credential context anywhere reports its first MD5 line.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	at, cred := 0, false
	for i, ln := range f.Raw {
		code := lower(linescan.Cut(ln))
		if at == 0 && strings.Contains(code, md5) {
			at = i + 1
		}
		for _, c := range credential {
			cred = cred || strings.Contains(code, c)
		}
	}
	if at == 0 || !cred {
		return nil, nil
	}
	return []lint.Finding{{Line: at, Msg: linescan.Trimmed(f.Raw[at-1])}}, nil
}

// lower folds ASCII letters only, as grep -i does in the C locale (strings.ToLower would also fold
// non-ASCII letters).
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
