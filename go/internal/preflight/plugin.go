package preflight

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
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
		if findwalk.HasDir(m2, r.plugin) {
			r.row("PASS", "plugin-pin", "plugin "+r.plugin+" (from --plugin-version) found in niagara_home/etc/m2")
		} else {
			advice := " — pick the version this install ships"
			if ships := shippedPlugins(m2); len(ships) > 0 {
				advice = " — this install ships " + strings.Join(ships, ", ") + ": pick one for --plugin-version"
			}
			r.row("FAIL", "plugin-pin", "plugin "+r.plugin+" (from --plugin-version) missing from "+m2+advice)
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
	case findwalk.HasDir(m2, ver):
		r.row("PASS", "plugin-pin", "plugin "+ver+" found in niagara_home/etc/m2")
	default:
		advice := " — copy from Niagara install"
		if ships := shippedPlugins(m2); len(ships) > 0 {
			option := "--plugin-version " + ships[0]
			if len(ships) > 1 {
				option = "--plugin-version <one of them>"
			}
			advice = " — this install ships " + strings.Join(ships, ", ") + ": pass " + option + " (build.sh, preflight.sh, ng-deploy.sh) or pin it in settings.gradle.kts"
		}
		r.row("FAIL", "plugin-pin", "plugin "+ver+" missing from "+m2+advice)
	}
}

// pluginVersionDir is the name of a plugin version directory: three numeric components.
var pluginVersionDir = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// shippedPlugins are the plugin versions the installation ships under m2, sorted: each directory named X.Y.Z
// whose parent's name holds "plugin", so the version of an ordinary library is not offered (retro
// build-kit-bench-session Δ7). Empty when m2 holds none, or does not exist.
func shippedPlugins(m2 string) []string {
	seen := map[string]bool{}
	findwalk.Walk(m2, false, findwalk.NoLimit, func(e findwalk.Entry) bool {
		if e.Dir && pluginVersionDir.MatchString(e.Name) && strings.Contains(strings.ToLower(lint.Basename(lint.Dirname(e.Path))), "plugin") {
			seen[e.Name] = true
		}
		return true
	}, nil)
	var out []string
	for v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return versionLess(out[i], out[j]) })
	return out
}

// grepText is what grep reads of b: nothing when b holds a NUL byte (grep reports a binary file on stderr).
func grepText(b []byte) string {
	if bytes.IndexByte(b, 0) >= 0 {
		return ""
	}
	return string(b)
}
