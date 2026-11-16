package ntplmanifest

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// module is what the build files under a module root declare.
type module struct {
	files []string        // every *.gradle.kts read (settings.gradle.kts is not a build file)
	names map[string]bool // each module a build file declares, and the module's own parts
	known bool            // every build file under the root was read: names is the whole declaration
}

// declaredIn reads every build file under root (a symlinked one and those under a symlinked directory
// too: lint.WalkFollow) through the shared reader (lint.DeclaredDeps: api, nre and
// implementation calls). A part is a dependency of its own templates without declaring itself, so the
// directory of each build file, and the file's own stem (<part>.gradle.kts), count as declared. A build
// file that cannot be read or tokenized is an error: what the module declares is then not known, and no
// template's dependencies are compared against the part of it that was read.
func declaredIn(root string) (module, error) {
	m := module{names: map[string]bool{}}
	files, walkErr := lint.WalkFollow(root, lint.Selection{Names: []string{"*.gradle.kts"}, Except: []string{settingsFile}, PrunePaths: lint.PartOutputs()})
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return m, fmt.Errorf(scanFailure, path, err)
		}
		toks, err := ktssrc.Tokens(string(data))
		if err != nil {
			return m, fmt.Errorf(scanFailure, path, err)
		}
		for _, name := range lint.DeclaredDeps(toks) {
			m.names[name] = true
		}
		m.names[lint.Basename(lint.Dirname(path))] = true
		if stem := strings.TrimSuffix(lint.Basename(path), ".gradle.kts"); stem != "build" {
			m.names[stem] = true
		}
		m.files = append(m.files, path)
	}
	if walkErr != nil {
		return m, fmt.Errorf(partialWalk, root, walkErr)
	}
	m.known = true
	return m, nil
}
