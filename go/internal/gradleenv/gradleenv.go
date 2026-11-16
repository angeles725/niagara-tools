// Package gradleenv is the environment a gradle build of a module gets when NIAGARA_HOME is a mirror, the one
// rule build.sh (go/internal/buildmodule) and ng-deploy (go/internal/ngdeploy) both apply, so both build the
// same jars.
//
// A mirror build is kept away from the real ~/.tridium keystore [ev: retro r8-mirror-userhome Δ1]:
// org.gradle.jvmargs=-Duser.home never reaches an already-running daemon, so the signing plugin's default
// profile (user.home/.tridium/security) would read the REAL keystore. JAVA_TOOL_OPTIONS reaches every JVM
// gradle starts and --no-daemon stops the reuse of a stale daemon. GRADLE_USER_HOME is pinned, because gradle
// derives it from user.home and offline resolution needs it; moving user.home also moves ~/.m2, so
// mavenLocal() is pinned to the real one unless already set.
//
// The JVM splits JAVA_TOOL_OPTIONS on whitespace (JDK 8 has no quoting), so both values it would carry are
// refused with whitespace: the mirror path and $HOME. [ev: retro r8-buildsh-316-308]
package gradleenv

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

const (
	// UserHomeDir is the user.home of a mirror build, under the mirror.
	UserHomeDir = ".mirror-user-home"
	// SigningProfile is where the signing plugin keeps its default profile, under user.home.
	SigningProfile = ".tridium/security/niagara.signing.xml"
	// blanks is the bash [[:space:]] class.
	blanks = " \t\n\v\f\r"
	// repoOption is the JVM option that names the maven repository.
	repoOption = "-Dmaven.repo.local="
)

// Isolation is what a mirror build needs: the user.home to make before gradle runs, the signing profile the
// plugin writes there, the environment gradle and every JVM it starts get, and the arguments after the tasks.
type Isolation struct {
	UserHome   string
	SigningXML string
	Env        map[string]string // GRADLE_USER_HOME and JAVA_TOOL_OPTIONS
	Args       []string
}

// Which value JAVA_TOOL_OPTIONS would carry with whitespace.
const (
	MirrorPath = "mirror" // the mirror niagara_home
	HomePath   = "HOME"   // $HOME, for the maven repository
)

// BlankError is a value JAVA_TOOL_OPTIONS would carry that holds whitespace.
type BlankError struct {
	Of   string // MirrorPath or HomePath
	Path string // the mirror niagara_home, or $HOME
}

func (e *BlankError) Error() string { return e.Of + " path contains whitespace: " + e.Path }

// Isolate is the isolation of a build against home, with env the environment it reads (JAVA_TOOL_OPTIONS,
// HOME, GRADLE_USER_HOME). It is nil, with no error, when home is not a mirror (modjar.IsMirror).
func Isolate(home string, env func(string) string) (*Isolation, error) {
	if !modjar.IsMirror(home) {
		return nil, nil
	}
	userHome := home + "/" + UserHomeDir
	if strings.ContainsAny(userHome, blanks) {
		return nil, &BlankError{Of: MirrorPath, Path: home}
	}
	options, real := env("JAVA_TOOL_OPTIONS"), modjar.HomeDir(env)
	repoSet := strings.Contains(options, repoOption)
	if !repoSet && strings.ContainsAny(real, blanks) {
		return nil, &BlankError{Of: HomePath, Path: real}
	}
	if options != "" {
		options += " "
	}
	options += "-Duser.home=" + userHome
	if !repoSet {
		options += " " + repoOption + real + "/.m2/repository"
	}
	return &Isolation{
		UserHome:   userHome,
		SigningXML: userHome + "/" + SigningProfile,
		Env:        map[string]string{"GRADLE_USER_HOME": modjar.GradleUserHome(env), "JAVA_TOOL_OPTIONS": options},
		Args:       []string{"--no-daemon"},
	}, nil
}
