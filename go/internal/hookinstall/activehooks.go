package hookinstall

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// activeHooks lists, in byte order, the hooks git would run from a default hooks directory: the
// executable regular files that are not `*.sample` (dot files are not hooks, and the shell glob never
// saw them). A missing directory has none; one that cannot be read is an error, never "none".
func activeHooks(dir string) ([]string, error) {
	isDir, err := installtools.IsDir(dir)
	if err != nil || !isDir {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var active []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, sampleSuffix) {
			continue
		}
		if installtools.IsExecutableFile(dir + "/" + name) {
			active = append(active, name)
		}
	}
	return active, nil
}
