package buildmodule

// The record of this checkout's own mirror builds (ownbuild.go).
const (
	msgOwnBuild          = "build.sh: INFO — the mirror's %s.jar is this checkout's own build of %s, not a release: drift check skipped"
	msgMirrorBuiltFailed = "build.sh: WARN — cannot write %s: %v (the next rebuild of this version will read as drift)"
)
