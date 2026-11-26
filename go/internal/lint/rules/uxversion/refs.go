package uxversion

import (
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

// ref is one reference to a resource, as written, and the 1-based line it is on (tag: the line of the tag it
// belongs to, which is line itself outside a multi-line tag).
// A ref with a chk is judged by chk instead of its text (a ?v= built in JavaScript); one with dynamic set is
// that expression, whose version is not known here.
type ref struct {
	text      string
	line, tag int
	chk       string
	dynamic   string
}

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	cssComment  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// element bodies: <script ...>...</script> and <style ...>...</style>; group 2 is the body.
	scriptBody = regexp.MustCompile(`(?is)(<script\b[^>]*>)(.*?)(</script\s*>)`)
	styleBody  = regexp.MustCompile(`(?is)(<style\b[^>]*>)(.*?)(</style\s*>)`)
	// the tags that load a resource, and their attributes that name it.
	tagRe  = regexp.MustCompile(`(?is)<(link|script|img|source|video|audio|image|input|iframe)\b[^>]*>`)
	attrRe = regexp.MustCompile(`(?is)\s(href|src|poster|xlink:href)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>"']+))`)
	// url(...) in CSS, in a style attribute and in a JavaScript string; @import "..." in CSS.
	urlRe    = regexp.MustCompile(`(?i)url\(\s*(?:"([^"]*)"|'([^']*)'|([^)\s"']*))\s*\)`)
	importRe = regexp.MustCompile(`(?i)@import\s+(?:"([^"]*)"|'([^']*)')`)
	// a JavaScript string that is one path to a static asset (a script, a style, an image or a font).
	assetRe = regexp.MustCompile(`(?i)^[^\s"'()<>\\]+\.(js|css|png|jpe?g|gif|svg|webp|avif|ico|bmp|woff2?|ttf|otf|eot)([?#]\S*)?$`)
)

// tagAttrs is the attributes of each tag that name a resource to load (a <a href> is a link, not a load).
var tagAttrs = map[string]string{
	"link": " href ", "script": " src ", "img": " src ", "source": " src ", "audio": " src ",
	"video": " src poster ", "input": " src ", "iframe": " src ", "image": " href xlink:href ",
}

// blankAll is s with every byte but a line break replaced by a blank, so offsets and lines stay.
func blankAll(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c != '\n' {
			b[i] = ' '
		}
	}
	return string(b)
}

// blank replaces every match of re in src by blanks, keeping line breaks.
func blank(re *regexp.Regexp, src string) string { return re.ReplaceAllStringFunc(src, blankAll) }

// mapBodies is src with fn applied to the body of every element body matches.
func mapBodies(src string, body *regexp.Regexp, fn func(string) string) string {
	var sb strings.Builder
	last := 0
	for _, m := range body.FindAllStringSubmatchIndex(src, -1) {
		sb.WriteString(src[last:m[4]])
		sb.WriteString(fn(src[m[4]:m[5]]))
		last = m[5]
	}
	sb.WriteString(src[last:])
	return sb.String()
}

// lineIndex numbers the lines of src: lineAt(off) is the 1-based line of byte offset off.
type lineIndex []int

func indexLines(src string) lineIndex {
	idx := lineIndex{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			idx = append(idx, i+1)
		}
	}
	return idx
}

func (l lineIndex) lineAt(off int) int {
	return sort.Search(len(l), func(i int) bool { return l[i] > off })
}

// htmlRefs is the references of an HTML page: the resource attributes of its tags, url() in its styles and
// style attributes, and the asset strings of its inline scripts. Comments are not read.
func htmlRefs(lines []string) []ref {
	src := blank(htmlComment, strings.Join(lines, "\n"))
	var refs, markup []ref
	for _, t := range jssrc.Tokens(jssrc.HTMLScripts(strings.Split(src, "\n"))) {
		refs = append(refs, jsRefsOf(t)...)
	}
	outside := mapBodies(src, scriptBody, blankAll)
	outside = mapBodies(outside, styleBody, func(b string) string { return blank(cssComment, b) })
	idx := indexLines(outside)
	for _, m := range tagRe.FindAllStringSubmatchIndex(outside, -1) {
		name := strings.ToLower(outside[m[2]:m[3]])
		tag := idx.lineAt(m[0])
		for _, a := range attrRe.FindAllStringSubmatchIndex(outside[m[0]:m[1]], -1) {
			attr := strings.ToLower(outside[m[0]+a[2] : m[0]+a[3]])
			if !strings.Contains(tagAttrs[name], " "+attr+" ") {
				continue
			}
			markup = append(markup, ref{text: group(outside[m[0]:m[1]], a, 4), line: idx.lineAt(m[0] + a[2]), tag: tag})
		}
	}
	markup = append(markup, urlRefs(outside, idx)...)
	for i := range markup {
		// Mutation: UXV-entity -- reading an attribute value undecoded sees a&amp;v=1.2.3 as one parameter named amp;v
		markup[i].text = htmltext.Unescape(markup[i].text) // an attribute value is HTML text: &amp; is &
	}
	return append(refs, markup...)
}

// cssRefs is the url(...) and @import references of a stylesheet, comments not read.
func cssRefs(lines []string) []ref {
	src := blank(cssComment, strings.Join(lines, "\n"))
	return urlRefs(src, indexLines(src))
}

// urlRefs is the url(...) and @import "..." references of src.
func urlRefs(src string, idx lineIndex) []ref {
	var refs []ref
	for _, re := range []*regexp.Regexp{urlRe, importRe} {
		for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
			n := idx.lineAt(m[0])
			refs = append(refs, ref{text: group(src, m, 2), line: n, tag: n})
		}
	}
	return refs
}

// group is the text of the first capture group from index from (2 is group 1) of the match m of src that took
// part in it (a double-quoted, a single-quoted or a bare value), "" when none did.
func group(src string, m []int, from int) string {
	for g := from; g+1 < len(m); g += 2 {
		if m[g] >= 0 {
			return strings.TrimSpace(src[m[g]:m[g+1]])
		}
	}
	return ""
}

// jsRefs is the references of a script's tokens: its string and template literals, comments not read.
func jsRefs(toks []jssrc.Token) []ref {
	var refs []ref
	for _, t := range toks {
		refs = append(refs, jsRefsOf(t)...)
	}
	return refs
}

// jsRefsOf is the references in one token: a string that holds url(...), or is one path to a static asset.
func jsRefsOf(t jssrc.Token) []ref {
	if t.Kind != jssrc.String && t.Kind != jssrc.Template {
		return nil
	}
	n := t.Line + 1
	if strings.Contains(t.Text, "url(") {
		var refs []ref
		for _, m := range urlRe.FindAllStringSubmatchIndex(t.Text, -1) {
			refs = append(refs, ref{text: group(t.Text, m, 2), line: n, tag: n})
		}
		return refs
	}
	if assetRe.MatchString(t.Text) {
		return []ref{{text: t.Text, line: n, tag: n}}
	}
	return nil
}
