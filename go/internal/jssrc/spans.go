package jssrc

import "sort"

// SpanKind tells a comment from the content of a literal.
type SpanKind int

// The span kinds.
const (
	Comment SpanKind = iota // a // or /* */ comment, markers included
	Literal                 // what sits between the delimiters of a string, template or regex literal
)

// Span is the byte range [From, To) of src that one comment or literal covers.
type Span struct {
	Kind     SpanKind
	From, To int
}

// mark records a span the scan passed.
func (s *scanner) mark(k SpanKind, from, to int) { s.spans = append(s.spans, Span{k, from, to}) }

// Spans is every comment and literal-content span of src, found in one pass and ordered by From. A template's
// content span holds its ${...} substitutions, whose own comments and literals are spans of their own; a
// comment or string that never closes runs to the line end (a string) or to the end of src.
func Spans(src string) []Span {
	s := &scanner{src: src}
	s.run(false)
	sort.SliceStable(s.spans, func(i, j int) bool { return s.spans[i].From < s.spans[j].From })
	return s.spans
}

// CommentAt reports that byte off of src is inside a comment (its // or /* */ markers included, the line break
// that ends a // comment not). Use Spans once when asking about many offsets of one source.
func CommentAt(src string, off int) bool {
	for _, sp := range Spans(src) {
		if sp.Kind == Comment && sp.From <= off && off < sp.To {
			return true
		}
	}
	return false
}

// LineCommentAt is the index of the // that starts a comment on one line of JavaScript, or -1: a // inside a
// '...', "..." or `...` literal or a regex literal is no comment, a literal still open at the line end is read to
// the end, an inline /* ... */ is skipped, and a /* that does not close ends the search.
func LineCommentAt(line string) int {
	for _, sp := range Spans(line) {
		if sp.Kind == Comment && line[sp.From:sp.From+2] == "//" {
			return sp.From
		}
	}
	return -1
}
