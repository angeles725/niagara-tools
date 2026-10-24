package verifymodule

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jarversion"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// jarFile is one jar as the bash preloaded it: LIST (unzip -Z1) and MX (META-INF/module.xml), the latter
// parsed by xmlsrc (doc), with the class of every <type> element that names one (types).
type jarFile struct {
	path  string
	zr    *zip.ReadCloser
	names []string
	list  grepText
	first map[string]*zip.File
	mx    string
	doc   *xmlsrc.Doc // nil when mx is empty
	types []string
}

// openJar preloads path, or returns the line the bash printed before it exited 3.
func openJar(path string) (*jarFile, string) {
	if !lint.IsFile(path) {
		return nil, "jar not readable: " + path
	}
	zr, err := jarversion.Open(path)
	if err != nil || len(zr.File) == 0 { // unzip -Z1 fails on an empty zip too
		if zr != nil {
			zr.Close()
		}
		return nil, "not a zip: " + path
	}
	j := &jarFile{path: path, zr: zr}
	j.names, j.first = jarversion.Index(zr)
	j.list = newGrepText(strings.Join(j.names, "\n"))
	if j.has("META-INF/module.xml") {
		data, err := jarversion.Read(j.first["META-INF/module.xml"])
		if err != nil {
			zr.Close()
			return nil, fmt.Sprintf("cannot read META-INF/module.xml in %s: %s", path, jarversion.Reason(err))
		}
		if j.mx = subst(string(data)); j.mx != "" {
			// A malformed module.xml is an env error with its position, never the tags a line grep finds.
			if j.doc, err = xmlsrc.Parse(path+"!META-INF/module.xml", data); err != nil {
				zr.Close()
				return nil, err.Error()
			}
			for _, i := range j.doc.Find("type") {
				if cls, _ := j.doc.Elements()[i].Attr("class"); cls != "" {
					j.types = append(j.types, cls)
				}
			}
		}
	}
	return j, ""
}

func (j *jarFile) close() { j.zr.Close() }

// typeCount is the number of <type> elements module.xml declares (0 without one).
func (j *jarFile) typeCount() int {
	if j.doc == nil {
		return 0
	}
	return len(j.doc.Find("type"))
}

// has is `grep -qx <name>` on LIST.
func (j *jarFile) has(name string) bool {
	_, ok := j.first[name]
	return ok
}

// text is entry name as `$(unzip -p jar name)` keeps it ("" for an absent entry), or the error that
// stopped reading it (a damaged or encrypted entry).
func (j *jarFile) text(name string) (string, error) {
	f, ok := j.first[name]
	if !ok {
		return "", nil
	}
	data, err := jarversion.Read(f)
	if err != nil {
		return "", err
	}
	return subst(string(data)), nil
}
