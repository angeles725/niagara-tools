// Package journal holds the append-only state files of the write tools
// (mcp-n4-kit/mcp_n4/safety.py:156-311): the JSONL file both share, the write-ahead Journal
// with its merged batch views, and the AuditLog. The state dir rules (CheckStateDir,
// LooseFileReason) are package safety's. A file this package creates is private from the start;
// an existing one is never chmodded.
package journal

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// PermissionError is the PermissionError an append raises on an existing file that is not
// private: it is never written to and never chmodded.
type PermissionError struct{ Msg string }

func (e *PermissionError) Error() string { return e.Msg }

// PyType names the Python exception.
func (e *PermissionError) PyType() string { return "PermissionError" }

// File is _JsonlFile: one JSON object per line in StateDir. Making one touches no file.
type File struct{ StateDir, Path string }

func newFile(stateDir, name string) File {
	return File{StateDir: stateDir, Path: safety.Join(stateDir, name)}
}

func looseText(path, reason string) string {
	return path + " " + reason + ": run `chmod 600 " + path + "` yourself (the server never changes " +
		"the permissions of an existing file)"
}

// CheckPrivate is check_private: a *safety.Error when the file exists and is not private.
func (f File) CheckPrivate() error {
	info, err := os.Stat(f.Path)
	if err != nil {
		return nil // missing: an append creates it private
	}
	if reason := safety.LooseFileReason(info); reason != "" {
		return &safety.Error{Msg: looseText(f.Path, reason)}
	}
	return nil
}

// Append writes json.dumps(entry) and a newline in one write. The state dir is created 0700
// when missing; a new file is created 0600 (O_EXCL), an existing one must already be private.
func (f File) Append(entry *pyjson.Object) error {
	if err := safety.PrivateDir(f.StateDir); err != nil {
		return err
	}
	fh, err := os.OpenFile(f.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		if fh, err = os.OpenFile(f.Path, os.O_WRONLY|os.O_APPEND, 0); err == nil {
			info, serr := fh.Stat()
			if serr != nil {
				fh.Close()
				return serr
			}
			if reason := safety.LooseFileReason(info); reason != "" {
				fh.Close()
				return &PermissionError{looseText(f.Path, reason)}
			}
		}
	}
	if err != nil {
		return err
	}
	defer fh.Close()
	line, err := pyjson.Marshal(entry, pyjson.Options{})
	if err != nil {
		return err
	}
	_, err = fh.Write(append(line, '\n'))
	return err
}

// line is one line of lines: its 1-based number and its JSON object, nil when it is not UTF-8.
type line struct {
	n     int
	entry *pyjson.Object
}

// lines is _JsonlFile._lines: the file split as bytes.splitlines splits it (LF, CRLF and CR,
// the line ends readlines() honours) and decoded line by line, so one bad byte costs one line,
// not the file (issues #448, #466). A missing file or a torn line is skipped.
func (f File) lines() []line {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return nil
	}
	var out []line
	for n := 1; len(data) > 0; n++ {
		raw, end := data, len(data)
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			raw, end = data[:i], i+1
			if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
				end++
			}
		}
		data = data[end:]
		if !utf8.Valid(raw) { // strict UTF-8, as bytes.decode("utf-8") (surrogates refused)
			out = append(out, line{n, nil})
			continue
		}
		if v, err := pyjson.Loads(raw); err == nil {
			if o, ok := v.(*pyjson.Object); ok {
				out = append(out, line{n, o})
			}
		}
	}
	return out
}

// AuditLog is the append-only record of every write call: planned, executed or refused.
type AuditLog struct{ File }

// NewAuditLog is AuditLog(state_dir): <state_dir>/audit.jsonl.
func NewAuditLog(stateDir string) *AuditLog { return &AuditLog{newFile(stateDir, "audit.jsonl")} }

// Scan is AuditLog.scan: every JSON object line, and each line that is not UTF-8 as an
// Unreadable (Reason "UnicodeDecodeError"), left out of the entries (issue #466).
func (a *AuditLog) Scan() ([]*pyjson.Object, []Unreadable) {
	var entries []*pyjson.Object
	var bad []Unreadable
	for _, l := range a.lines() {
		if l.entry == nil {
			bad = append(bad, Unreadable{Line: l.n, Reason: "UnicodeDecodeError"})
			continue
		}
		entries = append(entries, l.entry)
	}
	return entries, bad
}
