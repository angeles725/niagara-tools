package ntplmanifest

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rowpath"
)

// namedJar is the Spec.FileRoot of the rule: a root file named *.jar (any letter case) is a template module
// jar, the one .jar of .ntpl entries Make Module writes (types/logic-authoring.md § Shipping a template
// module). The engine has already checked that the root is a regular file.
func namedJar(path string) bool { return strings.HasSuffix(strings.ToLower(path), jarSuffix) }

// isJar reports a root that is a regular file named *.jar: the root of a jar run.
func isJar(root string) bool {
	st, err := os.Stat(root)
	return err == nil && st.Mode().IsRegular() && namedJar(root)
}

// checkJar checks every *.ntpl and *.napl entry of the jar at root, in entry-name order, as a shipped
// template is checked; each row names <jar>!/<entry>. The jar is opened within the bounds of a .ntpl
// (openBounded, openZip); each template inflates to at most maxJarTemplate bytes, at most maxTemplates are
// read, and the templates and their entries draw on the run's budget. A jar declares no build dependencies,
// so a dependency is judged against the installation alone.
func checkJar(root string, h home, stock map[string]bool) ([]lint.Finding, error) {
	display := rowpath.Display(root)
	refuse := func(msg string) []lint.Finding {
		return []lint.Finding{{Check: checkEntries, Path: display, Msg: msg}}
	}
	f, size, refusedMsg, err := openBounded(root)
	switch {
	case err != nil:
		return nil, fmt.Errorf(scanFailure, display, err)
	case refusedMsg != "":
		// Mutation: NTPL-jar-size-first -- opening the jar before its bound turns an unreadable jar over it into exit 3
		return refuse(refusedMsg), nil
	}
	defer f.Close()
	zr, rejected, err := openZip(f, size)
	switch {
	case err != nil:
		return nil, fmt.Errorf(scanFailure, display, err)
	case rejected != "":
		return refuse(rejected), nil
	}
	templates := jarTemplates(zr)
	if len(templates) == 0 {
		return []lint.Finding{{Severity: lint.Pass, Path: display, Msg: noJarTemplates}}, nil
	}
	c := checker{root: display, mod: module{known: true}, home: h, stock: stock, jar: true}
	b := newBudget()
	for i, e := range templates {
		entryPath := rowpath.Display(root + "!/" + e.Name)
		if i == maxTemplates {
			// Mutation: NTPL-jar-count -- reading every template lets a jar of many small ones cost without bound
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: fmt.Sprintf(tooManyTemplates, len(templates), maxTemplates)})
			break
		}
		data, res, err := b.inflate(e, maxJarTemplate)
		switch res {
		case overBudget:
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: b.spentMsg("the template")})
			return c.rows(), nil
		case overLimit:
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: fmt.Sprintf(jarTemplateTooLarge, maxJarTemplate)})
			continue
		case unread:
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: fmt.Sprintf(jarTemplateUnread, err)})
			continue
		}
		a, err := readZip(bytes.NewReader(data), int64(len(data)), b)
		if err != nil { // a bytes.Reader does not fail; kept so a change of reader cannot hide one
			return c.rows(), fmt.Errorf(scanFailure, entryPath, err)
		}
		c.check(entryPath, underStationDir("", e.Name), a)
		if b.spent { // the row naming where it stopped is the last one
			break
		}
	}
	return c.rows(), nil
}

// jarTemplates are the template entries of a jar (the templateNames extensions), sorted by name.
func jarTemplates(zr *zip.Reader) []*zip.File {
	var out []*zip.File
	for _, f := range zr.File {
		for _, pattern := range templateNames {
			if ok, _ := path.Match(pattern, path.Base(f.Name)); ok && !f.FileInfo().IsDir() {
				out = append(out, f)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
