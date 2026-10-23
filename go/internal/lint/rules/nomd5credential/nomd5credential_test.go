package nomd5credential

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-no-md5-credential-digest.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 21) }

// TestUnreadable: an unreadable sub-directory and file keep the awk_lint_list contract (ContinueOnWalkError):
// both env lines, the rows of the files that were read, exit 3.
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  BPassword p; MessageDigest m = MessageDigest.getInstance(\"MD5\");\n}\n")
}
