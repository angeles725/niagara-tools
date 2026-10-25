package triageconsole

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// shimDefaults reads the default package, own-tag and module-prefix lists from toolbelt/triage-console.sh, the
// wrapper that hands them to the binary (DefaultPackageEnv, DefaultTagsEnv, DefaultModulePrefixEnv): the
// goldens replay with the wrapper's own values, so a drift between the two fails here.
func shimDefaults(t *testing.T, repo string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, "build-n4-module-kit", "toolbelt", "triage-console.sh"))
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"PKG": DefaultPackageEnv, "OWN_TAGS": DefaultTagsEnv, "MOD_PREFIX": DefaultModulePrefixEnv}
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^(PKG|OWN_TAGS|MOD_PREFIX)="([^"$\\]*)"$`).FindAllSubmatch(b, -1) {
		out[vars[string(m[1])]] = string(m[2])
	}
	if len(out) != len(vars) {
		t.Fatalf("triage-console.sh sets %d of the defaults PKG, OWN_TAGS, MOD_PREFIX, want all 3: %v", len(out), out)
	}
	return out
}

// TestGolden replays every testdata case (recorded from the bash triage-console.sh by golden/record/record.sh,
// or a hand-written truth-* case) through golden.Runner. The bats-* cases read the kit fixtures in place
// (@REPO@/tests/fixtures/...).
func TestGolden(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	defaults := shimDefaults(t, repo)
	golden.Runner{
		Cases: 77,
		Subst: map[string]string{"@REPO@": repo},
		Setup: func(_ *testing.T, _ string, env map[string]string) []string {
			for k, v := range defaults {
				env[k] = v
			}
			return nil
		},
		Main: func(_, args []string, s cli.Stdio) int { return Main(args, s) },
	}.Run(t)
}
