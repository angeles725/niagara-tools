package preflight

import (
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// toolchains is the opt-in --toolchains row (corpus B1242 §1242.7 K6, from B1235 §1235.2): a 4.15 build needs a full
// JDK 8, and the raw or Windows gradlew route picks it from Gradle's toolchains, not from build.sh's -P paths (which
// the jdk8 row checks). gradle.properties org.gradle.java.installations.paths pinned to a full JDK 8 (bin/javac and
// a 1.8 release or java -version) passes without starting gradle; otherwise the course command
// `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` runs in the gradle root (-p) and a
// listed toolchain with Language Version 8 and Is JDK true passes. A listing without one fails; no executable
// gradlew, or a listing with no toolchain at all (gradle failed), is SKIP, never a false PASS.
func (r *run) toolchains() {
	if !r.toolchainsOn || r.win {
		return
	}
	const check = "toolchains"
	gw := r.gr + "/gradlew"
	if syscall.Access(gw, 1) != nil {
		r.row("SKIP", check, "no executable gradlew in "+r.gr)
		return
	}
	for _, p := range strings.Split(prop(r.gr+"/gradle.properties", "org.gradle.java.installations.paths"), ",") {
		if p = strings.TrimSpace(p); p != "" && r.fullJDK8(p) {
			r.row("PASS", check, "org.gradle.java.installations.paths pins a full JDK 8: "+p+" (gradle.properties)")
			return
		}
	}
	listed := parseToolchains(string(r.h.output(gw, []string{"-q", "-p", r.gr, "javaToolchains", "-Porg.gradle.java.installations.auto-detect=true"}, true)))
	if len(listed) == 0 {
		r.row("SKIP", check, "gradlew javaToolchains listed no toolchain (gradle failed or printed nothing)")
		return
	}
	for _, tc := range listed {
		if (tc["Language Version"] == "8" || tc["Language Version"] == "1.8") && tc["Is JDK"] == "true" {
			r.row("PASS", check, "full JDK 8 listed by gradlew javaToolchains: "+tc["Location"])
			return
		}
	}
	r.row("FAIL", check, "gradlew javaToolchains lists no full JDK 8 (a 4.15 build requires one, a JRE is not enough): install a JDK 8 or set org.gradle.java.installations.paths in gradle.properties")
}

// fullJDK8: dir holds bin/javac (a JDK, not a JRE) and is Java 8 by its release file or its bin/java -version.
func (r *run) fullJDK8(dir string) bool {
	if !lint.IsFile(dir + "/bin/javac") {
		return false
	}
	if lint.IsFile(dir + "/release") {
		return releaseIs8(dir + "/release")
	}
	return r.javaIs8(dir + "/bin/java")
}

// parseToolchains reads the javaToolchains report: each " + <name>" line opens a toolchain (the " + Options" block
// is not one) and its "| <field>: <value>" lines fill it.
func parseToolchains(out string) []map[string]string {
	var all []map[string]string
	var cur map[string]string
	for _, ln := range strings.Split(out, "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(t, "+ "):
			cur = nil
			if strings.TrimSpace(t[2:]) != "Options" {
				cur = map[string]string{}
				all = append(all, cur)
			}
		case strings.HasPrefix(t, "|") && cur != nil:
			if k, v, ok := strings.Cut(strings.TrimSpace(t[1:]), ":"); ok {
				cur[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	return all
}
