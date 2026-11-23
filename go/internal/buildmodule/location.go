package buildmodule

import "github.com/angeles725/niagara-tools/go/internal/fstype"

// warnSlowMounts warns (the build still runs) when the gradle root or niagara_home sits on a WSL 9p/drvfs
// mount (/mnt/<drive>): gradle's per-file stat and hash work multiplies the mount's per-file latency — a
// measured file walk was 10x slower than on ext4. [ev: client A retro "defrost sequencing and HMI reload" Δ7]
func (b *build) warnSlowMounts() int {
	places := []struct{ label, path string }{{"gradle-root", b.gradleRoot}, {"niagara_home", b.home}}
	for _, p := range places {
		typ := fstype.OfPath(p.path, b.s.Env(fstype.MountsEnv))
		if fstype.IsWindowsMount(typ) {
			b.warn(msgSlowMount, p.label, p.path, typ)
		}
	}
	return proceed
}

// preflight runs the environment preflight unless --no-preflight; any failure of it ends the chain. The
// plugin override goes with it, so its plugin-pin check tests the version gradle will use.
// [ev: retro continuous-fan-post-defrost-delay Δ1]
func (b *build) preflight() int {
	if b.opt.skipPreflight {
		return proceed
	}
	b.say(msgPreflight)
	var args []string
	if b.plugin != "" {
		args = append(args, "--plugin-version", b.plugin)
	}
	switch b.runScript(preflightScript, append(args, b.home, b.gradleRoot)...) {
	case 0:
		return proceed
	case preflightFail:
		b.warn(msgPreflightFail)
	}
	return ExitEnv
}
