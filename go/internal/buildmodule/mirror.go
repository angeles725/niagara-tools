package buildmodule

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/gradleenv"
)

// isolateMirror keeps a mirror build away from the real ~/.tridium keystore with the isolation ng-deploy
// applies too (gradleenv.Isolate: user.home in the mirror for every JVM, GRADLE_USER_HOME and the real maven
// repository pinned, --no-daemon) [ev: retro r8-mirror-userhome Δ1]. A value JAVA_TOOL_OPTIONS would carry
// with whitespace is refused (exit 33). [ev: retro r8-buildsh-316-308]
func (b *build) isolateMirror() int {
	iso, err := gradleenv.Isolate(b.home, b.s.Env)
	var blank *gradleenv.BlankError
	switch {
	case errors.As(err, &blank) && blank.Of == gradleenv.MirrorPath:
		b.warn(msgMirrorWhitespace, blank.Path)
		return ExitWhitespace
	case errors.As(err, &blank):
		b.warn(msgHomeWhitespace, blank.Path)
		return ExitWhitespace
	case iso == nil:
		return proceed
	}
	if _, code := b.run("mkdir", "-p", iso.UserHome); code != 0 {
		return code // as under bash's set -e
	}
	b.signingXML = iso.SigningXML
	for k, v := range iso.Env {
		b.setenv[k] = v
	}
	b.gradleArgs = append(b.gradleArgs, iso.Args...)
	b.say(msgMirrorIsolated, iso.UserHome, iso.Env["GRADLE_USER_HOME"])
	return proceed
}
