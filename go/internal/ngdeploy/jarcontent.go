package ngdeploy

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
)

// digest stands for the bytes of one entry in the comparison.
type digest [sha256.Size]byte

// errOverLimit is a manifest or a descriptor larger than maxDescriptor.
var errOverLimit = errors.New("over the limit")

// sameContent reports whether two jars hold the same module: the same set of entry names and the same bytes
// for every entry, apart from what a build of unchanged sources changes anyway (patterns.go lists it): the
// signature files directly under META-INF/, the digests of the manifest's per-entry sections, and the
// buildMillis and buildHost attributes of the root element of META-INF/module.xml. why is set, and same is
// false, when a jar cannot be read in full or is over a limit of the comparison: that is never "the same
// content".
func sameContent(a, b string) (same bool, why string) {
	x, why := jarContent(a)
	if why != "" {
		return false, why
	}
	y, why := jarContent(b)
	if why != "" {
		return false, why
	}
	if len(x) != len(y) {
		return false, ""
	}
	for name, content := range x {
		if other, ok := y[name]; !ok || content != other {
			return false, ""
		}
	}
	return true, ""
}

// jarContent is the digest of every entry of a jar the comparison reads, by name. Memory stays bounded
// whatever the jar holds: an entry is hashed in chunks of contentChunk, never read whole; the manifest and
// the descriptor, which are compared in part, are read up to maxDescriptor; and a jar whose directory
// declares more than maxJarContent bytes of content is not read at all. why says what stopped the read: a
// file that is not a zip, a jar with no entry or with two entries under one name (which one ships is the
// reader's choice), a jar or a file over its limit, a descriptor with no root element, an entry that fails
// its size or CRC.
func jarContent(jar string) (content map[string]digest, why string) {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return nil, fmt.Sprintf(whyNotAJarFile, jar, err)
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return nil, fmt.Sprintf(whyNoEntry, jar)
	}
	var declared uint64
	for _, entry := range zr.File {
		if declared += entry.UncompressedSize64; declared > maxJarContent || entry.UncompressedSize64 > maxJarContent {
			return nil, fmt.Sprintf(whyJarTooLarge, jar, maxJarContent)
		}
	}
	content, chunk := map[string]digest{}, make([]byte, contentChunk)
	for _, entry := range zr.File {
		if _, twice := content[entry.Name]; twice {
			return nil, fmt.Sprintf(whyTwoEntries, jar, entry.Name)
		}
		if isSignatureFile(entry.Name) {
			content[entry.Name] = digest{} // its bytes are not compared; its name is, below
			continue
		}
		sum, err := entryDigest(entry, chunk)
		switch {
		case errors.Is(err, errOverLimit):
			return nil, fmt.Sprintf(whyFileTooLarge, jar, entry.Name, maxDescriptor)
		case errors.Is(err, errNoRoot):
			return nil, fmt.Sprintf(whyNoRoot, jar, entry.Name, err)
		case err != nil:
			return nil, fmt.Sprintf(whyEntryUnread, jar, entry.Name, err)
		}
		content[entry.Name] = sum
	}
	for name := range content {
		if isSignatureFile(name) {
			delete(content, name)
		}
	}
	return content, ""
}

// entryDigest is the digest of one entry: of its bytes, read chunk by chunk, or, for the manifest and the
// descriptor, of what is left of them once the parts a rebuild changes are taken out. archive/zip checks
// the size and the CRC of an entry when its last byte is read.
func entryDigest(entry *zip.File, chunk []byte) (sum digest, err error) {
	r, err := entry.Open()
	if err != nil {
		return sum, err
	}
	defer r.Close()
	if entry.Name != manifestName && entry.Name != moduleXML {
		h := sha256.New()
		if _, err := io.CopyBuffer(h, r, chunk); err != nil {
			return sum, err
		}
		return digest(h.Sum(nil)), nil
	}
	data, err := io.ReadAll(io.LimitReader(r, maxDescriptor+1))
	switch {
	case err != nil:
		return sum, err
	case len(data) > maxDescriptor:
		return sum, errOverLimit
	case entry.Name == manifestName:
		return sha256.Sum256(withoutDigests(data)), nil
	}
	if data, err = withoutBuildStamp(data); err != nil {
		return sum, err
	}
	return sha256.Sum256(data), nil
}

// sameBytes is cmp -s: both files can be read and hold the same bytes. They are read side by side, a chunk
// at a time.
func sameBytes(a, b string) bool {
	x, errX := os.Open(a)
	y, errY := os.Open(b)
	if x != nil {
		defer x.Close()
	}
	if y != nil {
		defer y.Close()
	}
	if errX != nil || errY != nil {
		return false
	}
	p, q := make([]byte, contentChunk), make([]byte, contentChunk)
	for {
		n, errP := io.ReadFull(x, p)
		m, errQ := io.ReadFull(y, q)
		if n != m || !bytes.Equal(p[:n], q[:m]) {
			return false
		}
		ended := func(err error) bool { return err == io.EOF || err == io.ErrUnexpectedEOF }
		switch {
		case ended(errP) && ended(errQ):
			return true
		case errP != nil || errQ != nil:
			return false
		}
	}
}
