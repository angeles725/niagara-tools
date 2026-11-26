package buildmodule

import (
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/structure"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// signingTemplate is the root signing block, relative to the toolbelt directory.
const signingTemplate = "../templates/signing-root.gradle.kts"

// requireSigningBlock ends the chain before gradle (exit 10) when a signing alias is configured and the root
// build.gradle.kts applies the signing plugin without a niagaraSigning {} block (the lint-structure L15
// detection, structure.LacksSigningBlock): the plugin reads neither the alias nor the profile there, signs with
// the mirror's throwaway key, and the build ends at the verify gate with exit 50 (#1121). It also runs under
// --no-preflight, which skips the preflight script only. A root it cannot read or scan, or one that does not
// apply the plugin, is left to L15 and to gradle.
func (b *build) requireSigningBlock() int {
	alias := modjar.ConfiguredAlias(b.opt.gradleProps, b.s.Env, b.gradleRoot)
	root := b.gradleRoot + "/build.gradle.kts"
	if alias == "" {
		return proceed
	}
	src, err := os.ReadFile(root)
	if err != nil {
		return proceed
	}
	if lacks, err := structure.LacksSigningBlock(string(src)); err != nil || !lacks {
		return proceed
	}
	b.warn(msgNoSigningBlock, alias, root, filepath.Join(b.here, signingTemplate))
	return ExitEnv
}
