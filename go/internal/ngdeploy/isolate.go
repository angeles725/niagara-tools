package ngdeploy

import (
	"errors"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// isolateMirror gives every gradlew call of a mirror build the isolation build.sh gives it, from the one
// shared rule (gradleenv.Isolate): user.home in the mirror for every JVM gradle starts (the signing plugin
// then cannot read the real ~/.tridium keystore), GRADLE_USER_HOME and the real maven repository pinned, and
// --no-daemon. The environment goes to every command the deploy starts after it (environ), as build.sh's
// does, and never into this process: a second run in the same process starts from the same environment.
// A value JAVA_TOOL_OPTIONS would carry with whitespace is exit 10 before gradle runs. A home that is not a
// mirror (absent, or a real one opted in with --install-into-home) is left as it was.
func (d *deploy) isolateMirror() int {
	iso, err := gradleenv.Isolate(d.niagaraHome, d.env)
	var blank *gradleenv.BlankError
	switch {
	case errors.As(err, &blank) && blank.Of == gradleenv.MirrorPath:
		return d.fail(ExitEnv, msgMirrorBlank, blank.Path)
	case errors.As(err, &blank):
		return d.fail(ExitEnv, msgHomeBlank, blank.Path)
	case iso == nil:
		return proceed
	}
	if code := d.show("mkdir", "-p", iso.UserHome); code != 0 {
		return code
	}
	d.childEnv, d.isolation = iso.Env, iso.Args
	d.say(msgMirrorIsolated, iso.UserHome, iso.Env["GRADLE_USER_HOME"])
	return proceed
}

// pluginFlag is the option build.sh has: the niagara plugin version gradle builds with, forwarded as
// -PniagaraPluginVersion (retro build-kit-bench-session Δ7).
const pluginFlag = "--plugin-version"

// pluginVersionShape is the version --plugin-version takes: digits only, as scaffold-module checks it.
var pluginVersionShape = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// pluginVersion reads the value of --plugin-version into the -PniagaraPluginVersion every gradlew call gets.
func (d *deploy) pluginVersion(rest []string) int {
	got := "nothing"
	if len(rest) > 0 {
		if pluginVersionShape.MatchString(rest[0]) {
			d.gradleProps = append(d.gradleProps, "-PniagaraPluginVersion="+rest[0])
			return proceed
		}
		got = rest[0]
	}
	return d.fail(ExitEnv, msgPluginVersion, got)
}

// gradleProp reads one --gradle-prop k=v of the command line into the -Pk=v every gradlew call gets, as
// build.sh does (gradleenv.Prop); one that is not k=v is exit 10.
func (d *deploy) gradleProp(rest []string) int {
	prop, ok := "", false
	got := "nothing"
	if len(rest) > 0 {
		prop, ok = gradleenv.Prop(rest[0])
		if rest[0] != "" {
			got = rest[0]
		}
	}
	if !ok {
		return d.fail(ExitEnv, msgGradleProp, got)
	}
	d.gradleProps = append(d.gradleProps, prop)
	return proceed
}
