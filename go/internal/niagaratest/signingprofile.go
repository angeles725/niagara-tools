package niagaratest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
)

// signingProfileRefusal returns why the build's signing profile would re-mint its key at every signing, or "" when
// it would not (no profile property, a profile type that never mints a key, or a safe validity/refresh). The rule
// and the plugin facts behind it are go/internal/signingprofile, shared with lint-structure L16. The profile is the
// niagaraSigningProfile gradle property, from GRADLE_USER_HOME/gradle.properties first (gradle's precedence) and
// then the gradle root's; a relative path resolves against the gradle root, as the kit's root signing block does
// (layout.projectDirectory.file). A property passed with -P is not seen: the runner passes none.
func signingProfileRefusal(gradleRoot, gradleUserHome string) string {
	path := ""
	for _, f := range []string{filepath.Join(gradleUserHome, "gradle.properties"), filepath.Join(gradleRoot, "gradle.properties")} {
		if v, ok := gradleProperty(f, signingprofile.Property); ok {
			path = v
			break
		}
	}
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(gradleRoot, path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("signing profile %s (%s): cannot read it: %v", path, signingprofile.Property, err)
	}
	p, err := signingprofile.Parse(b)
	if err != nil {
		return fmt.Sprintf("signing profile %s: %v", path, err)
	}
	if why := p.Refusal(); why != "" {
		return fmt.Sprintf("signing profile %s: %s", path, why)
	}
	return ""
}

// gradleProperty is signingprofile.GradleProperty on one file; a file it cannot read has no property.
func gradleProperty(file, key string) (string, bool) {
	b, err := os.ReadFile(file)
	if err != nil {
		return "", false
	}
	return signingprofile.GradleProperty(b, key)
}
