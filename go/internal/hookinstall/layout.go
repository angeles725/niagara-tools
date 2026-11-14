// Package hookinstall is the Go side of scripts/install-hooks.sh (activate this repository's own
// .githooks) and scripts/install-client-hooks.sh (install the kit's pre-push gate into a client
// repository). The two script headers are the contracts.
//
// One file per concern: layout.go (names and paths, data), repohooks.go (install-hooks), postmerge.go
// (the one hook it runs), clienthooks.go (install-client-hooks), activehooks.go (the default hooks
// directory scan). The git config read, the copy and the file tests come from internal/installtools.
package hookinstall

const (
	hooksDir      = ".githooks"      // the hooks directory both installers point core.hooksPath at
	hooksPathKey  = "core.hooksPath" // as written with `git config`
	hooksPathRead = "core.hookspath" // as `git config --list` spells it
	postMerge     = "./" + hooksDir + "/post-merge"

	// install-client-hooks: paths under the kit checkout holding the launcher, and in the client clone.
	kitToolbelt   = "build-n4-module-kit/toolbelt"  // relative to the launcher's repository
	templateRel   = "/../templates/client-pre-push" // appended to the toolbelt path, as the script spelled it
	clientHook    = "/" + hooksDir + "/pre-push"    // appended to the client work tree root
	toolbeltKey   = "n4kit.toolbelt"                // local config: where the hook finds the kit
	sampleSuffix  = ".sample"                       // git's inactive example hooks
	clientIndent  = "                      "        // continuation lines of install-client-hooks
	installIndent = "              "                // continuation lines of install-hooks
)
