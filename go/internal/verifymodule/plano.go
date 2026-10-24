package verifymodule

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/jarversion"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

var (
	imgWRe     = rx(`IMG_W[[:space:]]*=[[:space:]]*[0-9]+`)
	imgHRe     = rx(`IMG_H[[:space:]]*=[[:space:]]*[0-9]+`)
	digitsEnd  = rx(`[0-9]+$`)
	overlayRe  = regexp.MustCompile(`id="zonas"|id="plano"`)
	viewBoxRe  = rx(`viewBox="[^"]*"`)
	pngB64Re   = rx(`data:image/png;base64,[A-Za-z0-9+/=]+`)
	aspectRe   = rx(`aspect-ratio:[[:space:]]*[^;{}]+`)
	aspectHead = regexp.MustCompile(`aspect-ratio:[[:space:]]*`)
	ratioGlob  = regexp.MustCompile(`(?s)^[0-9].*/[0-9].*$`) // case [0-9]*/[0-9]*
	pngMagic   = "\x89PNG\r\n\x1a\n"
)

// plano is --plano: Rc (IMG_W/IMG_H) == Ri (the #planoImg PNG header) == Rv (the zones viewBox) and every
// numeric aspect-ratio == Rc, by integer cross-multiplication. ok is false when bash arithmetic failed on
// an operand: the bash then left the --plano block and ran the gate on the same argument.
func plano(args []string, s cli.Stdio) (code int, ok bool) {
	if len(args) != 1 {
		fmt.Fprintln(s.Err, "usage: verify-module.sh --plano <index.html|jar>")
		return 2, true
	}
	in, html := args[0], ""
	if !lint.IsFile(in) {
		fmt.Fprintf(s.Err, "verify-module: file not readable: %s\n", in)
		return 3, true
	}
	if strings.HasSuffix(in, ".jar") {
		var found bool
		if html, found = jarEntry(in, "rc/index.html"); !found {
			fmt.Fprintf(s.Err, "verify-module: no rc/index.html in: %s\n", in)
			return 3, true
		}
	} else if data, err := os.ReadFile(in); err == nil {
		html = string(data)
	} else { // the two IMG_W/IMG_H greps had no 2>/dev/null
		msg := "grep: " + in + ": " + strerror.Text(err) + "\n"
		fmt.Fprint(s.Err, msg+msg)
	}
	p := planoRun{s: s, in: in}
	return p.check(newGrepText(html))
}

// jarEntry is `unzip -p jar name`: false when the jar cannot be read or holds no such entry.
func jarEntry(path, name string) (string, bool) {
	zr, err := jarversion.Open(path)
	if err != nil {
		return "", false
	}
	defer zr.Close()
	_, first := jarversion.Index(zr)
	f, ok := first[name]
	if !ok {
		return "", false
	}
	data, err := jarversion.Read(f)
	return string(data), err == nil
}

type planoRun struct {
	s    cli.Stdio
	in   string
	fail bool
}

func (p *planoRun) row(status, detail string) {
	fmt.Fprintf(p.s.Out, "%-4s  %-9s  %s  %s\n", status, "plano", p.in, detail)
}

func (p *planoRun) check(t grepText) (int, bool) {
	rcW, rcH := firstDigits(t.only(imgWRe)), firstDigits(t.only(imgHRe))
	if rcW == "" || rcH == "" {
		if t.count(overlayRe) == 0 {
			p.row("SKIP", "not a plano dashboard (3D/self-contained SPA) — IMG_W/IMG_H and overlay markers absent")
			return 0, true
		}
		p.row("FAIL", "Rc: IMG_W/IMG_H not found")
		return 1, true
	}
	vb := ""
	if ms := linesHolding(t, `id="zonas"`).only(viewBoxRe); len(ms) > 0 {
		if f := awkFields(strings.TrimSuffix(strings.TrimPrefix(ms[0], `viewBox="`), `"`)); len(f) >= 4 {
			vb = f[2] + " " + f[3]
		}
	}
	if vb == "" {
		p.row("FAIL", `Rv: no id="zonas" viewBox found`)
		return 1, true
	}
	rv := awkFields(vb)
	b64 := ""
	if ms := linesHolding(t, `id="planoImg"`).only(pngB64Re); len(ms) > 0 {
		b64 = strings.TrimPrefix(ms[0], "data:image/png;base64,")
		b64 = b64[:min(64, len(b64))]
	}
	if b64 == "" {
		p.row("FAIL", "Ri: no #planoImg with data:image/png;base64 found")
		return 1, true
	}
	riW, riH := pngSize(b64)
	if riW == 0 || riH == 0 {
		p.row("FAIL", "Ri: #planoImg PNG header invalid or unreadable")
		return 1, true
	}
	ri := []string{itoa(int(riW)), itoa(int(riH))}
	steps := []func() (bool, *arithErr){
		func() (bool, *arithErr) {
			return p.unless(rcW, rcH, ri[0], ri[1], fmt.Sprintf("Rc(%s/%s) != Ri(%s/%s): IMG_W/IMG_H vs intrinsic PNG", rcW, rcH, ri[0], ri[1]))
		},
		func() (bool, *arithErr) {
			return p.unless(rcW, rcH, rv[0], rv[1], fmt.Sprintf("Rv(%s/%s) != Rc(%s/%s): viewBox vs IMG_W/IMG_H", rv[0], rv[1], rcW, rcH))
		},
	}
	for _, ln := range linesHolding(t, "aspect-ratio:").lines {
		steps = append(steps, func() (bool, *arithErr) { return p.aspect(ln, rcW, rcH) })
	}
	for _, step := range steps {
		if _, err := step(); err != nil {
			fmt.Fprintf(p.s.Err, "verify-module: %s\n", err.msg)
			if err.unbound {
				return 1, true
			}
			return 0, false
		}
	}
	if !p.fail {
		p.row("PASS", fmt.Sprintf("Rc=%s/%s Ri=%s/%s Rv=%s/%s — all agree", rcW, rcH, ri[0], ri[1], rv[0], rv[1]))
		return 0, true
	}
	return 1, true
}

// unless prints a FAIL row with msg unless a/b == c/d (a*d == b*c in bash arithmetic).
func (p *planoRun) unless(a, b, c, d, msg string) (bool, *arithErr) {
	var v [4]int64
	for i, x := range []string{a, d, b, c} {
		n, err := bashInt(x)
		if err != nil {
			return false, err
		}
		v[i] = n
	}
	if v[0]*v[1] != v[2]*v[3] {
		p.row("FAIL", msg)
		p.fail = true
	}
	return true, nil
}

// aspect checks every aspect-ratio of one CSS line against Rc, as the bash read it: all of the line's
// values joined by newlines, blanks and tabs dropped, then cut at its first and last "/".
func (p *planoRun) aspect(ln, rcW, rcH string) (bool, *arithErr) {
	var vals []string
	for _, m := range newGrepText(ln).only(aspectRe) {
		k := aspectHead.FindStringIndex(m) // sed s/aspect-ratio:[[:space:]]*//: the first one only
		vals = append(vals, strings.TrimRight(m[:k[0]]+m[k[1]:], " \t\n\v\f\r"))
	}
	raw := subst(strings.Join(vals, "\n"))
	v := strings.NewReplacer(" ", "", "\t", "").Replace(raw)
	if raw == "" || v == "auto" {
		return true, nil
	}
	if !ratioGlob.MatchString(v) {
		p.row("FAIL", "unparseable aspect-ratio: "+v)
		p.fail = true
		return true, nil
	}
	n, d := v[:strings.IndexByte(v, '/')], v[strings.LastIndexByte(v, '/')+1:]
	if n == "" || d == "" || strings.Trim(n, "0123456789") != "" || strings.Trim(d, "0123456789") != "" {
		p.row("FAIL", "unparseable aspect-ratio: "+v)
		p.fail = true
		return true, nil
	}
	return p.unless(n, d, rcW, rcH, fmt.Sprintf("aspect-ratio %s != Rc(%s/%s)", v, rcW, rcH))
}

// firstDigits is `| grep -oE '[0-9]+$' | head -1` over grep -o matches.
func firstDigits(ms []string) string {
	for _, m := range ms {
		if d := digitsEnd.FindString(m); d != "" {
			return d
		}
	}
	return ""
}

// linesHolding is what `grep <fixed text> file` prints, as the next grep in the pipe reads it.
func linesHolding(t grepText, s string) grepText {
	var out grepText
	for _, ln := range t.lines {
		if strings.Contains(ln, s) && t.shown(ln) {
			out.lines = append(out.lines, ln)
		}
	}
	return out
}

// pngSize is the IHDR width and height of `base64 -d | od -N24`: coreutils decodes whole 4-character
// groups until the first bad one, and a header under 24 bytes or without the PNG magic and IHDR is 0 0.
func pngSize(b64 string) (uint32, uint32) {
	var data []byte
	for i := 0; i+4 <= len(b64); i += 4 {
		g, err := base64.StdEncoding.DecodeString(b64[i : i+4])
		if err != nil {
			break
		}
		data = append(data, g...)
	}
	if len(data) < 24 || string(data[:8]) != pngMagic || string(data[12:16]) != "IHDR" {
		return 0, 0
	}
	return binary.BigEndian.Uint32(data[16:20]), binary.BigEndian.Uint32(data[20:24])
}
