package niagaratest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// copyHome copies the install into <scratch>\home with robocopy: the dirs whole, then the top-level jars of
// modules\ but those an --exclude-jar glob matched and those whose module name has a dot (a stray
// `<name>.77-rt.jar` makes every boot rebuild the registry). The source is only read; licenses travel untouched.
func (h host) copyHome(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	p.windows = true
	src, dst := p.install.win, p.home().win
	for _, d := range copiedDirs {
		if _, err := os.Stat(p.install.wsl + "/" + d); err != nil {
			continue
		}
		if err := h.robocopy(ctx, src+`\`+d, dst+`\`+d, "/E"); err != nil {
			return -1, err
		}
	}
	es, err := os.ReadDir(p.install.wsl + "/modules")
	if err != nil {
		return -1, err
	}
	var jars, skipped []string
	for _, e := range es {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".jar") || isExcluded(p.excluded, n) {
			continue
		}
		if strings.Contains(strings.TrimSuffix(n, ".jar"), ".") || strings.ContainsAny(n, unsafeChars+" ") {
			skipped = append(skipped, n)
			continue
		}
		jars = append(jars, n)
	}
	// cmd.exe takes 8191 characters a line: copy the jars in batches well under it.
	for len(jars) > 0 {
		n, size := 0, 0
		for n < len(jars) && size+len(jars[n]) < 6000 {
			size += len(jars[n]) + 1
			n++
		}
		if err := h.robocopy(ctx, src+`\modules`, dst+`\modules`, jars[:max(n, 1)]...); err != nil {
			return -1, err
		}
		jars = jars[max(n, 1):]
	}
	for _, x := range p.excluded {
		row(s, "PASS", fmt.Sprintf("copy: excluded %s matching %s: %s", plural(len(x.jars), "jar", "jars"), x.glob, listed(x.jars, maxExcludedListed)))
	}
	if len(skipped) > 0 {
		row(s, "PASS", fmt.Sprintf("copy: %s (dotted module name): %s", plural(len(skipped), "jar skipped", "jars skipped"), strings.Join(skipped, " ")))
	}
	return -1, nil
}

// keptHome is the copy step of a --reuse-home run: nothing is copied over the kept home. Its copy, the registry
// the NRE built in it, the scratch niagara_user_home and user.home stay; the build installs the module jars again.
func (h host) keptHome(_ context.Context, p *plan, s cli.Stdio) (int, error) {
	p.windows = true
	row(s, "PASS", "reuse: "+p.dir.win+" (no copy: the home copy, its registry and the scratch niagara_user_home stay; the module jars are rebuilt)")
	return -1, nil
}

// robocopy fails on its exit 8 and above (0-7 report what was copied).
func (h host) robocopy(ctx context.Context, src, dst string, what ...string) error {
	args := append([]string{"/c", "robocopy", src, dst}, what...)
	code, err := h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: append(args, "/R:0", "/W:0", "/NFL", "/NDL", "/NJH", "/NJS", "/NP")})
	if err != nil {
		return err
	}
	if code >= 8 || code < 0 {
		return fmt.Errorf("robocopy %s exit %d", src, code)
	}
	return nil
}
