package scaffoldmodule

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// stringDefault finds a String slot's defaultValue in an @NiagaraProperty line.
var stringDefault = regexp.MustCompile(`type\s*=\s*"String"\s*,\s*defaultValue\s*=\s*"((?:[^"\\]|\\.)*)"`)

// javaExpr is a default Slot-o-Matic can copy as Java code: a dotted name such as BString.DEFAULT.
var javaExpr = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// Slot-o-Matic copied the service-job template's defaultValue = "ModuleA-export.csv" into the generated
// newProperty(...) as code, so every --type service module failed :compileJava (a scaffolded module built against
// a 4.14 mirror, issue #1093). A String default that is not a plain dotted name must be a Java string literal,
// written escaped: defaultValue = "\"text\"".
func TestTemplateStringDefaultsAreJavaStringLiterals(t *testing.T) {
	root := filepath.Join("..", "..", "..", "build-n4-module-kit", "templates")
	seen := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".java") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range stringDefault.FindAllStringSubmatch(line, -1) {
				seen++
				v := m[1]
				if v == "" || javaExpr.MatchString(v) || (strings.HasPrefix(v, `\"`) && strings.HasSuffix(v, `\"`)) {
					continue
				}
				t.Errorf("%s:%d: String defaultValue %q is copied as Java code; write it as \\\"%s\\\"", p, i+1, v, v)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("no String defaultValue found under " + root + " (wrong path?)")
	}
}
