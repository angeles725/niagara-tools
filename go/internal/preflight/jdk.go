package preflight

import (
	"bytes"
	"os"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// jdk8 finds a JDK 8 under --jvm-dir, then JAVA_HOME, never $HOME: a candidate whose release file says
// JAVA_VERSION="1.8.", else (no release file, as some WSL openjdk-8 packages ship) one whose
// bin/java -version reports "1.8. (research retro companero B792-B793).
func (r *run) jdk8() {
	found := ""
	if lint.IsDir(r.jvmDir) {
		cands := globDirs(r.jvmDir)
		for _, c := range cands {
			if c += "/"; lint.IsFile(c+"/release") && releaseIs8(c+"/release") {
				found = c
				break
			}
		}
		for _, c := range cands {
			if found != "" {
				break
			}
			if c += "/"; !lint.IsFile(c+"/release") && r.javaIs8(c+"/bin/java") {
				found = c
			}
		}
	}
	if jh := r.s.Env("JAVA_HOME"); found == "" && jh != "" {
		if lint.IsFile(jh + "/release") {
			if releaseIs8(jh + "/release") {
				found = jh
			}
		} else if r.javaIs8(jh + "/bin/java") {
			found = jh
		}
	}
	if found != "" {
		r.row("PASS", "jdk8", "JDK 8 found: "+found)
	} else {
		r.row("FAIL", "jdk8", "JDK 8 not found under "+r.jvmDir+" (or JAVA_HOME); use --jvm-dir to override")
	}
}

func releaseIs8(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && bytes.Contains(b, []byte(`JAVA_VERSION="1.8.`))
}

// javaIs8 is `[ -x java ] && java -version 2>&1 | grep -q '"1\.8\.'`.
func (r *run) javaIs8(java string) bool {
	return syscall.Access(java, 1) == nil && bytes.Contains(r.h.Output(java, []string{"-version"}, true), []byte(`"1.8.`))
}

// globDirs is the bash glob "<dir>"/*/ in the C locale: the non-hidden entries of dir that are
// directories (symlinks followed), bytewise sorted, each spelled dir + "/" + name.
func globDirs(dir string) []string {
	entries, _ := os.ReadDir(dir) // an unreadable dir globs to nothing
	var out []string
	for _, e := range entries {
		if p := dir + "/" + e.Name(); !strings.HasPrefix(e.Name(), ".") && lint.IsDir(p) {
			out = append(out, p)
		}
	}
	return out
}
