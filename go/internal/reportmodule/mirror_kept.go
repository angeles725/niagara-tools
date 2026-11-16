package reportmodule

import (
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// The jars a mirror keeps when a build replaces its modules/ copy (build.sh, go/internal/buildmodule
// replaced.go) and the pointer to the first one of a cycle. build.sh writes them; report-module --ready reads
// them to say whether the mirror holds the deployed jar again.

// ReplacedDir is where a mirror keeps the modules/ jars its builds replaced, named <MOD>-<profile>.jar.<sha12>
// after the first 12 hex digits of the replaced jar's sha256.
const ReplacedDir = ".mirror-replaced"

// FirstSuffix names the pointer to the first jar a cycle replaced: <name>.first beside the kept jars.
const FirstSuffix = ".first"

// KeptName is the name of a kept jar: the modules/ file name, then its hash.
var KeptName = regexp.MustCompile(`^(.+\.jar)\.([0-9a-f]{12})$`)

// LiveFirst is the hash dir/<name>.first names when it is live: it names a kept jar dir/<name>.<sha12> that is
// a file. ok is false with no pointer, an unreadable one, or one naming a jar that is gone.
func LiveFirst(dir, name string) (first string, ok bool) {
	data, err := os.ReadFile(dir + "/" + name + FirstSuffix)
	if err != nil {
		return "", false
	}
	first = strings.TrimSpace(string(data))
	if KeptName.MatchString(name+"."+first) && lint.IsFile(dir+"/"+name+"."+first) {
		return first, true
	}
	return "", false
}
