package readtools

import (
	"fmt"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// compareRuntime is every difference between the loaded parts and the jars of scan, in loaded
// order then jar order: a loaded part whose jar holds another version or changed after the boot,
// a loaded part with no jar, and a jar of no loaded part that changed after the boot. A part whose
// jar may be among the unread ones is no difference: it is unknown.
func compareRuntime(parts []loadedPart, scan diskScan, boot time.Time, bootKnown bool) []*pyjson.Object {
	byKey, loaded, unknown := map[string]diskJar{}, map[string]bool{}, map[string]bool{}
	for _, j := range scan.jars {
		if j.read {
			byKey[j.key] = j
		}
	}
	for _, p := range parts {
		loaded[p.name] = true
	}
	for _, name := range scan.unreadable {
		unknown[strings.TrimSuffix(name, runtimeJarSuffix)] = true
	}
	var out []*pyjson.Object
	for _, p := range parts {
		j, onDisk := byKey[p.name]
		switch {
		case !onDisk && !scan.cut && !mayBeUnknown(p.name, unknown):
			out = append(out, mismatchEntry(p.name, reasonLoadedNotOnDisk, p.version, nil))
		case !onDisk:
		case j.version != p.version:
			out = append(out, mismatchEntry(p.name, reasonVersionDiffers, p.version, &j))
		case bootKnown && afterBoot(j.modified, boot):
			out = append(out, mismatchEntry(p.name, reasonModifiedAfter, p.version, &j))
		}
	}
	for _, j := range scan.jars {
		if j.read && !loaded[j.key] && bootKnown && afterBoot(j.modified, boot) {
			out = append(out, mismatchEntry(j.key, reasonNewAfterBoot, nil, &j))
		}
	}
	return out
}

// mayBeUnknown reports whether a part's jar may be one of the unreadable jars (by file name).
func mayBeUnknown(part string, unknown map[string]bool) bool {
	return unknown[part] || mayHoldReverse(part, unknown)
}

// mayHoldReverse reports whether part is a profile part of a module jar named in names.
func mayHoldReverse(part string, names map[string]bool) bool {
	for _, p := range runtimeProfiles {
		if stem, ok := strings.CutSuffix(part, "-"+p); ok && names[stem] {
			return true
		}
	}
	return false
}

// mismatchEntry is one difference; j is nil for a part with no jar.
func mismatchEntry(module, reason string, loadedVersion any, j *diskJar) *pyjson.Object {
	var jar, version, modified any
	if j != nil {
		jar, version, modified = j.file, j.version, j.modified.Format(runtimeJarTimeLayout)
	}
	return obj("module", module, "reason", reason, "loaded_version", loadedVersion, "disk_version", version,
		"jar", jar, "jar_modified", modified)
}

// restartVerdict is restart_needed and its note: true with any difference, false only when
// every input is known and complete, null (with why) otherwise.
func restartVerdict(mismatches int, why []string) (any, string) {
	switch {
	case mismatches > 0:
		return true, fmt.Sprintf(runtimeRestartYes, mismatches)
	case len(why) > 0:
		return nil, fmt.Sprintf(runtimeRestartUnknown, strings.Join(why, runtimeUnknownSeparate))
	}
	return false, runtimeRestartNo
}
