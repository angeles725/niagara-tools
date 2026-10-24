// Package jarlist is the test side of the stored-repack goldens: List prints what a repacked jar must keep, in a
// form the bash recording and the Go replay both produce. The input jars are spelled in the "(jar)" sub-section
// format of the shared golden cases and built by golden.BuildJar.
//
// The golden contract is semantic, not byte identity: zip(1) and archive/zip write different timestamps,
// attributes and header layouts, so a listing holds what a consumer of the jar observes: the entry names,
// their order, each entry's method and size and a hash of its bytes, and whether its local header defers
// the sizes to a data descriptor.
package jarlist

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

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
