package buildmodule

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// selectProfiles decides which <MOD>-<profile> projects gradle builds: --profiles as given, else every
// rt/ux/wb directory with a gradle file AND sources. A gradle file alone is a scaffold (a client A -wb
// one has no sources): it is reported and skipped.
func (b *build) selectProfiles() int {
	if b.opt.profiles != "" {
		b.profiles = splitProfiles(b.opt.profiles)
	} else {
		for _, p := range autoProfiles {
			dir := b.profileDir(p)
			switch {
			case !lint.IsDir(dir):
			case hasGradleFile(dir) && hasSources(dir):
				b.profiles = append(b.profiles, p)
			default:
				b.say(msgSkipScaffold, b.opt.mod, p)
			}
		}
	}
	if len(b.profiles) == 0 {
		b.warn(msgNoProfile, b.root, b.opt.mod)
		return ExitEnv
	}
	for _, p := range b.profiles {
		if !lint.IsDir(b.profileDir(p)) {
			b.warn(msgProfileMissing, b.profileDir(p))
			return ExitEnv
		}
	}
	return proceed
}

// tasks are the gradle tasks of the selected profiles: clean, slotomatic and jar of each, in order.
func (b *build) tasks() []string {
	var tasks []string
	for _, p := range b.profiles {
		project := ":" + b.jarName(p) + ":"
		tasks = append(tasks, project+"clean", project+"slotomatic", project+"jar")
	}
	return tasks
}

// splitProfiles splits a --profiles value as `IFS=, read -r -a` does: the first line, on every comma, with
// one trailing comma closing the last name instead of adding an empty one. An empty first line names none.
func splitProfiles(list string) []string {
	list, _, _ = strings.Cut(list, "\n")
	if list == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(list, ","), ",")
}

// hasGradleFile reports a build.gradle, a build.gradle.kts or any visible *.gradle.kts in dir.
func hasGradleFile(dir string) bool {
	if lint.IsFile(dir+"/build.gradle") || lint.IsFile(dir+"/build.gradle.kts") {
		return true
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".gradle.kts") && !strings.HasPrefix(e.Name(), ".") {
			return true
		}
	}
	return false
}

// hasSources reports a regular .java, .js or .html file anywhere under dir/src.
func hasSources(dir string) bool {
	if !lint.IsDir(dir + "/src") {
		return false
	}
	found := false
	findwalk.Walk(dir+"/src", false, findwalk.NoLimit, func(e findwalk.Entry) bool {
		if found || e.Dir {
			return !found
		}
		for _, suffix := range sourceSuffixes {
			if strings.HasSuffix(e.Name, suffix) && isRegular(e.Path) {
				found = true
			}
		}
		return true
	}, nil)
	return found
}
