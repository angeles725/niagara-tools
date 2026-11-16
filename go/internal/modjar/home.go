// Package modjar is what the tools that build a Niagara module and the ones that deploy it both have to know
// about the niagara_home the build writes into and about who signs the module's jars: build-module
// (toolbelt/build.sh) and ng-deploy (scripts/ng-deploy.sh) read it, so that a guard of one is the guard of
// the other. home.go is the mirror marker; signer.go the signer the verify gate is told to expect.
package modjar

import "os"

// MirrorMarker is the file toolbelt/mirror-niagara-home.sh leaves at the root of a mirror: a niagara_home
// that holds it is a throwaway copy, and gradle's :jar step may replace the jars of its modules/ directory.
const MirrorMarker = ".niagara-mirror"

// IsMirror reports a niagara_home made by mirror-niagara-home.sh: one whose marker is a regular file.
func IsMirror(home string) bool {
	st, err := os.Stat(home + "/" + MirrorMarker)
	return err == nil && st.Mode().IsRegular()
}
