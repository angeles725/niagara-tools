package preflight

import (
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/signblock"
)

// sampleSign checks the root build.gradle.kts code (comments removed, strings emptied) for every part of
// the signing block: without allowDefaultProfile off a missing profile signs with the default key.
func (r *run) sampleSign() {
	kts := r.gr + "/build.gradle.kts"
	miss := signblock.Missing(ktsCode(kts))
	if len(miss) == 0 {
		r.row("PASS", "sample-sign", "root build.gradle.kts has allowDefaultProfile.set(false) + niagaraSigning { aliases; signingProfileFile }")
	} else {
		r.row("FAIL", "sample-sign", "root "+kts+" lacks (uncommented) "+strings.Join(miss, ", ")+" — the signing block is signingServices { signingProfileFactory { allowDefaultProfile.set(false) } } + niagaraSigning { aliases; signingProfileFile }")
	}
}

// sampleBuild warns about build/ output shipped with the sample (stale jars the first build would mix in).
func (r *run) sampleBuild() {
	if b := shippedBuildDirs(r.gr); b != "" {
		r.row("WARN", "sample-build", "shipped build/ output (delete it before the first build): "+b)
	} else {
		r.row("PASS", "sample-build", "no shipped build/ directory")
	}
}

// ktsCode is the Kotlin script file with // and /* */ comments removed (not nested: the first */
// closes) and each "…" string emptied to "" and """…""" raw string to """""", line by line: a comment
// marker inside a string is text and a quoted setting is not code. An unreadable file has no code.
func ktsCode(file string) []string {
	b, err := os.ReadFile(file)
	if err != nil || len(b) == 0 {
		return nil
	}
	var out []string
	inBlock, inRaw := false, false
	for _, ln := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		var o strings.Builder
		inStr := false
	scan:
		for i := 0; i < len(ln); {
			raw := strings.HasPrefix(ln[i:], `"""`)
			switch {
			case inBlock:
				if strings.HasPrefix(ln[i:], "*/") {
					inBlock, i = false, i+2
				} else {
					i++
				}
			case inRaw:
				if raw {
					inRaw, i = false, i+3
					o.WriteString(`"""`)
				} else {
					i++
				}
			case inStr:
				if ln[i] == '\\' {
					i += 2
					continue
				}
				if ln[i] == '"' {
					inStr = false
					o.WriteByte('"')
				}
				i++
			case raw:
				inRaw, i = true, i+3
				o.WriteString(`"""`)
			case ln[i] == '"':
				inStr, i = true, i+1
				o.WriteByte('"')
			case strings.HasPrefix(ln[i:], "//"):
				break scan
			case strings.HasPrefix(ln[i:], "/*"):
				inBlock, i = true, i+2
			default:
				o.WriteByte(ln[i])
				i++
			}
		}
		out = append(out, o.String())
	}
	return out
}

// shippedBuildDirs is the build/ directories at depth 1-3 under the gradle root (followed when a
// symlink, find -H issue #329; dot directories pruned, a build/ not entered), sorted, space-joined.
func shippedBuildDirs(gr string) string {
	var dirs []string
	findWalk(gr, true, 3, func(e entry) bool {
		switch {
		case e.depth == 0:
			return true
		case !e.dir:
			return false
		case strings.HasPrefix(e.name, "."):
			return false
		case e.name == "build":
			dirs = append(dirs, e.path)
			return false
		}
		return true
	}, nil)
	sort.Strings(dirs)
	return strings.TrimRight(strings.Join(dirs, " "), " ")
}
