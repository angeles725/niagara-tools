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
	uses     []member // template.bog (when there is exactly one) and every *.px entry, in archive order
}

// member is an entry the ntpl-uses check reads: the modules it uses (modulesUsed), or why they are not
// known (its ntpl-uses message). Only the modules are kept: an entry's bytes are dropped once read.
type member struct {
	name string
	mods []string
	why  string
}

// readArchive opens path as a zip within the bounds of messages.go. An archive that is refused whole (too
// large, not a zip, too many entries, an unsafe entry name) has one entries message and nothing else: no
// entry of it is read. A non-nil error is a file that could not be opened or measured (the caller fails
// closed, exit 3); what the archive holds is never an error, it is a finding.
func readArchive(path string) (archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return archive{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return archive{}, err
	}
	if st.Size() > maxArchive {
		return refused(tooLarge, st.Size(), maxArchive), nil
	}
	return readZip(f, st.Size())
}

// openZip reads the directory of the zip in r within the bounds every archive shares (a .ntpl, a template
// module jar, a template inside that jar): rejected is the entries message of one refused whole, the
// directory otherwise. A non-nil error is r failing to read, never what it holds.
func openZip(r io.ReaderAt, size int64) (zr *zip.Reader, rejected string, err error) {
	zr, err = zip.NewReader(r, size)
	switch {
	case errors.Is(err, zip.ErrInsecurePath): // the reader is usable; unsafe() names the entry below
	case errors.Is(err, zip.ErrFormat) || errors.Is(err, zip.ErrAlgorithm) || errors.Is(err, zip.ErrChecksum) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF):
		return nil, fmt.Sprintf(notZip, err), nil
	case err != nil:
		return nil, "", err
	}
	if len(zr.File) > maxEntries {
		return nil, fmt.Sprintf(tooManyNames, len(zr.File), maxEntries), nil
	}
	for _, f := range zr.File {
		if unsafe(f.Name) {
			// Mutation: NTPL-unsafe -- reading on past an unsafe name judges an archive whose entries may land outside it when unpacked
			return nil, fmt.Sprintf(unsafeName, f.Name), nil
		}
	}
	return zr, "", nil
}

// readZip reads the template archive in r: its two entries, its manifest and what its bog and Px use.
func readZip(r io.ReaderAt, size int64) (archive, error) {
	zr, rejected, err := openZip(r, size)
	if err != nil || rejected != "" {
		return archive{entries: nonEmpty(rejected)}, err
	}
	var bogs, manifests, dotted, pxs []*zip.File
	for _, f := range zr.File {
		switch {
		case f.Name == bogEntry:
			bogs = append(bogs, f)
		case f.Name == manifestEntry:
			manifests = append(manifests, f)
		case f.Name == manifestAlt:
			dotted = append(dotted, f)
		case strings.HasSuffix(strings.ToLower(f.Name), pxSuffix):
			pxs = append(pxs, f)
		}
	}
	if len(manifests) == 0 {
		// Mutation: NTPL-dotted -- dropping the fallback reports "lacks template-manifest.xml" for an archive spelled as the guide spells it
		manifests = dotted // the guide's spelling (manifestAlt), read only when the shipped one is absent
	}
	var a archive
	a.need(bogEntry, len(bogs))
	a.need(manifestEntry, len(manifests))
	if len(manifests) == 1 {
		a.manifest, a.unread = inflate(manifests[0])
	}
	if len(bogs) == 1 {
		pxs = append([]*zip.File{bogs[0]}, pxs...)
	}
	a.uses = readUses(pxs)
	return a, nil
}

// nonEmpty is msg as a one-message list, nil when msg is "".
func nonEmpty(msg string) []string {
	if msg == "" {
		return nil
	}
	return []string{msg}
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
