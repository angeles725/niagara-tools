package xmlsrc

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// U+0000 is not an XML character, but Niagara writes it into a .bog as "&#x0;" (nre javax.baja.xml.XWriter.safe,
// XWriter.java:331-339, escapes every char below 0x20) and reads it back: XParser.toCharData refuses only '<' and
// returns (char)x for any &#...; (XParser.java:958-1030), and read() passes a literal NUL byte through
// (XParser.java:916-947) into text, attribute values, comments and CDATA; only a name refuses it (isName,
// XParser.java:1265). So a bajaObjectGraph is read the same way, with one warning; any other document still
// refuses U+0000, as XML 1.0 and encoding/xml do.

// nulMark stands in for U+0000 while encoding/xml reads the text: a noncharacter, which XML allows in text and
// attribute values but not in names.
const nulMark = "﷐"

var (
	nulAny = regexp.MustCompile(`&#(?:0+|x0+);|\x00`)
	// markAny is nulMark as written; beside a U+0000 it could not be told from one.
	markAny = regexp.MustCompile(`\x{FDD0}|&#(?:0*64976|x0*[Ff][Dd][Dd]0);`)
)

// mark is one U+0000 replaced by nulMark: at is its offset in the masked text, orig..end its bytes as written.
type mark struct{ at, orig, end int64 }

// mask sets p.text to text with every U+0000 (a literal byte, or a character reference outside CDATA) replaced
// by nulMark, and p.marks to where.
func (p *parser) mask(text []byte) error {
	p.text = text
	if bytes.IndexByte(text, 0) < 0 && !bytes.Contains(text, []byte("&#0")) && !bytes.Contains(text, []byte("&#x0")) {
		return nil // the common case, without a regexp pass over the whole text
	}
	locs := nulAny.FindAllIndex(text, -1)
	if len(locs) == 0 {
		return nil
	}
	if l := markAny.FindIndex(text); l != nil {
		return p.failAt(int64(l[0]), "U+FDD0 and U+0000 in one document are not supported")
	}
	cdata := cdataSpans(text)
	out := make([]byte, 0, len(text))
	last := 0
	for _, l := range locs {
		if text[l[0]] == '&' && inSpan(cdata, l[0]) {
			continue // CDATA reads "&#0;" as those four characters
		}
		out = append(out, text[last:l[0]]...)
		p.marks = append(p.marks, mark{at: int64(len(out)), orig: int64(l[0]), end: int64(l[1])})
		out = append(out, nulMark...)
		last = l[1]
	}
	p.text = append(out, text[last:]...)
	return nil
}

// unmasked maps an offset of the masked text onto the text as written; inside a mark it is the mark's start.
func (p *parser) unmasked(off int64) int64 {
	i := sort.Search(len(p.marks), func(i int) bool { return p.marks[i].at > off }) - 1
	if i < 0 {
		return off
	}
	m := p.marks[i]
	if off < m.at+int64(len(nulMark)) {
		return m.orig
	}
	return m.end + off - m.at - int64(len(nulMark))
}

// nulVerdict turns the result of run over a masked text into Parse's. Without marks err stands. A name holding
// one is refused at it, as Niagara refuses it. Any other document refuses its first U+0000 unless an error comes
// before it; a bajaObjectGraph keeps its error, or carries the warning when it has none.
func (p *parser) nulVerdict(err error) (*Error, error) {
	if len(p.marks) == 0 {
		return nil, err
	}
	first := p.failAt(p.marks[0].orig, "illegal character code U+0000")
	var xe *Error
	if errors.As(err, &xe) && strings.Contains(xe.Msg, nulMark) { // encoding/xml quotes the name it refused
		at := p.marks[0].orig
		for _, m := range p.marks {
			if before(p.failAt(m.orig, ""), xe) {
				at = m.orig
			}
		}
		return nil, p.failAt(at, "illegal character code U+0000 in a name")
	}
	if p.root != "bajaObjectGraph" && (err == nil || before(first, xe)) {
		return nil, first
	}
	if err != nil {
		return nil, err
	}
	places := "1 place"
	if n := len(p.marks); n > 1 {
		places = fmt.Sprintf("%d places", n)
	}
	first.Msg = "U+0000 in " + places + " (Niagara reads it; XML 1.0 forbids it)"
	return first, nil
}

func before(a, b *Error) bool { return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col }

func unmask(s string, masked bool) string {
	if !masked {
		return s
	}
	return strings.ReplaceAll(s, nulMark, "\x00")
}

// cdataSpans are the [start, end) offsets of every CDATA section.
func cdataSpans(text []byte) [][2]int {
	var out [][2]int
	for i := 0; ; {
		j := bytes.Index(text[i:], []byte("<![CDATA["))
		if j < 0 {
			return out
		}
		k := bytes.Index(text[i+j:], []byte("]]>"))
		if k < 0 {
			return append(out, [2]int{i + j, len(text)})
		}
		out = append(out, [2]int{i + j, i + j + k})
		i += j + k
	}
}

func inSpan(spans [][2]int, off int) bool {
	for _, s := range spans {
		if off >= s[0] && off < s[1] {
			return true
		}
	}
	return false
}
