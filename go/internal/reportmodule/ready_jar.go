package reportmodule

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// What report-module --ready reads from a jar (its META-INF/module.xml and its bytes) and the mirror row.

// maxModuleXML bounds the module.xml read from a jar.
const maxModuleXML = 8 << 20

var (
	// partName is the module part name on the <module ...> root tag (name=, not moduleName=).
	partName = regexp.MustCompile(`<module\s(?:[^>]*\s)?name="([^"]*)"`)
	// bajaFloor is the vendorVersion of the baja dependency: the Niagara the module requires at least.
	bajaFloor = regexp.MustCompile(`<dependency\s[^>]*name="baja"[^>]*vendorVersion="([^"]*)"`)
)

// jarInfo is what the ready rows need from a jar; why says what could not be read ("" when all of it was).
type jarInfo struct {
	name, version, bajaFloor, sum, why string
}

// readJar reads the module part, its version and its baja floor from META-INF/module.xml, and the jar's sha256.
func readJar(jar string) jarInfo {
	var info jarInfo
	if data, err := os.ReadFile(jar); err == nil {
		s := sha256.Sum256(data)
		info.sum = hex.EncodeToString(s[:])
	}
	xml, why := moduleXMLOf(jar)
	if why != "" {
		info.why = why
		return info
	}
	if m := partName.FindStringSubmatch(xml); m != nil {
		info.name = m[1]
	}
	if m := bajaFloor.FindStringSubmatch(xml); m != nil {
		info.bajaFloor = m[1]
	}
	info.version = vendorversion.ModuleXMLVersion(xml)
	if info.name == "" || info.version == "" {
		info.why = "META-INF/module.xml names no module part or no vendorVersion on its <module> tag"
	}
	return info
}

// moduleXMLOf is the META-INF/module.xml text of a jar, or why it cannot be read.
func moduleXMLOf(jar string) (xml, why string) {
	z, err := zip.OpenReader(jar)
	if err != nil {
		return "", "cannot read the jar: " + strerror.Text(err)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name != "META-INF/module.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", "cannot read META-INF/module.xml: " + strerror.Text(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, maxModuleXML))
		if err != nil {
			return "", "cannot read META-INF/module.xml: " + strerror.Text(err)
		}
		return string(data), ""
	}
	return "", "no META-INF/module.xml in the jar"
}

// mirrorRow says whether the mirror's modules/<jar> is the deployed jar again. A live first-replaced pointer
// (build.sh keeps the jar a build replaced) names the deployed jar: modules/ equal to it PASSes, anything else
// WARNs with the --restore-mirror command. With no pointer, modules/ holding this very build WARNs. It never
// FAILs: the mirror is the next build's baseline, not what the station loads.
func mirrorRow(tb, home, jar, sum string) (string, string, string) {
	const check = "mirror"
	if home == "" {
		return skip, check, "no --niagara-home: no mirror to check"
	}
	if !modjar.IsMirror(home) {
		return skip, check, home + " is not a mirror (" + "no " + modjar.MirrorMarker + "): nothing to restore"
	}
	name := filepath.Base(jar)
	rel := "modules/" + name
	cur := fileSum12(home + "/" + rel)
	if cur == "" {
		return warn, check, rel + " is missing or unreadable: the mirror holds no deployed copy"
	}
	dir := home + "/" + ReplacedDir
	if first, ok := LiveFirst(dir, name); ok {
		if cur == first {
			return pass, check, rel + " is the deployed jar again (sha256 " + first + ")"
		}
		return warn, check, rel + " (sha256 " + cur + ") is not the deployed jar (sha256 " + first + "); restore with: " +
			ShellWord(tb+"/build.sh") + " --restore-mirror " + ShellWord(dir+"/"+name+"."+first)
	}
	if cur == short(sum) {
		return warn, check, rel + " is this build (sha256 " + cur + "), not the deployed jar, and no first-replaced jar is" +
			" recorded under " + dir + ": put the deployed jar back before the next build"
	}
	return pass, check, rel + " (sha256 " + cur + ") is not this build: nothing to restore"
}

// fileSum12 is the first 12 hex digits of a file's sha256, "" when it cannot be read.
func fileSum12(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])[:12]
}
