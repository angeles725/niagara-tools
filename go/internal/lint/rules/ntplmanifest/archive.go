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
	manifest     []byte
	manifestName string   // the manifest entry read: manifestEntry, or manifestAlt when only it is present
	entries      []string // ntpl-entries messages
	unread       string   // an ntpl-manifest message, "" when manifest holds the bytes or the entry is absent
	uses         []member // template.bog (when there is exactly one) and every *.px entry, in archive order
}

// member is an entry the ntpl-uses check reads: the modules it uses (modulesUsed), or why they are not
// known (its ntpl-uses message). Only the modules are kept: an entry's bytes are dropped once read.
type member struct {
	name string
	mods []string
	why  string
}

// readArchive opens path as a zip within the bounds of messages.go, drawing what it inflates from b. An
// archive that is refused whole (too large, not a zip, too many entries, an unsafe entry name) has one
// entries message and nothing else: no entry of it is read. A non-nil error is a file that could not be
// opened or measured (the caller fails closed, exit 3); what the archive holds is never an error, it is a
// finding.
func readArchive(path string, b *budget) (archive, error) {
	f, size, refusedMsg, err := openBounded(path)
	if err != nil || refusedMsg != "" {
		return archive{entries: nonEmpty(refusedMsg)}, err
	}
	defer f.Close()
	return readZip(f, size, b)
}

// openBounded opens the archive at path when it is within maxArchive bytes on disk; refusedMsg is the
// entries message of one over it (f is then nil). The bound is read from the metadata before the file is
// opened: an archive over it is its row even when it cannot be opened. A non-nil error is a file that could
// not be opened or measured.
func openBounded(path string) (f *os.File, size int64, refusedMsg string, err error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, 0, "", err
	}
	if st.Size() > maxArchive {
		// Mutation: NTPL-size-first -- opening before the bound turns an unreadable archive over it into exit 3
		return nil, 0, fmt.Sprintf(tooLarge, st.Size(), maxArchive), nil
	}
	if f, err = os.Open(path); err != nil {
		return nil, 0, "", err
	}
	if st, err = f.Stat(); err != nil {
		f.Close()
		return nil, 0, "", err
	}
	if st.Size() > maxArchive { // the file grew between the two reads
		f.Close()
		return nil, 0, fmt.Sprintf(tooLarge, st.Size(), maxArchive), nil
	}
	return f, st.Size(), "", nil
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

// readZip reads the template archive in r: its two entries, its manifest and what its bog and Px use,
// drawing what it inflates from b. Once b is spent, nothing more of the archive is read.
func readZip(r io.ReaderAt, size int64, b *budget) (archive, error) {
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
	a := archive{manifestName: manifestEntry}
	if len(manifests) == 0 && len(dotted) > 0 {
		// Mutation: NTPL-dotted -- dropping the fallback reports "lacks template-manifest.xml" for an archive spelled as the guide spells it
		manifests, a.manifestName = dotted, manifestAlt // the guide's spelling, read only when the shipped one is absent
	}
	a.need(bogEntry, len(bogs))
	a.need(a.manifestName, len(manifests))
	if len(manifests) == 1 {
		a.manifest, a.unread = inflate(manifests[0], b)
	}
	if b.spent {
		return a, nil
	}
	if len(bogs) == 1 {
		pxs = append([]*zip.File{bogs[0]}, pxs...)
	}
	a.uses = readUses(pxs, b)
	return a, nil
}

// nonEmpty is msg as a one-message list, nil when msg is "".
func nonEmpty(msg string) []string {
	if msg == "" {
		return nil
	}
	return []string{msg}
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

// inflate reads the manifest entry up to maxManifest bytes, drawn from b; why is the ntpl-manifest message
// when it could not be read whole.
func inflate(f *zip.File, b *budget) (data []byte, why string) {
	data, res, err := b.inflate(f, maxManifest)
	switch res {
	case overLimit:
		return nil, fmt.Sprintf(manifestTooLarge, f.Name, maxManifest)
	case overBudget:
		return nil, spentMsg(f.Name)
	case unread:
		return nil, fmt.Sprintf(manifestUnread, f.Name, err)
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
