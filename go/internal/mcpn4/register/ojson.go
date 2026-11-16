package register

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Harness config files are hand-edited, so an entry is spliced in by byte offsets: only the bytes
// of the entry added or replaced change, plus the comma and line break that join it to its
// neighbours. Every other member keeps its bytes, order and formatting exactly.

// span is one member of an object: offsets into the document.
type span struct {
	key                      string
	keyEnd, valStart, valEnd int
}

// objSpan is one object: the offsets of its braces and its members.
type objSpan struct {
	open, close int
	members     []span
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// scanObject reads the object that starts at doc[at] (after optional white space).
func scanObject(doc []byte, at int) (objSpan, error) {
	i := at
	for i < len(doc) && isSpace(doc[i]) {
		i++
	}
	if i >= len(doc) || doc[i] != '{' {
		return objSpan{}, fmt.Errorf("not a JSON object")
	}
	o := objSpan{open: i}
	dec := json.NewDecoder(bytes.NewReader(doc[i:]))
	if _, err := dec.Token(); err != nil {
		return o, err
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return o, err
		}
		m := span{key: kt.(string), keyEnd: i + int(dec.InputOffset())}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return o, err
		}
		m.valEnd = i + int(dec.InputOffset())
		for m.valStart = m.keyEnd; isSpace(doc[m.valStart]) || doc[m.valStart] == ':'; m.valStart++ {
		}
		o.members = append(o.members, m)
	}
	if _, err := dec.Token(); err != nil {
		return o, err
	}
	o.close = i + int(dec.InputOffset()) - 1
	return o, nil
}

func (o objSpan) find(key string) *span {
	for i := range o.members {
		if o.members[i].key == key {
			return &o.members[i]
		}
	}
	return nil
}

// getPath returns the value at path in doc (an empty doc is {}) and whether it is there.
func getPath(doc []byte, path []string) (json.RawMessage, bool, error) {
	if len(bytes.TrimSpace(doc)) == 0 {
		return nil, false, nil
	}
	at := 0
	for i, k := range path {
		o, err := scanObject(doc, at)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", pathName(path[:i]), err)
		}
		m := o.find(k)
		if m == nil {
			return nil, false, nil
		}
		if i == len(path)-1 {
			return doc[m.valStart:m.valEnd], true, nil
		}
		at = m.valStart
	}
	return nil, false, nil
}

// setPath sets the value v (compact JSON) at path in doc, creating missing objects and appending
// new keys last. A new or empty document is written indented by two spaces with a final newline.
func setPath(doc []byte, path []string, v []byte) ([]byte, error) {
	if len(bytes.TrimSpace(doc)) == 0 {
		return append(render(nest(path, v), "", true), '\n'), nil
	}
	multi := bytes.Contains(bytes.TrimSpace(doc), []byte("\n"))
	at := 0
	for i, k := range path {
		o, err := scanObject(doc, at)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pathName(path[:i]), err)
		}
		m := o.find(k)
		if m == nil {
			return insert(doc, o, k, nest(path[i+1:], v), multi), nil
		}
		if i == len(path)-1 {
			return splice(doc, m.valStart, m.valEnd, render(v, lineIndent(doc, m.keyEnd), multi)), nil
		}
		at = m.valStart
	}
	return doc, nil
}

// insert adds "key": v as the last member of o, in the layout of its last member (or, for an
// empty object, one level deeper than the line of its opening brace).
func insert(doc []byte, o objSpan, key string, v []byte, multi bool) []byte {
	k := string(encodeKey(key))
	if !multi {
		sep := ""
		if len(o.members) > 0 {
			sep = ","
		}
		return splice(doc, o.close, o.close, []byte(sep+k+":"+string(v)))
	}
	if len(o.members) > 0 {
		last := o.members[len(o.members)-1]
		ind := lineIndent(doc, last.keyEnd)
		return splice(doc, last.valEnd, last.valEnd, []byte(",\n"+ind+k+": "+string(render(v, ind, true))))
	}
	ind := lineIndent(doc, o.open)
	inner := ind + "  "
	return splice(doc, o.open+1, o.close, []byte("\n"+inner+k+": "+string(render(v, inner, true))+"\n"+ind))
}

// nest wraps v in one object per key of path.
func nest(path []string, v []byte) []byte {
	for i := len(path) - 1; i >= 0; i-- {
		v = []byte("{" + string(encodeKey(path[i])) + ":" + string(v) + "}")
	}
	return v
}

// render is v indented by two spaces under a line indented by prefix (compact when !multi).
func render(v []byte, prefix string, multi bool) []byte {
	var b bytes.Buffer
	if !multi || json.Indent(&b, v, prefix, "  ") != nil {
		return v
	}
	return b.Bytes()
}

// lineIndent is the white space that starts the line holding doc[pos].
func lineIndent(doc []byte, pos int) string {
	start := bytes.LastIndexByte(doc[:pos], '\n') + 1
	end := start
	for end < len(doc) && (doc[end] == ' ' || doc[end] == '\t') {
		end++
	}
	return string(doc[start:end])
}

func splice(doc []byte, from, to int, with []byte) []byte {
	out := make([]byte, 0, len(doc)+len(with))
	out = append(out, doc[:from]...)
	out = append(out, with...)
	return append(out, doc[to:]...)
}

func encodeKey(k string) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(k)
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

func pathName(path []string) string {
	if len(path) == 0 {
		return "the document"
	}
	return fmt.Sprintf("%q", path)
}
