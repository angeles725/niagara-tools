package signinginventory

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
	"github.com/angeles725/niagara-tools/go/internal/verifymodule"
)

// skipDirs are never walked for built jars.
var skipDirs = map[string]bool{".git": true, "node_modules": true, ".gradle": true}

// module adds what a module records about its signer and what its built jars say, and the FAIL rows when
// the two disagree with the inventory: the recorded expected_signer is in no listed store (or only expired),
// or a built jar carries another signer. An error is an unreadable input (the root, a named --build-state).
func (r *run) module(o options) error {
	root, err := filepath.Abs(o.module)
	if err != nil {
		return err
	}
	if err := exists(root, true); err != nil {
		return err
	}
	name := o.name
	if name == "" {
		name = filepath.Base(root)
	}
	expected, err := r.expected(o.buildState, root, name)
	if err != nil {
		return err
	}
	if expected != "" {
		r.checkRecorded(expected)
	}
	for _, jar := range builtJars(root) {
		r.jar(root, jar, expected)
	}
	return nil
}

// expected is the module's recorded expected_signer ("" when none) and adds its INFO row. The file is
// --build-state (which must be readable), else <root>/BUILD-STATE.md when present, else the kit's
// BUILD-STATE.src.md beside the toolbelt the wrapper names. A malformed record is a FAIL row, not an error.
func (r *run) expected(explicit, root, module string) (string, error) {
	file := explicit
	if file == "" {
		file = filepath.Join(root, "BUILD-STATE.md")
		if _, err := os.Stat(file); err != nil {
			file = ""
			if tb := r.s.Env(ToolbeltEnv); tb != "" {
				file = filepath.Join(tb, "..", "BUILD-STATE.src.md")
			}
		}
	}
	text, err := os.ReadFile(file)
	if err != nil {
		if explicit != "" {
			return "", unreadable(err)
		}
		r.add("INFO", "expected-signer", "%s: no BUILD-STATE file to read", module)
		return "", nil
	}
	alias, err := buildstatefields.ExpectedSigner(string(text), module)
	switch {
	case err != nil:
		r.add("FAIL", "expected-signer", "%s: %v (%s)", module, err, file)
	case alias == "":
		r.add("INFO", "expected-signer", "%s: none recorded in %s", module, file)
	default:
		r.add("INFO", "expected-signer", "%s: %s (recorded in %s)", module, alias, file)
	}
	r.recordTarget(string(text), file, module)
	return alias, nil
}

// checkRecorded is the FAIL row when the recorded alias is in no listed store, or only in expired ones.
func (r *run) checkRecorded(want string) {
	var expired []string
	for _, a := range r.found {
		if !strings.EqualFold(a.Name, want) {
			continue
		}
		if !a.Expired {
			r.add("PASS", "expected-signer-known", "%s is in %s and in date", a.Name, a.Store)
			return
		}
		expired = append(expired, a.Store)
	}
	if len(expired) > 0 {
		r.add("FAIL", "expected-signer-known", "%s is only in expired form (in %s): re-issue it", want, strings.Join(expired, ", "))
		return
	}
	r.add("FAIL", "expected-signer-known", "%s is in none of the %d store(s) listed (%d locked or unreadable)",
		want, r.listed, r.locked)
}

// jar adds the row of one built jar: its signer(s) and the Niagara version it was compiled for, and a FAIL
// row when expected is recorded and the jar is not signed by it.
func (r *run) jar(root, jar, expected string) {
	rel, _ := filepath.Rel(root, jar)
	signers := modjar.Signers(jar)
	by := "unsigned"
	if len(signers) > 0 {
		by = "signed by " + strings.Join(signers, ", ")
	}
	compiled, version := "compiled for no Niagara version", ""
	if v, err := niagaraversion.CompiledFor(jar); err == nil {
		compiled, version = "compiled for Niagara "+v, v
	}
	r.add("INFO", "jar", "%s  %s; %s", rel, by, compiled)
	r.targetRow(rel, version)
	if expected == "" {
		return
	}
	file := verifymodule.SigfileName(expected)
	for _, s := range signers {
		if s == file {
			return
		}
	}
	r.add("FAIL", "jar-signer", "%s: %s, expected %s (META-INF/%s.SF)", rel, by, expected, file)
}

// builtJars is every <root>/**/build/libs/*.jar, in path order.
func builtJars(root string) (jars []string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir():
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
		case strings.HasSuffix(p, ".jar") && filepath.Base(filepath.Dir(p)) == "libs" &&
			filepath.Base(filepath.Dir(filepath.Dir(p))) == "build":
			jars = append(jars, p)
		}
		return nil
	})
	return jars
}
