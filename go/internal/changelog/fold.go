package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// FoldRootEnv names the repository root used when no <repo-root> argument is given; the
// scripts/gen-changelog.sh wrapper exports it as its own ../ directory.
const FoldRootEnv = "GEN_CHANGELOG_DEFAULT_ROOT"

const foldTool = "gen-changelog"

// FoldMain implements `gen-changelog --check|--write [<repo-root>]`. --check validates every
// changelog.d/*.md and the `## [Unreleased]` header; --write appends the fragments, in file-name (date then
// slug) order, to the end of the `## [Unreleased]` section of CHANGELOG.md and then deletes them. Any
// violation means nothing is written or deleted. Exits: gen.ExitOK, gen.ExitStale (a violation), gen.ExitEnv.
func FoldMain(args []string, s cli.Stdio) int {
	mode, root := "", ""
	for _, a := range args {
		switch {
		case (a == "--check" || a == "--write") && mode == "":
			mode = a
		case strings.HasPrefix(a, "-") || root != "":
			mode = "bad"
		default:
			root = a
		}
	}
	if root == "" {
		root = s.Env(FoldRootEnv)
	}
	if (mode != "--check" && mode != "--write") || root == "" {
		fmt.Fprintf(s.Err, "usage: %s.sh --check|--write [<repo-root>]\n", foldTool)
		return gen.ExitEnv
	}
	src, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		fmt.Fprintf(s.Err, "%s: cannot read %s/CHANGELOG.md: %v\n", foldTool, root, err)
		return gen.ExitEnv
	}
	dir := filepath.Join(root, FragmentDir)
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(s.Err, "%s: cannot list %s: %v; nothing folded\n", foldTool, dir, err)
		return gen.ExitEnv
	}
	var names, bodies, problems []string
	for _, e := range entries { // os.ReadDir sorts by name: date, then slug
		why := CheckName(e.Name())
		if why == "" && !e.Type().IsRegular() {
			why = "not a regular file"
		}
		var body []byte
		if why == "" {
			if body, err = os.ReadFile(filepath.Join(dir, e.Name())); err != nil {
				fmt.Fprintf(s.Err, "%s: cannot read %s/%s: %v; nothing folded\n", foldTool, FragmentDir, e.Name(), err)
				return gen.ExitEnv
			}
			why = CheckFragment(string(body))
		}
		if why != "" {
			problems = append(problems, fmt.Sprintf("%s/%s: %s", FragmentDir, e.Name(), why))
			continue
		}
		names, bodies = append(names, e.Name()), append(bodies, strings.TrimRight(string(body), "\n"))
	}
	lines := strings.Split(string(src), "\n")
	start, end, ok := UnreleasedBounds(lines)
	if !ok {
		problems = append(problems, "CHANGELOG.md has no "+UnreleasedHeader+" line")
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(s.Err, "%s: %s\n", foldTool, p)
		}
		fmt.Fprintf(s.Err, "%s: nothing folded\n", foldTool)
		return gen.ExitStale
	}
	if mode == "--check" {
		fmt.Fprintf(s.Out, "%s: %d fragment(s) under %s/ and CHANGELOG.md %s are valid\n", foldTool, len(names), FragmentDir, UnreleasedHeader)
		return gen.ExitOK
	}
	if len(names) == 0 {
		fmt.Fprintf(s.Out, "%s: no fragment under %s/; CHANGELOG.md unchanged\n", foldTool, FragmentDir)
		return gen.ExitOK
	}
	// insert before the section's trailing blank lines and its `---` separator, if any
	at := end
	for at-1 > start && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	if at-1 > start && strings.TrimSpace(lines[at-1]) == "---" {
		for at--; at-1 > start && strings.TrimSpace(lines[at-1]) == ""; at-- {
		}
	}
	var add []string
	for _, b := range bodies {
		add = append(add, "")
		add = append(add, strings.Split(b, "\n")...)
	}
	if at < len(lines) && strings.TrimSpace(lines[at]) != "" {
		add = append(add, "")
	}
	out := append(append(append([]string{}, lines[:at]...), add...), lines[at:]...)
	if code := (gen.File{Tool: foldTool, Path: filepath.Join(root, "CHANGELOG.md"), Content: strings.Join(out, "\n")}).Write(s); code != gen.ExitOK {
		return code
	}
	for _, n := range names {
		if err := os.Remove(filepath.Join(dir, n)); err != nil {
			fmt.Fprintf(s.Err, "%s: CHANGELOG.md is written but %s/%s could not be deleted: %v; delete it by hand\n", foldTool, FragmentDir, n, err)
			return gen.ExitEnv
		}
		fmt.Fprintf(s.Out, "%s: folded %s/%s\n", foldTool, FragmentDir, n)
	}
	return gen.ExitOK
}
