package jarversion

import (
	"archive/zip"
	"compress/bzip2"
	"errors"
	"io"
)

// Finding is the first sampled class entry of a jar whose major version is above Java 8.
type Finding struct {
	Entry string
	Major int
}

// methodBzip2 is zip compression method 12, which unzip reads and archive/zip does not register.
const methodBzip2 = 12

// flagEncrypted is zip general-purpose bit 0: unzip cannot read such an entry without a password.
const flagEncrypted = 0x1

// CheckJar reads the sampled class entries of the jar at path and returns the first one above Java 8.
// A jar that cannot be opened as a zip, and an entry that cannot be read, count as clean: the bash tool
// ran `unzip -Z1` and `unzip -p` with 2>/dev/null, so neither was ever an error.
func CheckJar(path string) (Finding, bool) {
	r, err := zip.OpenReader(path)
	// ErrInsecurePath (a "../" or absolute entry name, under GODEBUG zipinsecurepath=0) still returns a
	// usable reader; unzip -Z1 listed such names, so they are sampled too.
	if err != nil && !(errors.Is(err, zip.ErrInsecurePath) && r != nil) {
		return Finding{}, false
	}
	defer r.Close()
	r.RegisterDecompressor(methodBzip2, func(rd io.Reader) io.ReadCloser { return io.NopCloser(bzip2.NewReader(rd)) })

	// unzip -p <name> reads the first record of a duplicated name, so index the first one only.
	first := make(map[string]*zip.File, len(r.File))
	names := make([]string, len(r.File))
	for i, f := range r.File {
		names[i] = f.Name
		if _, seen := first[f.Name]; !seen {
			first[f.Name] = f
		}
	}
	for _, name := range Sample(names) {
		if major := Major(head(first[name])); AboveJava8(major) {
			return Finding{name, major}, true
		}
	}
	return Finding{}, false
}

// head returns up to the first 8 bytes of entry f; nil when it is encrypted or cannot be opened.
func head(f *zip.File) []byte {
	if f.Flags&flagEncrypted != 0 {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	buf := make([]byte, 8)
	n, _ := io.ReadFull(rc, buf)
	return buf[:n]
}
