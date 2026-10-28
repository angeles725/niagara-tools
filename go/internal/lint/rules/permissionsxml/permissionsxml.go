// Package permissionsxml is lint-permissions-xml (G21): a module-permissions.xml the security manager does not read as
// written, and an absolute file path in -rt code that only a FILE_WRITE/FILE_READ grant would allow. The contract is
// the deferred G21 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-sign.md, C47); toolbelt/lint-permissions-xml.sh
// is the wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every module-permissions.xml and every *.java under a -rt part's src/
// (srcTest/ and a part's own test/ and build/ not entered). The XML is parsed (comments are not elements, so the kit
// template's commented-out requests are not read; ISO-8859-1 and US-ASCII are decoded). FAIL: no root element (an empty
// or comment-only file) or a root other than <permissions> (baja 4.14.0.162 NModule.java:442-452 reads only
// <permissions>), a <niagara-permission-groups> whose type is not station, workbench or all. WARN: a <req-permission>
// or <opt-permission> without <name> or <purposeKey>; a new File("/…") or Paths.get("/…") whose first argument is an
// absolute string literal in -rt code (Java read as javasrc tokens, so a comment or a string is never one).
package permissionsxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column.
const ID = "lint-permissions-xml"

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), FAIL and WARN rows (exit 1 on
// any FAIL, and on a WARN under --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-permissions-xml.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java", "module-permissions.xml"}, Prune: []string{"srcTest"}, PrunePaths: partOutputs},
		EmptyError: lint.EmptyRow{Msg: emptyMsg}}
}

const emptyMsg = "no module-permissions.xml and no *.java under a -rt part's src/ (wrong path: pass the module root)"

// partOutputs are a part's own test/ and build/ directories, never entered. They are pruned by path, not by name: a
// Java package named test or build under src/ (com/acme/test/) is source and is read (run-3 review R3).
var partOutputs = []string{"*-rt/test", "*-rt/build", "*-ux/test", "*-ux/build", "*-wb/test", "*-wb/build", "*-se/test", "*-se/build"}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var rtPart = regexp.MustCompile(`(^|/)[^/]+-rt/src/`)

// groupTypes are the <niagara-permission-groups> types the devkit wizard and the devguide name.
var groupTypes = map[string]bool{"station": true, "workbench": true, "all": true}

// CheckTree reads every permissions file, then every -rt source.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	read := false
	for _, f := range files {
		switch {
		case baseName(f.Path) == "module-permissions.xml":
			read = true
			fs, err := checkXML(f)
			out = append(out, fs...)
			if err != nil {
				return out, fmt.Errorf("cannot parse %s: %v", f.Path, err)
			}
		case strings.HasSuffix(f.Path, ".java") && rtPart.MatchString(f.Path):
			read = true
			out = append(out, absolutePaths(f)...)
		}
	}
	if !read {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

func baseName(p string) string { return p[strings.LastIndexByte(p, '/')+1:] }

// charsetReader decodes the non-UTF-8 encodings Niagara writes: ISO-8859-1 maps each byte to the rune of the same
// value, and US-ASCII is its subset. Any other declared encoding is an error (exit 3), never a raw pass-through.
func charsetReader(label string, r io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "iso-8859-1", "iso8859-1", "iso_8859-1", "latin1", "latin-1", "l1", "us-ascii", "ascii":
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var u strings.Builder
		u.Grow(len(b))
		for _, c := range b {
			u.WriteRune(rune(c))
		}
		return strings.NewReader(u.String()), nil
	}
	return nil, fmt.Errorf("unsupported encoding %q (UTF-8, ISO-8859-1 or US-ASCII)", label)
}

// request is an open <req-permission> or <opt-permission>: its line and the children seen so far.
type request struct {
	tag, name string
	line      int
	hasName   bool
	hasKey    bool
}

// checkXML parses one module-permissions.xml; a document that does not parse is an error (exit 3).
func checkXML(f *javasrc.File) ([]lint.Finding, error) {
	d := xml.NewDecoder(bytes.NewReader([]byte(strings.Join(f.Raw, "\n"))))
	// Niagara writes UTF-8, ISO-8859-1 or US-ASCII. encoding/xml reads UTF-8 itself; the other two are decoded
	// byte for byte (US-ASCII is a subset of ISO-8859-1), so a non-ASCII name prints as its character.
	d.CharsetReader = charsetReader
	var out []lint.Finding
	add := func(sev lint.Severity, line int, msg string) {
		out = append(out, lint.Finding{Severity: sev, Path: f.Path, Line: line, Msg: msg})
	}
	depth := 0
	rooted := false // a root element was read
	var req *request
	var child string // the local name of the open child element of req
	var text strings.Builder
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			// Mutation: PX-noroot -- an empty or comment-only file read as clean grants nothing and FAILs nothing
			if !rooted {
				add(lint.Fail, 1, "no root element: baja reads only a <permissions> root (NModule), so the file grants nothing")
			}
			return out, nil
		}
		if err != nil {
			return out, err
		}
		line, _ := d.InputPos()
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			rooted = true
			name := t.Name.Local
			switch {
			case depth == 1 && name != "permissions":
				add(lint.Fail, line, fmt.Sprintf("root element <%s> is not <permissions>: baja reads only <permissions> (NModule), so every request is ignored", name))
			case name == "niagara-permission-groups":
				typ, ok := attr(t, "type")
				if !ok {
					add(lint.Fail, line, "<niagara-permission-groups> has no type (station, workbench or all)")
				} else if !groupTypes[typ] {
					add(lint.Fail, line, fmt.Sprintf("<niagara-permission-groups type=%q> is not station, workbench or all", typ))
				}
			case name == "req-permission" || name == "opt-permission":
				req = &request{tag: name, line: line}
			case req != nil && depth > 0:
				child = name
				text.Reset()
			}
		case xml.CharData:
			if child != "" {
				text.Write(t)
			}
		case xml.EndElement:
			depth--
			name := t.Name.Local
			switch {
			case req != nil && name == req.tag:
				if msg := req.missing(); msg != "" {
					add(lint.Warn, req.line, msg)
				}
				req = nil
			case req != nil && name == child:
				v := strings.TrimSpace(text.String())
				if child == "name" && v != "" {
					req.hasName, req.name = true, v
				}
				if child == "purposeKey" && v != "" {
					req.hasKey = true
				}
				child = ""
			}
		}
	}
}

// missing words the WARN of a request without <name> or <purposeKey>, or "" when it has both.
func (r *request) missing() string {
	var what string
	switch {
	case !r.hasName && !r.hasKey:
		what = "<name> or <purposeKey>"
	case !r.hasName:
		what = "<name>"
	case !r.hasKey:
		what = "<purposeKey>"
	default:
		return ""
	}
	label := "<" + r.tag + ">"
	if r.name != "" {
		label += " " + r.name
	}
	return label + " has no " + what + " (both <name> and <purposeKey> are required)"
}

func attr(e xml.StartElement, name string) (string, bool) {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// absolutePaths finds new File("/…") and Paths.get("/…") whose first argument is an absolute string literal.
func absolutePaths(f *javasrc.File) []lint.Finding {
	toks := javasrc.Tokens(f.Stripped())
	var out []lint.Finding
	for k := 2; k < len(toks); k++ {
		lit := toks[k]
		if lit.Kind != javasrc.TokLiteral || !strings.HasPrefix(lit.Text, `"/`) || toks[k-1].Text != "(" {
			continue
		}
		callee := ""
		switch {
		case toks[k-2].Text == "File" && isNew(toks, k-2):
			callee = "new File"
		case toks[k-2].Text == "get" && k >= 4 && toks[k-3].Text == "." && toks[k-4].Text == "Paths":
			callee = "Paths.get"
		default:
			continue
		}
		rest := ")"
		if k+1 < len(toks) && toks[k+1].Text != ")" {
			rest = ", …)"
		}
		out = append(out, lint.Finding{Path: f.Path, Line: lit.Line + 1, Msg: callee + "(" + lit.Text + rest +
			" is an absolute path outside the station and user homes: the security manager refuses it without a FILE_READ or FILE_WRITE grant for it (use Sys.getStationHome())"})
	}
	return out
}

// isNew reports whether the type name at toks[i] is the class of a `new` expression, qualified or not
// (new File, new java.io.File).
func isNew(toks []javasrc.Token, i int) bool {
	for i >= 2 && toks[i-1].Text == "." && toks[i-2].Kind == javasrc.TokIdent {
		i -= 2
	}
	return i >= 1 && toks[i-1].Text == "new"
}
