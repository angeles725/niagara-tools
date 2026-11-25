package preflight

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// siblingHome warns when a sibling module root pins another niagara_home than the one this run targets: one
// client repository held three roots pinned to 4.13, 4.14 and 4.15, and a module was compiled against 4.13 while
// the JACE ran 4.15 (F24). A sibling is an immediate directory of the gradle root's parent holding
// settings.gradle.kts or settings.gradle (the markers of a gradle root); its pin is the uncommented
// `niagara_home=` line of its gradle.properties, the one place the kit's templates tell the operator to set it
// (settings.gradle.kts only reads the property or the environment, and the environment is no per-root pin).
// A sibling that is the gradle root through a symlink, without the marker or without a pin is not looked at; a
// C:\\… value is read as its /mnt/c/… mount. The pin and the target are compared as resolved paths (homeKey), so
// a symlinked install or a relative pin naming the same directory is no mismatch; the row prints the pin as written.
// WARN only: the exit code is unchanged. The ledger half (a VERSION-LEDGER.md recording the Niagara version a
// build used) is ledger-niagara, in ledgerniagara.go.
func (r *run) siblingHome() {
	parent := lint.Dirname(r.gr)
	if r.win || !lint.IsDir(parent) || sharedDir(parent) {
		return
	}
	want := homeKey(r.nh, "")
	for _, sib := range globDirs(parent) {
		if sameDir(sib, r.gr) || !isModuleRoot(sib) {
			continue
		}
		pin := prop(sib+"/gradle.properties", "niagara_home")
		if pin == "" || homeKey(pin, sib) == want {
			continue
		}
		r.row("WARN", "sibling-home", sib+" pins niagara_home "+pin+", but the target is "+r.nh+
			" — roots of one client built against different Niagara versions (repin one, or pass -Pniagara_home)")
	}
}

// sameDir says whether a and b are one directory (a symlinked gradle root is its own target, not a sibling).
func sameDir(a, b string) bool {
	ra, ea := filepath.EvalSymlinks(a)
	rb, eb := filepath.EvalSymlinks(b)
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") || (ea == nil && eb == nil && ra == rb)
}

// isModuleRoot says whether dir carries a gradle root's settings file.
func isModuleRoot(dir string) bool {
	return lint.IsFile(dir+"/settings.gradle.kts") || lint.IsFile(dir+"/settings.gradle")
}

// homeKey is a niagara_home value as a comparable path: a drive-letter form read as its /mnt/<d>/… mount, a
// relative value resolved against base (the sibling root a pin was read in; "" for the target, which is relative
// to the working directory), the trailing slash and . or .. segments dropped, and symlinks resolved when the
// path exists (the cleaned string otherwise), so two spellings of one install compare equal.
func homeKey(p, base string) string {
	p = winMount(p)
	if base != "" && !path.IsAbs(p) {
		p = base + "/" + p
	}
	p = path.Clean(p)
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
}

// winMount reads a value starting `<drive>:` (backslashes included) as /mnt/<drive>/…; any other value is
// returned as it is.
func winMount(home string) string {
	if len(home) >= 2 && home[1] == ':' && strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", rune(home[0])) {
		return "/mnt/" + strings.ToLower(home[:1]) + strings.ReplaceAll(home[2:], `\`, "/")
	}
	return home
}
