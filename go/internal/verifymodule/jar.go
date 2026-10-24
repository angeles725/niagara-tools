package verifymodule

import (
	"archive/zip"
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jarversion"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// jarFile is one jar as the bash preloaded it: LIST (unzip -Z1), MX (META-INF/module.xml) and TYPES (the
// class of every <type> element).
type jarFile struct {
	path  string
	zr    *zip.ReadCloser
	names []string
	list  grepText
	first map[string]*zip.File
	mx    string
	types []string
}

var (
	typeClassRe = rx(`<type [^>]*class="[^"]+"`)
	lastClassRe = regexp.MustCompile(`^.*class="([^"]+)"`)
)

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
		j.mx = subst(string(data))
		for _, m := range newGrepText(j.mx).only(typeClassRe) {
			j.types = append(j.types, lastClassRe.FindStringSubmatch(m)[1])
		}
	}
	return j, ""
}

func (j *jarFile) close() { j.zr.Close() }

// has is `grep -qx <name>` on LIST.
func (j *jarFile) has(name string) bool {
	_, ok := j.first[name]
	return ok
}

// text is entry name as `$(unzip -p jar name 2>/dev/null || true)` kept it: "" when it cannot be read.
func (j *jarFile) text(name string) string {
	data, err := jarversion.Read(j.first[name])
	if err != nil {
		return ""
	}
	return subst(string(data))
}
