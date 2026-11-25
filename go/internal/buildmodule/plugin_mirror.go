package buildmodule

import (
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// pluginVersionDir is the name of a plugin version directory: three numeric components.
var pluginVersionDir = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// pickMirrorPlugin settles the plugin version before preflight, whose plugin-pin check would fail the build on
// a pin the niagara_home does not ship. Each install ships ONE niagara-module plugin, and the module's own pin
// (gradle.properties, else the settings.gradle.kts default) may name another. Without an override and with a
// pin that etc/m2 lacks: the one plugin version it ships is used (a NOTICE names both, and it becomes the
// override preflight and gradle get); several or none is exit 10 naming the pin and what was found. An
// override, no pin, or a pin that is there change nothing. [ev: retro client-e-dashboard-live-check Δ29]
func (b *build) pickMirrorPlugin() int {
	if b.plugin != "" {
		return proceed
	}
	pin := b.quietPin()
	repo := b.home + "/" + m2Repository
	if pin == "" || findwalk.HasDirWithin(repo, pin, m2Depth) {
		return proceed
	}
	ships := shippedPlugins(repo)
	if len(ships) != 1 {
		found := "no plugin version"
		if len(ships) > 1 {
			found = strings.Join(ships, ", ")
		}
		b.warn(msgPluginAbsent, pin, b.home, found)
		return ExitEnv
	}
	b.warn(msgPluginPicked, pin, b.home, ships[0])
	b.plugin = ships[0]
	return proceed
}

// quietPin is the module's pinned plugin version without the grep rows pinnedPlugin prints: configureGradle
// reads the same files again and says them once.
func (b *build) quietPin() string {
	loud := b.s.Err
	b.s.Err = io.Discard
	defer func() { b.s.Err = loud }()
	return b.pinnedPlugin()
}

// shippedPlugins are the plugin versions under repo, in version order: each directory named X.Y.Z whose
// parent's name holds "plugin", so the version of an ordinary library is not counted.
func shippedPlugins(repo string) []string {
	seen := map[string]bool{}
	findwalk.Walk(repo, false, m2Depth, func(e findwalk.Entry) bool {
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

// versionLess orders X.Y.Z strings numerically per component.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range pa {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return false
}
