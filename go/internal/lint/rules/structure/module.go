package structure

import (
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

var (
	hostPath     = regexp.MustCompile(`^(niagara_home|niagara_user_home|user_home|nodeHome)=[A-Za-z]:[\\/]`)
	jdkPaths     = regexp.MustCompile(`^[[:space:]]*org\.gradle\.java\.installations\.paths[[:space:]]*=`)
	jdkAuto      = regexp.MustCompile(`^[[:space:]]*org\.gradle\.java\.installations\.auto-detect[[:space:]]*=`)
	pluginLit    = regexp.MustCompile(`^[[:space:]]*val[[:space:]]+gradlePluginVersion[[:space:]]*(:[[:space:]]*String[[:space:]]*)?=[[:space:]]*"`)
	signingApply = regexp.MustCompile(`id[[:space:]]*\([[:space:]]*"com\.tridium\.niagara-signing"`)
)

// signBlock is the signing block L15 requires, each part named as its row names it. The patterns are
// preflight.sh SAMPLE_SIGN_ERE, verbatim (tests/lint-structure.bats LS15-sync keeps them equal).
var signBlock = []struct {
	name string
	re   *regexp.Regexp
}{
	{"allowDefaultProfile.set(false) (or = false)", regexp.MustCompile(`allowDefaultProfile[[:space:]]*(\.set\([[:space:]]*false[[:space:]]*\)|=[[:space:]]*false([^[:alnum:]_]|$))`)},
	{"niagaraSigning {", regexp.MustCompile(`niagaraSigning[[:space:]]*\{`)},
	{"aliases", regexp.MustCompile(`(^|[^[:alnum:]_])aliases([^[:alnum:]_]|$)`)},
	{"signingProfileFile", regexp.MustCompile(`signingProfileFile`)},
}

// hostPaths is L10: every gradle.properties under the root (dot directories pruned), then the one in
// each parent directory up to the first holding a .git directory, may not pin an absolute host path.
func (s *scan) hostPaths() {
	for _, p := range s.files(s.root, true, "gradle.properties") {
		s.hostPath(p)
	}
	for dir := s.root; ; {
		up := dirname(dir)
		if up == dir {
			break
		}
		dir = up
		s.hostPath(dir + "/gradle.properties")
		if isDir(dir + "/.git") {
			break
		}
	}
}

func (s *scan) hostPath(p string) {
	if isFile(p) && grep(s.lines(p), hostPath) {
		s.row(lint.Unset, p, "L10: absolute host path in tracked gradle.properties")
	}
}

// jdkPin is L12: the module root's gradle.properties declares both JDK installation keys, uncommented.
func (s *scan) jdkPin() {
	p := s.root + "/gradle.properties"
	if !isFile(p) {
		return
	}
	if lines := s.lines(p); !grep(lines, jdkPaths) || !grep(lines, jdkAuto) {
		s.row(lint.Unset, s.rel(p), "L12: absent/commented org.gradle.java.installations block (add paths + auto-detect=false to pin JDK 8)")
	}
}

// siblingHomes is L13: the first niagara_home= of each <group>/*/gradle.properties (the group being the
// root's parent) must agree; every sibling that sets one gets a row when two or more differ.
func (s *scan) siblingHomes() {
	group := dirname(s.root)
	if group == s.root || !isDir(group) {
		return
	}
	entries, _ := os.ReadDir(group) // an unreadable group lists nothing, as the bash glob did
	type home struct{ val, path string }
	var homes []home
	distinct := map[string]bool{}
	for _, e := range entries {
		p := group + "/" + e.Name() + "/gradle.properties"
		if strings.HasPrefix(e.Name(), ".") || !isFile(p) {
			continue
		}
		for _, ln := range s.lines(p) {
			if val, ok := strings.CutPrefix(ln, "niagara_home="); ok {
				if val != "" {
					homes = append(homes, home{val, p})
					first, _, _ := strings.Cut(val, "\t") // awk -F'\t' '{print $1}'
					distinct[first] = true
				}
				break
			}
		}
	}
	if len(distinct) < 2 || len(homes) < 2 {
		return
	}
	for _, h := range homes {
		// IFS=<tab> read -r v p: leading tabs dropped, v ends at the next tab, p is the rest.
		line := strings.Trim(h.val+"\t"+h.path, "\t")
		v, p, _ := strings.Cut(line, "\t")
		p = strings.TrimLeft(p, "\t")
		s.row(lint.Unset, strings.TrimPrefix(p, group+"/"), "L13: divergent niagara_home across sibling gradle.properties (value: "+v+")")
	}
}

// pluginPin is L14 and L15 over the gradle root: the root's own settings.gradle.kts, else the nearest
// ancestor's (the walk stops at a directory holding .git). L14: it may not pin gradlePluginVersion to a
// literal. L15: the build.gradle.kts beside it, when it applies the signing plugin, carries the block.
func (s *scan) pluginPin() {
	settings := ""
	for dir := s.root; ; {
		if p := dir + "/settings.gradle.kts"; isFile(p) {
			settings = p
			break
		}
		if isDir(dir + "/.git") {
			break
		}
		up := dirname(dir)
		if up == dir {
			break
		}
		dir = up
	}
	if settings == "" {
		return
	}
	if grep(s.lines(settings), pluginLit) {
		s.row(lint.Unset, settings, `L14: hardcoded gradlePluginVersion literal (use providers.gradleProperty("niagaraPluginVersion").getOrElse(...) so -PniagaraPluginVersion applies)`)
	}
	if build := strings.TrimSuffix(settings, "/settings.gradle.kts") + "/build.gradle.kts"; isFile(build) {
		s.signing(build)
	}
}

// signing is L15 on one root build file: only uncommented code counts, and the block's names are
// matched with string contents blanked; the plugin id, itself a string, before blanking.
func (s *scan) signing(build string) {
	data, err := os.ReadFile(build)
	if err != nil { // a visible row, never a silent skip
		s.row(lint.Warn, build, "L15: could not scan the root for the signing block (awk exit 2: awk: fatal: cannot open file `"+
			build+"' for reading: "+strerror.Text(err)+") — signing block unchecked")
		return
	}
	lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
	stripped, _ := javasrc.Strip(lines)
	if !grep(stripped, signingApply) {
		return
	}
	code, _ := javasrc.BlankLiterals(stripped, nil)
	var miss []string
	for _, part := range signBlock {
		if !grep(code, part.re) {
			miss = append(miss, part.name)
		}
	}
	if len(miss) > 0 {
		s.row(lint.Warn, build, "L15: applies com.tridium.niagara-signing without (uncommented) "+strings.Join(miss, ", ")+
			" — add signingServices { signingProfileFactory { allowDefaultProfile.set(false) } } + niagaraSigning { aliases; signingProfileFile } (templates/signing-root.gradle.kts)")
	}
}
