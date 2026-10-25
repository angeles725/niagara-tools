package structure

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/props"
	"github.com/angeles725/niagara-tools/go/internal/signblock"
	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

var (
	// a java.util.Properties line: indent, key, "=" or ":" or blanks; then a drive path ("C\:" escaped
	// too), a POSIX path or a UNC share (truth audit 2026-10-22 T7: the bash regex took only key=X:\). The
	// keys are the shared host build-location keys (props.BuildLocationKeys, R10-D8) plus user_home.
	hostPath = regexp.MustCompile(`^[ \t\f]*(` + hostKeys() + `)([ \t\f]*[=:][ \t\f]*|[ \t\f]+)([A-Za-z]\\?:[\\/]|/|\\\\)`)
	// L12 asks only for the machine-independent half of the JDK pin (R10-D8): the paths are host keys.
	jdkAuto   = regexp.MustCompile(`^[[:space:]]*org\.gradle\.java\.installations\.auto-detect[[:space:]]*=[[:space:]]*false[[:space:]]*$`)
	pluginLit = regexp.MustCompile(`^[[:space:]]*val[[:space:]]+gradlePluginVersion[[:space:]]*(:[[:space:]]*String[[:space:]]*)?=[[:space:]]*"`)
)

// hostPaths is L10: every gradle.properties under the root (dot directories pruned), then the one in
// each parent directory up to the first holding a .git directory, may not pin an absolute host path.
func (s *scan) hostPaths() {
	for _, p := range s.files(s.root, true, "gradle.properties") {
		s.hostPath(p)
	}
	for dir := s.root; ; {
		up := lint.Dirname(dir)
		if up == dir {
			break
		}
		dir = up
		s.hostPath(dir + "/gradle.properties")
		if lint.IsDir(dir + "/.git") {
			break
		}
	}
}

// hostPath rows one gradle.properties. A host build-location key outside the kit's fixtures and templates is
// a WARN: it belongs in the user-level gradle.properties, where every machine sets its own (R10-D8;
// BUILD-LOOP.md bring-up step 2). Every other host path, and those keys in a kit fixture or template
// (METHODOLOGY.md K16), is the FAIL.
func (s *scan) hostPath(p string) {
	if !lint.IsFile(p) {
		return
	}
	template := inKitTemplate(p)
	for i, ln := range s.lines(p) {
		m := hostPath.FindStringSubmatch(ln)
		switch {
		case m == nil:
		case props.BuildLocationKey(m[1]) && !template:
			s.out = append(s.out, lint.Finding{Severity: lint.Warn, Path: p, Line: i + 1, Msg: "L10: " + m[1] +
				" is a host build-location key: migrate it to the user-level gradle.properties (BUILD-LOOP.md bring-up step 2)"})
		default:
			s.out = append(s.out, lint.Finding{Path: p, Line: i + 1,
				Msg: "L10: absolute host path in gradle.properties (tracked or not: pass host paths with -P, as build.sh does)"})
		}
	}
}

// hostKeys is the L10 key alternation: the shared build-location keys and user_home, regexp-quoted.
func hostKeys() string {
	keys := append(props.BuildLocationKeys(), "user_home")
	for i, k := range keys {
		keys[i] = regexp.QuoteMeta(k)
	}
	return strings.Join(keys, "|")
}

// kitTemplates are the kit trees K16 keeps dev-machine paths out of, matched on the absolute path.
var kitTemplates = []string{"/build-n4-module-kit/fixtures/", "/build-n4-module-kit/templates/"}

// inKitTemplate: p, made absolute so a relative root resolves too, lies under a kit fixture or template.
func inKitTemplate(p string) bool {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	p = filepath.ToSlash(p)
	for _, t := range kitTemplates {
		if strings.Contains(p, t) {
			return true
		}
	}
	return false
}

// jdkPin is L12: the module root's gradle.properties declares org.gradle.java.installations.auto-detect=false,
// uncommented. The paths key is a host build-location key and may live in the user-level file (R10-D8).
func (s *scan) jdkPin() {
	p := s.root + "/gradle.properties"
	if !lint.IsFile(p) {
		return
	}
	if !grep(s.lines(p), jdkAuto) {
		s.row(lint.Unset, s.rel(p), "L12: absent/commented org.gradle.java.installations.auto-detect=false (pin it in the repo; the JDK paths go in the user-level gradle.properties)")
	}
}

// siblingHomes is L13: the first niagara_home= of each <group>/*/gradle.properties (the group being the
// root's parent) must agree; every sibling that sets one gets a row when two or more differ.
func (s *scan) siblingHomes() {
	group := lint.Dirname(s.root)
	if group == s.root || !lint.IsDir(group) {
		return
	}
	entries, _ := os.ReadDir(group) // an unreadable group lists nothing, as the bash glob did
	type home struct{ val, path string }
	var homes []home
	distinct := map[string]bool{}
	for _, e := range entries {
		p := group + "/" + e.Name() + "/gradle.properties"
		if strings.HasPrefix(e.Name(), ".") || !lint.IsFile(p) {
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

// pluginPin is L14, L15 and L16 over the gradle root: the root's own settings.gradle.kts, else the nearest
// ancestor's (the walk stops at a directory holding .git). L14: it may not pin gradlePluginVersion to a
// literal, matched on the script's code lines (comments and literal contents out). L15: the build.gradle.kts
// beside it, when it applies the signing plugin, carries the block. L16: the signing profile its gradle.properties
// names keeps a safe validity/refresh margin.
func (s *scan) pluginPin() {
	settings := ""
	for dir := s.root; ; {
		if p := dir + "/settings.gradle.kts"; lint.IsFile(p) {
			settings = p
			break
		}
		if lint.IsDir(dir + "/.git") {
			break
		}
		up := lint.Dirname(dir)
		if up == dir {
			break
		}
		dir = up
	}
	if settings == "" {
		return
	}
	s.lineRows(settings, s.kotlinLines(settings), pluginLit, `L14: hardcoded gradlePluginVersion literal (use providers.gradleProperty("niagaraPluginVersion").getOrElse(...) so -PniagaraPluginVersion applies)`)
	gradleRoot := strings.TrimSuffix(settings, "/settings.gradle.kts")
	if build := gradleRoot + "/build.gradle.kts"; lint.IsFile(build) {
		s.signing(build)
	}
	s.signingMargin(gradleRoot)
}

// signingMargin is L16 on the gradle root: its gradle.properties names, through niagaraSigningProfile, a
// LocalSigningProfile whose validity - refresh is below signingprofile.MinMargin, so the signing plugin re-mints the
// key at every signing (the rule run-niagara-test enforces too, go/internal/signingprofile). A relative path
// resolves against the gradle root, as layout.projectDirectory.file does; the user-level gradle.properties is out of
// a source lint's scope. No property is no row; a named profile that does not exist is a WARN (margin unchecked);
// one the lint cannot read or parse fails the run (exit 3), never a pass.
func (s *scan) signingMargin(gradleRoot string) {
	props := gradleRoot + "/gradle.properties"
	data, err := os.ReadFile(props)
	if errors.Is(err, fs.ErrNotExist) {
		return
	} else if err != nil {
		s.fail(errors.New("cannot scan " + props + ": " + strerror.Text(err)))
		return
	}
	name, ok := signingprofile.GradleProperty(data, signingprofile.Property)
	if !ok || name == "" {
		return
	}
	profile := name
	if !filepath.IsAbs(profile) {
		profile = gradleRoot + "/" + profile
	}
	b, err := os.ReadFile(profile)
	if errors.Is(err, fs.ErrNotExist) {
		s.row(lint.Warn, props, "L16: "+signingprofile.Property+" names "+profile+", which does not exist — signing-profile validity/refresh margin unchecked")
		return
	} else if err != nil {
		s.fail(errors.New("cannot scan " + profile + ": " + strerror.Text(err)))
		return
	}
	p, err := signingprofile.Parse(b)
	if err != nil {
		s.fail(errors.New("cannot scan " + profile + ": " + err.Error()))
		return
	}
	if why := p.Refusal(); why != "" {
		s.row(lint.Unset, props, "L16: "+signingprofile.Property+" names the "+signingprofile.LocalType+" "+profile+" with "+why)
	}
}

// signing is L15 on one root build file, read as Kotlin (go/internal/ktssrc): it applies the signing plugin when
// an id("com.tridium.niagara-signing") call is code, and the block's names are matched on the code lines, comments
// dropped and literal contents removed. A file it cannot read or scan leaves a visible row, never a silent skip.
func (s *scan) signing(build string) {
	data, err := os.ReadFile(build)
	if err != nil {
		s.row(lint.Warn, build, "L15: could not scan the root for the signing block (awk exit 2: awk: fatal: cannot open file `"+
			build+"' for reading: "+strerror.Text(err)+") — signing block unchecked")
		return
	}
	toks, err := ktssrc.Tokens(string(data))
	if err != nil {
		s.row(lint.Warn, build, "L15: could not scan the root for the signing block ("+err.Error()+") — signing block unchecked")
		return
	}
	if !appliesSigning(toks) {
		return
	}
	code := ktssrc.Code(toks)
	if miss := signblock.Missing(code); len(miss) > 0 {
		s.row(lint.Warn, build, "L15: applies com.tridium.niagara-signing without (uncommented) "+strings.Join(miss, ", ")+
			" — add signingServices { signingProfileFactory { allowDefaultProfile.set(false) } } + niagaraSigning { aliases; signingProfileFile } (templates/signing-root.gradle.kts)")
	}
}

// appliesSigning: the script calls id("com.tridium.niagara-signing").
func appliesSigning(toks []ktssrc.Token) bool {
	for k := 0; k+2 < len(toks); k++ {
		if toks[k].Kind == ktssrc.Ident && toks[k].Text == "id" && toks[k+1].Text == "(" &&
			toks[k+2].Kind == ktssrc.String && toks[k+2].Value == "com.tridium.niagara-signing" {
			return true
		}
	}
	return false
}
