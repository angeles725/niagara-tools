// Package jarlist is the test side of the stored-repack goldens: Build spells an input jar from a text spec,
// and List prints what a repacked jar must keep, in a form the bash recording and the Go replay both produce.
//
// The golden contract is semantic, not byte identity: zip(1) and archive/zip write different timestamps,
// attributes and header layouts, so a listing holds what a consumer of the jar observes: the entry names,
// their order, each entry's method and size and a hash of its bytes, and whether its local header defers
// the sizes to a data descriptor.
package jarlist

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// Build is the zip a spec spells: one "--- <entry> (stored|deflated)[ (noeol)|(base64)] ---" header line per
// entry, in archive order, its body the entry's bytes (the "(jar)" sub-section spelling of the shared golden
// format). An entry without a method marker is deflated; a name ending in "/" is a directory entry.
func Build(spec []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var name, body string
	open := false
	flush := func() error {
		if !open {
			return nil
		}
		method, entry := uint16(zip.Deflate), name
		data := body
		for {
			switch {
			case strings.HasSuffix(entry, " (stored)"):
				method, entry = zip.Store, strings.TrimSuffix(entry, " (stored)")
				continue
			case strings.HasSuffix(entry, " (deflated)"):
				method, entry = zip.Deflate, strings.TrimSuffix(entry, " (deflated)")
				continue
			case strings.HasSuffix(entry, " (noeol)"):
				entry, data = strings.TrimSuffix(entry, " (noeol)"), strings.TrimSuffix(data, "\n")
				continue
			case strings.HasSuffix(entry, " (base64)"):
				entry = strings.TrimSuffix(entry, " (base64)")
				b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data))
				if err != nil {
					return fmt.Errorf("%s: %w", entry, err)
				}
				data = string(b)
				continue
			}
			break
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: entry, Method: method})
		if err == nil {
			_, err = io.WriteString(w, data)
		}
		return err
	}
	for _, ln := range strings.SplitAfter(string(spec), "\n") {
		if strings.HasPrefix(ln, "--- ") && strings.HasSuffix(ln, " ---\n") {
			if err := flush(); err != nil {
				return nil, err
			}
			name, body, open = strings.TrimSuffix(strings.TrimPrefix(ln, "--- "), " ---\n"), "", true
		} else if open {
			body += ln
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	err := zw.Close()
	return buf.Bytes(), err
}

// List describes the zip at path: an "entries <n>" line, then one line per entry,
// "<part> <name> <method> size=<n> sha256=<16 hex>[ +descriptor]". The "head" part is the leading run of
// META-INF/MANIFEST.MF, META-INF/<ALIAS>.SF and the .RSA/.DSA/.EC blocks of an .SF the jar holds, in
// archive order; every entry after it is "rest", sorted by name, because the bash repack wrote those in
// the directory-walk order of its extraction dir (a filesystem property, not a contract).
func List(file string) (string, error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	lines := make([]string, len(zr.File))
	inHead := true
	head := 0
	for i, f := range zr.File {
		desc, err := describe(f)
		if err != nil {
			return "", err
		}
		if inHead && signing(f.Name, names) {
			head++
			lines[i] = "head " + desc
			continue
		}
		inHead = false
		lines[i] = "rest " + desc
	}
	rest := lines[head:]
	sort.Slice(rest, func(a, b int) bool { return rest[a] < rest[b] })
	return fmt.Sprintf("entries %d\n%s\n", len(zr.File), strings.Join(lines, "\n")), nil
}

// signing reports whether name belongs to the head of a repacked jar.
func signing(name string, names map[string]bool) bool {
	if name == "META-INF/MANIFEST.MF" {
		return true
	}
	dir, base := path.Split(name)
	i := strings.LastIndex(base, ".")
	if dir != "META-INF/" || i < 0 || strings.HasPrefix(base, ".") {
		return false // bash's META-INF/*.SF glob matches neither a subdirectory nor a dot-name
	}
	switch stem, ext := base[:i], base[i+1:]; ext {
	case "SF":
		return true
	case "RSA", "DSA", "EC":
		return names["META-INF/"+stem+".SF"]
	}
	return false
}

// describe is one entry's listing text after its part.
func describe(f *zip.File) (string, error) {
	method := map[uint16]string{zip.Store: "stored", zip.Deflate: "deflated"}[f.Method]
	if method == "" {
		method = fmt.Sprintf("method-%d", f.Method)
	}
	h := sha256.New()
	rc, err := f.Open()
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.Name, err)
	}
	n, err := io.Copy(h, rc)
	if cerr := rc.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.Name, err)
	}
	s := fmt.Sprintf("%s %s size=%d sha256=%x", f.Name, method, n, h.Sum(nil)[:8])
	if f.Flags&0x8 != 0 {
		s += " +descriptor"
	}
	return s, nil
}
