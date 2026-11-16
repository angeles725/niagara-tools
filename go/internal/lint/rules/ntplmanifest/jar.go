package ntplmanifest

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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
// (openZip); each template inflates to at most maxJarTemplate bytes and all of them to maxJarTemplates.
// A jar declares no build dependencies, so a dependency is judged against the installation alone.
func checkJar(root string, h home, stock map[string]bool) ([]lint.Finding, error) {
	display := rowpath.Display(root)
	f, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf(scanFailure, display, err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf(scanFailure, display, err)
	}
	refuse := func(msg string) []lint.Finding {
		return []lint.Finding{{Check: checkEntries, Path: display, Msg: msg}}
	}
	if st.Size() > maxArchive {
		return refuse(fmt.Sprintf(tooLarge, st.Size(), maxArchive)), nil
	}
	zr, rejected, err := openZip(f, st.Size())
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
	left := maxJarTemplates
	for _, e := range templates {
		entryPath := rowpath.Display(root + "!/" + e.Name)
		data, why := inflateTemplate(e, left)
		switch {
		case why == budgetSpent:
			// Mutation: NTPL-jar-budget -- reading on past the jar's total lets a jar of many templates inflate without bound
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: fmt.Sprintf(jarBudget, maxJarTemplates)})
			return c.rows(), nil
		case why != "":
			c.out = append(c.out, lint.Finding{Check: checkEntries, Path: entryPath, Msg: why})
			continue
		}
		left -= int64(len(data))
		a, err := readZip(bytes.NewReader(data), int64(len(data)))
		if err != nil { // a bytes.Reader does not fail; kept so a change of reader cannot hide one
			return c.rows(), fmt.Errorf(scanFailure, entryPath, err)
		}
		c.check(entryPath, underStationDir("", e.Name), a)
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

// budgetSpent is the why of a template the jar's total bound leaves no room for.
const budgetSpent = "budget"

// inflateTemplate reads one template entry: past maxJarTemplate it is not read (why is its row), and past
// left, what the jar's total still allows, why is budgetSpent. The declared size is checked first and the
// read is bounded too, so an entry that lies about its size is caught all the same.
func inflateTemplate(f *zip.File, left int64) (data []byte, why string) {
	tooBig := fmt.Sprintf(jarTemplateTooLarge, maxJarTemplate)
	switch {
	case f.UncompressedSize64 > uint64(maxJarTemplate):
		return nil, tooBig
	case int64(f.UncompressedSize64) > left:
		return nil, budgetSpent
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Sprintf(jarTemplateUnread, err)
	}
	defer rc.Close()
	data, err = io.ReadAll(io.LimitReader(rc, min(maxJarTemplate, left)+1))
	switch {
	case err != nil:
		return nil, fmt.Sprintf(jarTemplateUnread, err)
	case int64(len(data)) > maxJarTemplate:
		return nil, tooBig
	case int64(len(data)) > left:
		return nil, budgetSpent
	}
	return data, ""
}
