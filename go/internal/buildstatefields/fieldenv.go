package buildstatefields

import (
	"fmt"
	"regexp"
	"strings"
)

// fieldEnv is what one build-state.v1 envelope says about a set of optional fields: its module, the value of every
// `<name>:` field at column 0 (comment removed, blanks trimmed) and every line that is a field name in some other
// shape (strays). It is signerEnv generalised to several fields, for the fields that came after expected_signer.
type fieldEnv struct {
	module string
	values map[string][]string
	strays map[string][]string
}

// fieldEnvelopes reads every envelope of a text, in order, for the named fields. Like signerEnvelopes: names start
// at column 0 and a `# comment` ends the value; a byte-order mark that opens the text is dropped and a CR ends a line.
func fieldEnvelopes(text string, names ...string) (found []fieldEnv) {
	stray := map[string]*regexp.Regexp{}
	for _, n := range names {
		stray[n] = regexp.MustCompile(`^[ \t]*` + regexp.QuoteMeta(n) + `[ \t]*:`)
	}
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\xef\xbb\xbf"), "\r", "\n")
	var cur *fieldEnv
	for _, line := range Lines(text) {
		switch {
		case strings.HasPrefix(line, OpenMarker):
			found = append(found, fieldEnv{values: map[string][]string{}, strays: map[string][]string{}})
			cur = &found[len(found)-1]
		case strings.HasPrefix(line, CloseMarker):
			cur = nil
		case cur != nil:
			name, val, ok := strings.Cut(line, ":")
			if i := strings.IndexByte(val, '#'); i >= 0 {
				val = val[:i]
			}
			if _, known := stray[name]; ok && known {
				cur.values[name] = append(cur.values[name], trimSpace(val))
				continue
			}
			if ok && name == "module" && cur.module == "" {
				cur.module = trimSpace(val)
				continue
			}
			for _, n := range names {
				if stray[n].MatchString(line) {
					cur.strays[n] = append(cur.strays[n], line)
				}
			}
		}
	}
	return found
}

// fieldRule is one optional field: its name, what a valid value looks like (for the message) and the test for it.
// `unknown` is always valid and records nothing, the same as leaving the field out.
type fieldRule struct {
	name, want string
	valid      func(string) bool
}

// Messages of the structural errors of a fieldRule: the first %s is the module, the field name follows.
const (
	fieldTwice     = "envelope (%s) %s is given twice: '%s' and '%s'"
	fieldStray     = "envelope (%s) has a line that is not the %s field: '%s' (write %s: <value> at column 0)"
	fieldInvalid   = "envelope (%s) %s invalid: '%s' (want %s)"
	fieldTwoModule = "envelope (%s) is declared twice and %s is set in one of them: keep one envelope"
)

// problem is the first error of the envelopes of one module (every envelope of a text for the sweep) for the field
// of r, "" when they are well formed: a line that mistypes the name, the field twice in one envelope, a value that
// is not valid, then two envelopes of the module when any of them sets the field.
func (r fieldRule) problem(envs []fieldEnv) string {
	for _, e := range envs {
		v := e.values[r.name]
		switch {
		case len(e.strays[r.name]) > 0:
			return fmt.Sprintf(fieldStray, e.module, r.name, e.strays[r.name][0], r.name)
		case len(v) > 1:
			return fmt.Sprintf(fieldTwice, e.module, r.name, v[0], v[1])
		case len(v) == 1 && v[0] != SignerUnknown && !r.valid(v[0]):
			return fmt.Sprintf(fieldInvalid, e.module, r.name, v[0], r.want)
		}
	}
	seen := map[string]bool{}
	for _, e := range envs {
		if e.module != "" && seen[e.module] {
			for _, o := range envs {
				if o.module == e.module && len(o.values[r.name]) > 0 {
					return fmt.Sprintf(fieldTwoModule, e.module, r.name)
				}
			}
		}
		seen[e.module] = true
	}
	return ""
}
