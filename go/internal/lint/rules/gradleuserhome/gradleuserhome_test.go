package gradleuserhome

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt (G17, ledger.d/2026-10-23-r8-mirror-userhome.md). GU1 an
// uncommented org.gradle.jvmargs holding -Duser.home= and GU2 an uncommented systemProp.user.home WARN; a plain
// jvmargs, a commented key (a comment ending in a backslash does not continue), a continued value, a nested file and a `key : value` spelling; --strict; the empty
// root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 10) }
