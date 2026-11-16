package buildmodule

import "strings"

// isolateMirror keeps a mirror build away from the real ~/.tridium keystore [ev: retro r8-mirror-userhome Δ1]:
// org.gradle.jvmargs=-Duser.home never reaches an already-running daemon, so the signing plugin's default
// profile (user.home/.tridium/security) would read the REAL keystore. JAVA_TOOL_OPTIONS reaches every JVM
// gradle starts and --no-daemon stops the reuse of a stale daemon. GRADLE_USER_HOME is pinned, because gradle
// derives it from user.home and offline resolution needs it; moving user.home also moves ~/.m2, so
// mavenLocal() is pinned to the real one unless already set.
//
// The JVM splits JAVA_TOOL_OPTIONS on whitespace (JDK 8 has no quoting), so both values it would carry are
// refused with whitespace: the mirror path and $HOME. [ev: retro r8-buildsh-316-308]
func (b *build) isolateMirror() int {
	if !b.isMirror() {
		return proceed
	}
	userHome := b.home + "/" + mirrorUserHome
	if strings.ContainsAny(userHome, blanks) {
		b.warn(msgMirrorWhitespace, b.home)
		return ExitWhitespace
	}
	options, home := b.s.Env("JAVA_TOOL_OPTIONS"), b.homeDir()
	m2Set := strings.Contains(options, "-Dmaven.repo.local=")
	if !m2Set && strings.ContainsAny(home, blanks) {
		b.warn(msgHomeWhitespace, home)
		return ExitWhitespace
	}
	if _, code := b.run("mkdir", "-p", userHome); code != 0 {
		return code // as under bash's set -e
	}
	b.signingXML = userHome + "/" + signingProfile
	gradleHome := b.gradleUserHome()
	if options != "" {
		options += " "
	}
	options += "-Duser.home=" + userHome
	if !m2Set {
		options += " -Dmaven.repo.local=" + home + "/.m2/repository"
	}
	b.setenv["GRADLE_USER_HOME"], b.setenv["JAVA_TOOL_OPTIONS"] = gradleHome, options
	b.gradleArgs = append(b.gradleArgs, "--no-daemon")
	b.say(msgMirrorIsolated, userHome, gradleHome)
	return proceed
}
