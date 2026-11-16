// Package skillinstall is the Go side of scripts/install-skill.sh and scripts/check-skill-drift.sh: the
// tracked skill launchers of this repository and their installed copies in each agent harness
// (<harness-dir>/skills/<skill>/SKILL.md, plus <harness-dir>/prompts/<skill>.md where the harness needs a
// prompt file for a bare /<skill>). The two script headers are the contracts.
//
// One file per concern: layout.go (names and exit codes, data), harness.go (the --harness table and its
// paths), prompt.go (the prompt file, rendered from <kit>/skill/prompt.md or prompts/default.md), table.go (the SKILLS table read from the
// installer script), options.go (the two command lines), install.go, target.go, materialize.go and
// plan.go (install-skill: every file checked before the first write), drift.go and driftrow.go (check-skill-drift). Hashing, copying and file tests come from internal/installtools.
package skillinstall

const (
	installerScript = "install-skill.sh" // the script whose SKILLS line is the table, for both commands
	defaultSkill    = "build-n4-module"
	// kitSkill is the one skill whose install also materializes a kit's generated state.
	kitSkill       = "build-n4-module"
	kitEnv         = "BUILD_N4_KIT"          // a kit checkout, used when it holds kitMarker
	kitMarker      = "METHODOLOGY.md"        // what makes a directory a kit
	repoKit        = "build-n4-module-kit"   // the kit of the launcher's own repository
	kitStateScript = "toolbelt/kit-state.sh" // run as `bash <kit>/toolbelt/kit-state.sh --write <kit>`
)

// Exit codes shared by the two commands (their script headers).
const (
	exitOK       = 0
	exitDiverged = 1 // install-skill: diverged without --force; check-skill-drift: any FAIL row
	exitUsage    = 2
	exitEnv      = 3
)
