package ngdeploy

import (
	"os"
	"strings"
)

// checkSettings refuses a deploy that lacks a required setting or whose station path is not a WSL
// directory, then the option values (checkOptions).
func (d *deploy) checkSettings() int {
	for _, name := range required {
		if d.env(name) == "" {
			return d.fail(ExitEnv, msgRequired, name)
		}
	}
	mode := d.raw[optMode]
	for _, profile := range profilesOf[mode] {
		if name := expectedTypes[profile]; d.env(name) == "" {
			return d.fail(ExitEnv, msgRequiredForMode, name, mode)
		}
	}
	station := d.env("STATION_MODULES_DIR")
	if strings.Contains(station, `\`) {
		return d.fail(ExitEnv, msgBackslashes)
	}
	if !isDir(station) {
		return d.fail(ExitEnv, msgStationMissing, station)
	}
	if code := d.checkOptions(); code != proceed {
		return code
	}
	for _, profile := range profilesOf[mode] {
		if name := expectedTypes[profile]; !wholeNumber.MatchString(d.env(name)) {
			return d.fail(ExitEnv, msgTypeCount, name)
		}
	}
	d.module, d.station = d.env("MODULE_NAME"), station
	d.moduleDir = dirname(d.env("GRADLEW_PATH")) + "/" + d.module
	return proceed
}

// jarName is "<MODULE>-<profile>.jar".
func (d *deploy) jarName(profile string) string { return d.module + "-" + profile + ".jar" }

// profileDir is the source directory of a profile, beside gradlew.
func (d *deploy) profileDir(profile string) string {
	return d.moduleDir + "/" + d.module + "-" + profile
}

// builtJar is the jar gradle assembles for a profile.
func (d *deploy) builtJar(profile string) string {
	return d.profileDir(profile) + "/build/libs/" + d.jarName(profile)
}

// builtJars are the built jars of the mode, in order.
func (d *deploy) builtJars() []string {
	var jars []string
	for _, profile := range profilesOf[d.raw[optMode]] {
		jars = append(jars, d.builtJar(profile))
	}
	return jars
}

// hasProfile reports whether the mode builds the profile.
func (d *deploy) hasProfile(profile string) bool {
	for _, p := range profilesOf[d.raw[optMode]] {
		if p == profile {
			return true
		}
	}
	return false
}

// dirname and basename are the POSIX commands of the same name over a path.
func dirname(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		if path == "" {
			return "."
		}
		return "/"
	}
	i := strings.LastIndex(trimmed, "/")
	if i < 0 {
		return "."
	}
	if dir := strings.TrimRight(trimmed[:i], "/"); dir != "" {
		return dir
	}
	return "/"
}

func basename(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		if path == "" {
			return ""
		}
		return "/"
	}
	return trimmed[strings.LastIndex(trimmed, "/")+1:]
}

// isDir is test -d: false too for a path that cannot be checked, which every caller refuses.
func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
