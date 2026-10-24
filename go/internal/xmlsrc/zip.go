package xmlsrc

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
)

// zipMagic starts a ZIP local file header: a saved station config.bog is file.xml zipped.
var zipMagic = []byte("PK\x03\x04")

// maxMember caps the inflated file.xml so a hostile archive cannot exhaust memory.
var maxMember = 1 << 30 // a var so the cap is testable

// readSource reads path once. A ZIP (detected by content, not extension) yields its file.xml
// member; anything else is returned as read. Every failure is an error naming path.
func readSource(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(data, zipMagic) {
		return data, err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%s: corrupt zip archive: %w", path, err)
	}
	for _, f := range zr.File {
		if f.Name != "file.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: cannot open file.xml: %w", path, err)
		}
		defer rc.Close()
		xml, err := io.ReadAll(io.LimitReader(rc, int64(maxMember)+1))
		if err != nil {
			return nil, fmt.Errorf("%s: cannot inflate file.xml: %w", path, err)
		}
		if len(xml) > maxMember {
			return nil, fmt.Errorf("%s: file.xml inflates past %d bytes", path, maxMember)
		}
		return xml, nil
	}
	return nil, fmt.Errorf("%s: zip archive has no file.xml member", path)
}
