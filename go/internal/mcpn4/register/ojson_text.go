package register

import (
	"bytes"
	"encoding/json"
	"fmt"
)

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

// eol gives the inserted text the file's line ending: CRLF when the file uses it.
func eol(text []byte, crlf bool) []byte {
	if !crlf {
		return text
	}
	return bytes.ReplaceAll(text, []byte("\n"), []byte("\r\n"))
}
