package jarversion

import (
	"archive/zip"
	"compress/bzip2"
	"errors"
	"io"
	"strings"
	"syscall"
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
// err is the first reason the jar could not be checked in full: it cannot be opened as a zip, or a
// sampled class cannot be read (encrypted, an unsupported method, data shorter than its header says).
// The sample goes on past an unreadable class, so a finding and an error can both be returned.
// The bash tool ran `unzip -Z1` and `unzip -p` with 2>/dev/null and counted all of these as clean.
func CheckJar(path string) (found Finding, hit bool, err error) {
	r, err := zip.OpenReader(path)
	// ErrInsecurePath (a "../" or absolute entry name, under GODEBUG zipinsecurepath=0) still returns a
	// usable reader; unzip -Z1 listed such names, so they are sampled too.
	if err != nil && !(errors.Is(err, zip.ErrInsecurePath) && r != nil) {
		return Finding{}, false, err
	}
	err = nil
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
		h, herr := head(first[name])
		if herr != nil {
			if err == nil {
				err = &entryError{name, herr}
			}
			continue
		}
		if major := Major(h); AboveJava8(major) {
			return Finding{name, major}, true, err
		}
	}
	return Finding{}, false, err
}

// errEncrypted is an entry unzip cannot read without a password.
var errEncrypted = errors.New("encrypted entry")

// head returns the first 8 bytes of entry f, or all of it when it is shorter.
func head(f *zip.File) ([]byte, error) {
	if f.Flags&flagEncrypted != 0 {
		return nil, errEncrypted
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	n := uint64(8)
	if f.UncompressedSize64 < n {
		n = f.UncompressedSize64
	}
	buf := make([]byte, n)
	// Read exactly what the header promises: a short read is a damaged entry, not a short class.
	if _, err := io.ReadFull(rc, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// entryError is a sampled class entry that could not be read.
type entryError struct {
	name string
	err  error
}

func (e *entryError) Error() string { return "cannot read " + e.name + ": " + e.err.Error() }
func (e *entryError) Unwrap() error { return e.err }

// Reason is err as the toolbelt's "cannot scan" and find messages word it: the C strerror text of an
// errno, else the error text without its package prefix ("zip: ", "flate: "); an unreadable entry is
// "cannot read <entry>: <reason>".
func Reason(err error) string {
	var ee *entryError
	if errors.As(err, &ee) {
		return "cannot read " + ee.name + ": " + Reason(ee.err)
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return strerror(errno)
	}
	msg := err.Error()
	for _, pkg := range []string{"zip: ", "flate: ", "bzip2 data invalid: "} {
		if strings.HasPrefix(msg, pkg) {
			return strings.TrimPrefix(msg, pkg)
		}
	}
	return msg
}
