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
	seen := map[string]bool{}
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
		if seen[m.key] {
			// JSON.parse keeps the last one; which one a harness reads is not ours to guess.
			return o, fmt.Errorf("duplicate key %q", m.key)
		}
		seen[m.key] = true
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

// checkDocument refuses anything but one JSON object (white space around it allowed): trailing
// data or a second document would be kept, unread, by a byte splice.
func checkDocument(doc []byte) error {
	if len(bytes.TrimSpace(doc)) == 0 {
		return nil
	}
	if !json.Valid(doc) {
		var v any
		err := json.Unmarshal(doc, &v)
		return fmt.Errorf("not one valid JSON document: %v", err)
	}
	_, err := scanObject(doc, 0)
	return err
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
	if err := checkDocument(doc); err != nil {
		return nil, err
	}
	multi := bytes.Contains(bytes.TrimSpace(doc), []byte("\n"))
	crlf := bytes.Contains(doc, []byte("\r\n"))
	at := 0
	for i, k := range path {
		o, err := scanObject(doc, at)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pathName(path[:i]), err)
		}
		m := o.find(k)
		if m == nil {
			return insert(doc, o, k, nest(path[i+1:], v), multi, crlf), nil
		}
		if i == len(path)-1 {
			return splice(doc, m.valStart, m.valEnd, eol(render(v, lineIndent(doc, m.keyEnd), multi), crlf)), nil
		}
		at = m.valStart
	}
	return doc, nil
}

// insert adds "key": v as the last member of o, in the layout of its last member (or, for an
// empty object, one level deeper than the line of its opening brace).
func insert(doc []byte, o objSpan, key string, v []byte, multi, crlf bool) []byte {
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
		return splice(doc, last.valEnd, last.valEnd, eol([]byte(",\n"+ind+k+": "+string(render(v, ind, true))), crlf))
	}
	ind := lineIndent(doc, o.open)
	inner := ind + "  "
	return splice(doc, o.open+1, o.close, eol([]byte("\n"+inner+k+": "+string(render(v, inner, true))+"\n"+ind), crlf))
}
