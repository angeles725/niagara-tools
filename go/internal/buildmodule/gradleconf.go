package buildmodule

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// warnPluginNotInM2 warns (never blocks) when the niagara plugin version gradle will use — the override, else
// the gradlePluginVersion of settings.gradle.kts — has no directory under <niagara_home>/etc/m2: gradle then
// fails to resolve the plugin after a retarget to another Niagara installation.
func (b *build) warnPluginNotInM2() int {
	settings := b.gradleRoot + "/settings.gradle.kts"
	if !lint.IsFile(settings) {
		return proceed
	}
	version := b.plugin
	if lines, binary, _ := grepLines(settings); version == "" && !binary {
		for _, line := range lines {
			if m := settingsPluginVersion.FindStringSubmatch(line); m != nil {
				version = m[1]
				break
			}
		}
	}
	if version != "" && !findwalk.HasDirWithin(b.home+"/"+m2Repository, version, m2Depth) {
		b.warn(msgPluginNotInM2, version, b.home)
	}
	return proceed
}

// configureGradle settles the plugin version and the arguments gradle gets after its tasks.
func (b *build) configureGradle() int {
	if b.plugin == "" {
		// No override: use the module's OWN pinned niagara plugin version.
		if b.plugin = b.pinnedPlugin(); b.plugin != "" {
			b.say(msgPluginDetected, b.plugin)
		}
	}
	b.gradleArgs = []string{"-Pniagara_home=" + b.home, "-Porg.gradle.java.installations.paths=" + b.java8}
	if b.plugin != "" {
		b.gradleArgs = append(b.gradleArgs, "-PniagaraPluginVersion="+b.plugin)
	}
	b.gradleArgs = append(b.gradleArgs, b.opt.gradleProps...)
	if b.opt.werror {
		b.gradleArgs = append(b.gradleArgs, "--init-script", b.here+"/"+werrorInit)
	}
	return proceed
}

// pinnedPlugin is the niagara plugin version the module root pins: the first niagaraPluginVersion line of
// gradle.properties (what follows its last "="), else the getOrElse("X") default of settings.gradle.kts.
func (b *build) pinnedPlugin() string {
	version := b.grepFirst(b.root+"/gradle.properties", firstMatch, func(line string) (string, bool) {
		if !propertiesPlugin.MatchString(line) {
			return "", false
		}
		return strings.Trim(line[strings.LastIndex(line, "=")+1:], blanks), true
	})
	if version != "" {
		return version
	}
	return b.grepFirst(b.root+"/settings.gradle.kts", everyMatch, func(line string) (string, bool) {
		if m := settingsPluginDefault.FindStringSubmatch(line); m != nil {
			return m[1], true
		}
		return "", false
	})
}

// How far grepFirst reads: the first matching line only, or every matching line until one has a value.
const (
	firstMatch = true
	everyMatch = false
)

// grepFirst is the first non-empty value match finds in the lines of a regular file, "" when there is none;
// match also says whether the line matched at all, and firstOnly stops at the first line that did. As for
// grep, a file that cannot be read is said so and a match in a file holding a NUL byte is only reported.
func (b *build) grepFirst(path string, firstOnly bool, match func(line string) (value string, matched bool)) string {
	lines, binary, err := grepLines(path)
	if err != nil {
		b.warn(msgGrepUnreadable, path, strerror.Text(err))
	}
	for _, line := range lines {
		value, matched := match(line)
		if matched && binary {
			b.warn(msgGrepBinary, path)
			return ""
		}
		if value != "" || (matched && firstOnly) {
			return value
		}
	}
	return ""
}

// grepLines is the lines of a regular file: none when path is not one. binary reports a NUL byte, which
// makes the file binary to grep; err is why a regular file could not be read.
func grepLines(path string) (lines []string, binary bool, err error) {
	if !lint.IsFile(path) {
		return nil, false, nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err
		}
		return nil, false, err
	}
	return strings.Split(string(text), "\n"), bytes.IndexByte(text, 0) >= 0, nil
}
