package niagaratest

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// --exclude-jar <glob> (repeatable) leaves the top-level modules\*.jar it matches out of the copy, so a stale jar
// of the module under test (an old <module>-ux.jar) never sits next to the freshly built ones. The glob is a jar
// basename of letters, digits, _ . - and the wildcards * and ?, matched case-insensitively as Windows names are;
// no path separator, no "..", no cmd.exe or PowerShell metacharacter. A glob matching no jar is FAIL, exit 2
// (probably a typo), before anything is copied. Only the copy is affected: --install is never written.

var excludeGlobRe = regexp.MustCompile(`^[A-Za-z0-9_.*?-]+$`)

// maxExcludedListed is how many excluded jar names a copy row lists before "and N more".
const maxExcludedListed = 10

// excludeFlag is --exclude-jar: each use appends a validated glob.
type excludeFlag struct{ globs *[]string }

func (f excludeFlag) String() string { return "" }

func (f excludeFlag) Set(v string) error {
	if !excludeGlobRe.MatchString(v) || strings.Contains(v, "..") {
		return fmt.Errorf("--exclude-jar %q is not a jar basename glob (letters, digits, _ . - and * ?; no path, no ..)", v)
	}
	*f.globs = append(*f.globs, v)
	return nil
}

// jarExclusion is one --exclude-jar glob and the top-level jars of the install's modules\ it matched.
type jarExclusion struct {
	glob string
	jars []string
}

// matchExclusions matches every glob against the top-level jars of dir, in directory order.
func matchExclusions(dir string, globs []string) ([]jarExclusion, error) {
	es, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	xs := make([]jarExclusion, len(globs))
	for i, g := range globs {
		xs[i].glob = g
		for _, e := range es {
			if n := e.Name(); !e.IsDir() && strings.HasSuffix(n, ".jar") && globMatch(g, n) {
				xs[i].jars = append(xs[i].jars, n)
			}
		}
	}
	return xs, nil
}

// globMatch never errs: excludeGlobRe leaves path.Match no bracket or escape to reject.
func globMatch(glob, name string) bool {
	ok, _ := path.Match(strings.ToLower(glob), strings.ToLower(name))
	return ok
}

func isExcluded(xs []jarExclusion, name string) bool {
	for _, x := range xs {
		if globMatch(x.glob, name) {
			return true
		}
	}
	return false
}

// listed joins names with spaces, the first limit of them, then "and N more".
func listed(names []string, limit int) string {
	if len(names) <= limit {
		return strings.Join(names, " ")
	}
	return strings.Join(names[:limit], " ") + " and " + strconv.Itoa(len(names)-limit) + " more"
}
