package buildmodule

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// mirrorBuiltFile is the record of the jars this checkout installed into a mirror, relative to the module root:
// one line per jar, "<jar file name> <sha256 of the jar bytes>". It lives in the checkout, never in modules/
// (Niagara reads that directory), and should be gitignored. What the mirror holds is a build, not a release; the
// record is how the drift gate tells the checkout's own earlier build from a jar somebody released.
const mirrorBuiltFile = ".build/mirror-built"

// fileSHA256 is the hex sha256 of the bytes of the file at path, "" when it cannot be read.
func fileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readMirrorBuilt is the record as jar name -> sha256; a missing or unreadable file is an empty record and a
// line that is not "<name> <hex sha256>" is skipped, so a damaged record can only make the drift gate stricter.
func (b *build) readMirrorBuilt() map[string]string {
	data, _ := os.ReadFile(b.root + "/" + mirrorBuiltFile)
	record := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if f := strings.Fields(line); len(f) == 2 && len(f[1]) == sha256.Size*2 && f[1] == strings.ToLower(f[1]) {
			record[f[0]] = f[1]
		}
	}
	return record
}

// ownBuild says whether the jar the mirror held before this build (the snapshot) is a build of this checkout:
// its sha256 is the one the record holds for its name.
func (b *build) ownBuild(profile string, snap snapshot) bool {
	want, ok := b.readMirrorBuilt()[b.jarName(profile)+".jar"]
	return ok && want == fileSHA256(snap.backup)
}

// recordMirrorBuild writes the record once the gate passed: the jars of this run that gradle installed into a
// mirror (a modules/ jar byte-identical to the one just built), added to the lines of jars this run did not
// build. Nothing is written for a home that is no mirror, or when no jar was installed. A record that cannot be
// written is a WARN: the build is fine, only the next rebuild of this version reads as drift.
func (b *build) recordMirrorBuild() {
	if !b.isMirror() {
		return
	}
	record, installed := b.readMirrorBuilt(), false
	for _, p := range b.profiles {
		jar := b.deployedJar(p)
		if lint.IsFile(jar) && sameContent(jar, b.builtJar(p)) {
			record[b.jarName(p)+".jar"], installed = fileSHA256(jar), true
		}
	}
	if !installed {
		return
	}
	names := make([]string, 0, len(record))
	for name := range record {
		names = append(names, name)
	}
	sort.Strings(names)
	var text strings.Builder
	for _, name := range names {
		text.WriteString(name + " " + record[name] + "\n")
	}
	path := b.root + "/" + mirrorBuiltFile
	err := os.MkdirAll(b.root+"/.build", 0o755)
	if err == nil {
		err = atomicfile.Write(path, []byte(text.String()))
	}
	if err != nil {
		b.warn(msgMirrorBuiltFailed, path, err)
	}
}
