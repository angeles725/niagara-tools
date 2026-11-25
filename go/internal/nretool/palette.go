package nretool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	paletteClass = "devkit:com.tridium.devkit.util.PaletteGenerator"
	paletteUsage = "usage: nre-tool palette <niagara_home> <moduleDir>"
)

// palette runs PaletteGenerator <moduleDir> (B1453 §1453.4). It writes <moduleDir>\module.palette
// (PaletteGenerator.java:33), replacing the file that is there, from the .gif files under <moduleDir>\src
// (:32,:37). It runs only on a module part directory of the kit's layout, says which file it will write, and
// needs nre.exe: the separator is a literal backslash, so a Linux nre writes a file named after the path.
func palette(args []string, s cli.Stdio) int {
	if len(args) != 2 {
		fmt.Fprintln(s.Err, paletteUsage)
		return exitUsage
	}
	home, dir := args[0], args[1]
	if !isPartDir(dir) {
		fmt.Fprintf(s.Err, "nre-tool: palette: %s is not a module part directory (needs module-include.xml and src/)\n", dir)
		return exitUsage
	}
	nre := findNre(home)
	if nre == "" {
		fmt.Fprintf(s.Err, "nre-tool: no bin/nre.exe or bin/nre under %s\n", home)
		return exitUsage
	}
	if !strings.HasSuffix(nre, ".exe") {
		fmt.Fprintln(s.Err, "nre-tool: palette needs bin/nre.exe: PaletteGenerator writes <moduleDir>\\module.palette "+
			"with a Windows separator, so a Linux nre would write a file named after the whole path")
		return exitUsage
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(s.Err, "nre-tool: palette: %v\n", err)
		return exitUsage
	}
	win, ok := nativePath(nre, abs)
	if !ok {
		fmt.Fprintf(s.Err, "nre-tool: palette: no Windows path for %s: wslpath -w is missing or failed, and PaletteGenerator "+
			"joins with a backslash, so a POSIX path would write a stray file\n", abs)
		return exitEnv
	}
	fmt.Fprintf(s.Err, "nre-tool: PaletteGenerator writes %s (replacing it; review the change with git diff)\n",
		filepath.Join(abs, "module.palette"))
	code, out := runStreaming(nre, []string{paletteClass, win}, s)
	if code == 0 && strings.Contains(out, "Failed to generate palette") {
		fmt.Fprintln(s.Err, "nre-tool: PaletteGenerator reported a failure (main catches it and exits 0, :245)")
		return exitFound
	}
	return code
}

// isPartDir is a module part directory of the kit's layout (fixtures/MinimalDash/MinimalDash/MinimalDash-rt):
// module-include.xml beside src/.
func isPartDir(dir string) bool {
	inc, err := os.Stat(filepath.Join(dir, "module-include.xml"))
	src, err2 := os.Stat(filepath.Join(dir, "src"))
	return err == nil && !inc.IsDir() && err2 == nil && src.IsDir()
}
