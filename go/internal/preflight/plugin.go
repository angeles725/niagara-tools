package preflight

import (
	"bytes"
	"os"
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// pluginPin checks that the Niagara gradle plugin version the build uses is under <niagara_home>/etc/m2:
// --plugin-version / $NIAGARA_PLUGIN_VERSION [ev: retro continuous-fan-post-defrost-delay Δ1], else the
// first "X.Y.Z" string of settings.gradle.kts, each as an exact directory name (7.6.2 never passes on
// 7.6.22). The bash matched the settings version anywhere in the path, niagara_home included, so a home
// named n4-7.6.17 passed with an empty etc/m2. A missing pin fails the build with a cryptic repository error.
func (r *run) pluginPin() {
	if r.win {
		return
	}
	m2 := r.nh + "/etc/m2"
	if r.plugin != "" {
		if hasDir(m2, r.plugin) {
			r.row("PASS", "plugin-pin", "plugin "+r.plugin+" (from --plugin-version) found in niagara_home/etc/m2")
		} else {
			r.row("FAIL", "plugin-pin", "plugin "+r.plugin+" (from --plugin-version) missing from "+m2+" — pick the version this install ships")
		}
		return
	}
	settings := r.gr + "/settings.gradle.kts"
	if !lint.IsFile(settings) {
		r.row("FAIL", "plugin-pin", "settings.gradle.kts not found in "+r.gr)
		return
	}
	b, _ := os.ReadFile(settings)
	ver := strings.Trim(quotedVersion.FindString(grepText(b)), `"`)
	switch {
	case ver == "":
		r.row("FAIL", "plugin-pin", "cannot determine plugin version from "+settings)
	case hasDir(m2, ver):
		r.row("PASS", "plugin-pin", "plugin "+ver+" found in niagara_home/etc/m2")
	default:
		r.row("FAIL", "plugin-pin", "plugin "+ver+" missing from "+m2+" — copy from Niagara install")
	}
}

// grepText is what grep reads of b: nothing when b holds a NUL byte (grep reports a binary file on stderr).
func grepText(b []byte) string {
	if bytes.IndexByte(b, 0) >= 0 {
		return ""
	}
	return string(b)
}

// hasDir is `find <root> -type d -name <name> | grep -q .`: a directory under root, root included, whose
// name matches the pattern name.
func hasDir(root, name string) bool {
	found := false
	findWalk(root, false, 1<<30, func(e entry) bool {
		ok, _ := path.Match(name, e.name)
		found = found || (e.dir && ok)
		return !found
	}, nil)
	return found
}
