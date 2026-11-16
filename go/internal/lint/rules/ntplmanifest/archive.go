package ntplmanifest

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// archive is what one .ntpl read gives: the manifest bytes when it could be read, and why the archive or
// its entries are refused (the messages of ntpl-entries and ntpl-manifest rows, in report order).
type archive struct {
	manifest []byte
	entries  []string // ntpl-entries messages
	unread   string   // an ntpl-manifest message, "" when manifest holds the bytes or the entry is absent
}

// readArchive opens path as a zip within the bounds of messages.go. An archive that is refused whole (too
// large, not a zip, too many entries, an unsafe entry name) has one entries message and nothing else: no
// entry of it is read. A non-nil error is a file that could not be opened or measured (the caller fails
// closed, exit 3); what the archive holds is never an error, it is a finding.
func readArchive(path string) (archive, error) {
	st, err := os.Stat(path)
	if err != nil {
		return archive{}, err
	}
	if st.Size() > maxArchive {
		return refused(tooLarge, st.Size(), maxArchive), nil
	}
	zr, err := zip.OpenReader(path)
	switch {
	case errors.Is(err, zip.ErrInsecurePath): // the reader is usable; unsafe() names the entry below
	case errors.Is(err, zip.ErrFormat) || errors.Is(err, zip.ErrAlgorithm) || errors.Is(err, zip.ErrChecksum) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF):
		return refused(notZip, err), nil
	case err != nil:
		return archive{}, err
	}
	defer zr.Close()
	if len(zr.File) > maxEntries {
		return refused(tooManyNames, len(zr.File), maxEntries), nil
	}
	var bogs, manifests []*zip.File
	for _, f := range zr.File {
		switch {
		case unsafe(f.Name):
			// Mutation: NTPL-unsafe -- reading on past an unsafe name judges an archive whose entries may land outside it when unpacked
			return refused(unsafeName, f.Name), nil
		case f.Name == bogEntry:
			bogs = append(bogs, f)
		case f.Name == manifestEntry:
			manifests = append(manifests, f)
		}
	}
	var a archive
	a.need(bogEntry, len(bogs))
	a.need(manifestEntry, len(manifests))
	if len(manifests) == 1 {
		a.manifest, a.unread = inflate(manifests[0])
	}
	return a, nil
}

func refused(format string, args ...any) archive {
	return archive{entries: []string{fmt.Sprintf(format, args...)}}
}

// need records an entry that is absent or present more than once.
func (a *archive) need(name string, n int) {
	switch {
	case n == 0:
		// Mutation: NTPL1 -- dropping this row passes a .ntpl with no template.bog or no manifest
		a.entries = append(a.entries, fmt.Sprintf(lacksEntry, name))
	case n > 1:
		a.entries = append(a.entries, fmt.Sprintf(repeatedEntry, n, name))
	}
}

// inflate reads the manifest entry up to maxManifest bytes; why is the ntpl-manifest message when it could
// not be read whole.
func inflate(f *zip.File) (data []byte, why string) {
	if f.UncompressedSize64 > maxManifest {
		return nil, fmt.Sprintf(manifestTooLarge, maxManifest)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Sprintf(manifestUnread, err)
	}
	defer rc.Close()
	data, err = io.ReadAll(io.LimitReader(rc, maxManifest+1))
	switch {
	case err != nil:
		return nil, fmt.Sprintf(manifestUnread, err)
	case len(data) > maxManifest:
		return nil, fmt.Sprintf(manifestTooLarge, maxManifest)
	}
	return data, ""
}

// unsafe reports an entry name that could leave the directory an archive is unpacked into, or that a
// terminal or a row cannot print as it is: empty, absolute, holding a ".." element, a backslash, a drive
// letter or a control character. The name is matched as data, never handed to a tool.
func unsafe(name string) bool {
	if name == "" || name[0] == '/' || strings.ContainsRune(name, '\\') || (len(name) > 1 && name[1] == ':') {
		return true
	}
	for i := 0; i < len(name); i++ {
		if name[i] < 0x20 || name[i] == 0x7f {
			return true
		}
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
