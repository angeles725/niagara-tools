package n4probe

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// absent is what a field that is not in the document renders as.
const absent = "<absent>"

// stringList is a repeatable flag.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

// decode reads a JSON document; numbers keep the text the station wrote (1.50 stays 1.50).
func decode(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// render is the JSON text of the value at the dot path (object keys, array indexes: a.b.0.c), or absent.
func render(doc any, path string) string {
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[seg]
			if !ok {
				return absent
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(v) {
				return absent
			}
			cur = v[i]
		default:
			return absent
		}
	}
	out, err := json.Marshal(cur)
	if err != nil {
		return absent
	}
	return string(out)
}
