package buildmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// signingMirrorBuild is a build over a fresh mirror directory with the given --gradle-prop values; err gets its output.
func signingMirrorBuild(t *testing.T, props ...string) (*build, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".niagara-mirror"), []byte("source=/mnt/c/X\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var err bytes.Buffer
	b := &build{s: cli.Stdio{Err: &err}, home: home}
	for _, p := range props {
		gp, ok := gradleenv.Prop(p)
		if !ok {
			t.Fatalf("gradleenv.Prop(%q) refused", p)
		}
		b.opt.gradleProps = append(b.opt.gradleProps, gp)
	}
	return b, &err
}

// A mirror build without a named profile signs with the mirror home's throwaway key: the warning stays.
func TestAMirrorBuildWithoutAProfileWarnsAboutTheThrowawayKey(t *testing.T) {
	b, err := signingMirrorBuild(t, "niagaraSigningAlias=CodeSigning")
	b.warnMirrorSigning()
	if !strings.Contains(err.String(), "throwaway key of the mirror user home") {
		t.Errorf("stderr %q, want the mirror signing warning", err.String())
	}
}

// --gradle-prop niagaraSigningProfile=<file> signs with that file's key on a mirror build too (seen live,
// issue #1096): the warning would be false, so it is not printed. An empty value names no profile.
func TestAMirrorBuildNamingAProfileDoesNotWarn(t *testing.T) {
	b, err := signingMirrorBuild(t, "niagaraSigningAlias=CodeSigning", "niagaraSigningProfile=/tmp/p/profile.xml")
	b.warnMirrorSigning()
	if err.Len() != 0 {
		t.Errorf("stderr %q, want nothing", err.String())
	}
	b, err = signingMirrorBuild(t, "niagaraSigningProfile=")
	b.warnMirrorSigning()
	if !strings.Contains(err.String(), "throwaway key") {
		t.Errorf("empty profile: stderr %q, want the warning", err.String())
	}
}
