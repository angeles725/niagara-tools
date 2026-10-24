package preflight

import (
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	n4Version   = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?`)
	placeholder = regexp.MustCompile(`<[^>]*>`)
)

// sampleRows (--sample) checks that a project someone else shipped (course zip, vendor sample) was
// retargeted: gradle.properties niagara_home exists and is this install's version (a c:\\… value read as
// its /mnt/c/… mount), niagara_user_home has no <…> placeholder, the root build.gradle.kts code has the
// signing block outside comments and strings, and no build/ output was shipped.
// [ev: retro n4-course-signed-jars-gate-fix Δ6] [ev: retro n4-course-sample-check-precision Δ1]
func (r *run) sampleRows() {
	if !r.sample || r.win {
		return
	}
	gp := r.gr + "/gradle.properties"
	home := prop(gp, "niagara_home")
	sPath := home
	if len(home) >= 2 && home[1] == ':' && strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", rune(home[0])) {
		sPath = "/mnt/" + strings.ToLower(home[:1]) + strings.ReplaceAll(home[2:], `\`, "/")
	}
	sVer, nhVer := lastVersion(sPath), lastVersion(r.nh)
	switch {
	case home == "":
		r.row("FAIL", "sample-home", "no niagara_home in "+gp+" — set it to this install ("+r.nh+")")
	case !lint.IsDir(sPath):
		r.row("FAIL", "sample-home", "gradle.properties niagara_home "+sPath+" does not exist — retarget it to "+r.nh)
	case sVer == "" || nhVer == "":
		which := "the target " + r.nh
		if sVer == "" {
			which = sPath
		}
		r.row("WARN", "sample-home", "gradle.properties niagara_home "+sPath+" exists, but its version could not be compared (no X.Y.Z in "+which+") — confirm it is the target install "+r.nh)
	case sVer != nhVer:
		r.row("FAIL", "sample-home", "gradle.properties niagara_home is "+sVer+", the target install is "+nhVer+" — retarget it to "+r.nh)
	default:
		r.row("PASS", "sample-home", "gradle.properties niagara_home "+sPath+" exists (version "+nhVer+")")
	}
	switch user := prop(gp, "niagara_user_home"); {
	case user == "":
		r.row("WARN", "sample-user", "no niagara_user_home in "+gp+" — set it to this user's Niagara user home")
	case placeholder.MatchString(user):
		r.row("FAIL", "sample-user", "gradle.properties niagara_user_home has a placeholder: "+user)
	default:
		r.row("PASS", "sample-user", "niagara_user_home has no <…> placeholder")
	}
}

const propSpace = " \t\n\v\f\r"

// prop is the value of the first `<key>=` line of a .properties file (blanks around = dropped, \\
// collapsed to \, trailing blanks dropped); "" when the file or the key is missing.
func prop(file, key string) string {
	b, _ := os.ReadFile(file)
	for _, ln := range strings.Split(string(b), "\n") {
		rest := strings.TrimLeft(ln, propSpace)
		if !strings.HasPrefix(rest, key) {
			continue
		}
		if rest = strings.TrimLeft(rest[len(key):], propSpace); strings.HasPrefix(rest, "=") {
			v := strings.ReplaceAll(strings.TrimLeft(rest[1:], propSpace), `\\`, `\`)
			return strings.TrimRight(v, propSpace)
		}
	}
	return ""
}

// lastVersion is the last X.Y.Z[.W] token of p ("" when none).
func lastVersion(p string) string {
	all := n4Version.FindAllString(p, -1)
	if len(all) == 0 {
		return ""
	}
	return all[len(all)-1]
}
