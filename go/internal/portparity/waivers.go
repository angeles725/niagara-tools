package portparity

import (
	"regexp"
	"strings"
)

// Waiver is one line of a parity file: `<kind> <name> -- <reason> [decision: <ref>]`, or a rename:
// `rename: <ref-file> -> <port-file>` (a file under another basename; optional ` -- <reason>`), or
// `rename: <function|id> <ref> -> <port> -- <reason>` (the behaviour kept under another name; the reason is
// required, checked by Check). Problem is set, and the other fields empty, for a line that matches no shape.
type Waiver struct {
	Line             int
	Kind             Kind
	Of               Kind // a rename's item kind: KindFile, KindID or KindFunction
	Name, To         string
	Reason, Decision string
	Problem          string
}

// WaiverFile is a parsed parity file: the optional `ref:` / `port:` headers (paths relative to the file's
// directory) and its waivers in line order. Blank lines and `#` comments are ignored.
type WaiverFile struct {
	Ref, Port string
	Waivers   []Waiver
}

const shape = `malformed waiver (want "<file|id|function> <name> -- <reason> [decision: <ref>]" or "rename: [function|id] <ref> -> <port> -- <reason>"): `

var (
	headerRe   = regexp.MustCompile(`^(ref|port):\s*(.*)$`)
	renameRe   = regexp.MustCompile(`^rename:\s*(?:(id|function)\s+)?(\S+)\s*->\s*(\S+)(?:\s+--\s+(.*))?$`)
	waiverRe   = regexp.MustCompile(`^(file|id|function)\s+(\S+)\s+--\s+(.*)$`)
	decisionRe = regexp.MustCompile(`\[decision:\s*([^\]]*)\]`)
)

// ParseWaivers parses a parity file's text.
func ParseWaivers(text string) WaiverFile {
	var wf WaiverFile
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n := i + 1
		if m := headerRe.FindStringSubmatch(line); m != nil {
			if m[1] == "ref" {
				wf.Ref = strings.TrimSpace(m[2])
			} else {
				wf.Port = strings.TrimSpace(m[2])
			}
			continue
		}
		if m := renameRe.FindStringSubmatch(line); m != nil {
			of := KindFile
			if m[1] != "" {
				of = Kind(m[1])
			}
			wf.Waivers = append(wf.Waivers, Waiver{Line: n, Kind: KindRename, Of: of, Name: m[2], To: m[3], Reason: strings.TrimSpace(m[4])})
			continue
		}
		m := waiverRe.FindStringSubmatch(line)
		if m == nil || m[2] == "--" {
			wf.Waivers = append(wf.Waivers, Waiver{Line: n, Problem: shape + line})
			continue
		}
		w := Waiver{Line: n, Kind: Kind(m[1]), Name: m[2], Reason: m[3]}
		if d := decisionRe.FindStringSubmatchIndex(w.Reason); d != nil {
			w.Decision = strings.TrimSpace(w.Reason[d[2]:d[3]])
			w.Reason = w.Reason[:d[0]] + w.Reason[d[1]:]
		}
		if w.Reason = strings.TrimSpace(w.Reason); w.Reason == "" {
			wf.Waivers = append(wf.Waivers, Waiver{Line: n, Problem: shape + line})
			continue
		}
		wf.Waivers = append(wf.Waivers, w)
	}
	return wf
}
