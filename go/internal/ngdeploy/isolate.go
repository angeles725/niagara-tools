package ngdeploy

import (
	"errors"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// isolateMirror gives every gradlew call of a mirror build the isolation build.sh gives it, from the one
// shared rule (gradleenv.Isolate): user.home in the mirror for every JVM gradle starts (the signing plugin
// then cannot read the real ~/.tridium keystore), GRADLE_USER_HOME and the real maven repository pinned, and
// --no-daemon. The environment is set in this process, so the commands after it inherit it as build.sh's do.
// A value JAVA_TOOL_OPTIONS would carry with whitespace is exit 10 before gradle runs. A home that is not a
// mirror (absent, or a real one opted in with --install-into-home) is left as it was.
func (d *deploy) isolateMirror() int {
	iso, err := gradleenv.Isolate(d.env("NIAGARA_HOME"), d.env)
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
	for name, value := range iso.Env {
		os.Setenv(name, value)
	}
	d.isolation = iso.Args
	d.say(msgMirrorIsolated, iso.UserHome, iso.Env["GRADLE_USER_HOME"])
	return proceed
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
