package writeops

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
)

// The entries a component template (.ntpl, a zip) holds that a deploy reads: its manifest (the
// Templates guide spells it template.manifest.xml; the platform writes template-manifest.xml)
// and the bog of its component tree. Px files and images stay with Workbench.
const (
	tplManifestEntry = "template-manifest.xml"
	tplManifestAlt   = "template.manifest.xml"
	tplBogEntry      = "template.bog"
	// tplMaxArchive bounds the .ntpl on disk and tplMaxEntry each entry inflated (the bounds
	// of the kit's ntpl-manifest lint): a larger file is refused before it is read.
	tplMaxArchive = 32 << 20
	tplMaxEntry   = 32 << 20
)

// tplArchive is the two entries of a template a deploy reads.
type tplArchive struct {
	manifest, bog []byte
}

// readTemplateArchive opens path as a .ntpl within the bounds; every failure is a refusal that
// names the file.
func readTemplateArchive(path string) (*tplArchive, error) {
	refuse := func(why string) error { return plan.ToolError("template_file " + path + ": " + why) }
	if !strings.HasSuffix(strings.ToLower(path), ".ntpl") {
		return nil, refuse("not a component template (.ntpl); an application template (.napl) " +
			"is installed with Workbench")
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, refuse("cannot be read (" + osWords(err) + ")")
	}
	if st.Size() > tplMaxArchive {
		return nil, refuse(fmt.Sprintf("%d bytes, over the %d-byte bound", st.Size(), tplMaxArchive))
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, refuse("not a zip archive (" + err.Error() + ")")
	}
	defer z.Close()
	out := &tplArchive{}
	for _, f := range z.File {
		var dst *[]byte
		switch {
		case f.Name == tplBogEntry:
			dst = &out.bog
		case f.Name == tplManifestEntry, f.Name == tplManifestAlt && out.manifest == nil:
			dst = &out.manifest
		default:
			continue
		}
		data, err := readEntry(f)
		if err != nil {
			return nil, refuse(f.Name + ": " + err.Error())
		}
		*dst = data
	}
	if out.manifest == nil {
		return nil, refuse("no " + tplManifestEntry + " entry: not a template")
	}
	if out.bog == nil {
		return nil, refuse("no " + tplBogEntry + " entry: the template holds no component tree")
	}
	return out, nil
}

// readEntry inflates f, refusing one larger than tplMaxEntry.
func readEntry(f *zip.File) ([]byte, error) {
	if f.UncompressedSize64 > tplMaxEntry {
		return nil, fmt.Errorf("%d bytes inflated, over the %d-byte bound", f.UncompressedSize64, tplMaxEntry)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, tplMaxEntry+1))
	if err != nil {
		return nil, err
	}
	if len(data) > tplMaxEntry {
		return nil, fmt.Errorf("over the %d-byte bound inflated", tplMaxEntry)
	}
	return data, nil
}

// osWords is an os error without the path it repeats.
func osWords(err error) string {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err.Error()
	}
	return err.Error()
}
