package buildmodule

import "github.com/angeles725/niagara-tools/go/internal/modjar"

// warnMirrorSigning says, before gradle runs, what a mirror build cannot do: gradle runs with user.home
// isolated to <mirror>/.mirror-user-home (isolateMirror), so the signing plugin never sees the user's
// ~/.tridium profile and signs with a throwaway key it creates there. A station will not take that jar, so
// the build only proves the module compiles. The warning names the build that signs with the real profile.
// It is printed on every mirror build, whether or not a profile exists yet (warnDevKey speaks after the
// build, and only when one does). [ev: retro client-e-dashboard-live-check Δ28]
func (b *build) warnMirrorSigning() {
	if modjar.IsMirror(b.home) {
		b.warn(msgMirrorSigning, b.home, b.home)
	}
}
