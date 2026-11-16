package ngdeploy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
)

// sameContent reports whether two jars hold the same module: the same set of entry names and the same bytes
// for every entry, apart from what a build of unchanged sources changes anyway (patterns.go lists it): the
// signature files directly under META-INF/, the digests of the manifest's per-entry sections, and the
// buildMillis attribute of the <module> element of META-INF/module.xml. why is set, and same is false, when a
// jar cannot be read in full: that is never "the same content".
func sameContent(a, b string) (same bool, why string) {
	x, why := jarContent(a)
	if why != "" {
		return false, why
	}
	y, why := jarContent(b)
	if why != "" {
		return false, why
	}
	if len(x) != len(y) {
		return false, ""
	}
	for name, content := range x {
		if other, ok := y[name]; !ok || !bytes.Equal(content, other) {
			return false, ""
		}
	}
	return true, ""
}

// jarContent is every entry of a jar the comparison reads, by name, with the parts a rebuild changes taken
// out. why says what stopped the read: a file that is not a zip, a jar with no entry or with two entries
// under one name (which one ships is the reader's choice), an entry that fails its CRC.
func jarContent(jar string) (content map[string][]byte, why string) {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return nil, fmt.Sprintf(whyNotAJarFile, jar, err)
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return nil, fmt.Sprintf(whyNoEntry, jar)
	}
	content = map[string][]byte{}
	for _, entry := range zr.File {
		if _, twice := content[entry.Name]; twice {
			return nil, fmt.Sprintf(whyTwoEntries, jar, entry.Name)
		}
		data, err := entryBytes(entry)
		if err != nil {
			return nil, fmt.Sprintf(whyEntryUnread, jar, entry.Name, err)
		}
		content[entry.Name] = data
	}
	for name, data := range content {
		switch {
		case isSignatureFile(name):
			delete(content, name)
		case name == manifestName:
			content[name] = withoutDigests(data)
		case name == moduleXML:
			content[name] = buildMillis.ReplaceAll(data, []byte(`${1}""`))
		}
	}
	return content, ""
}

// entryBytes reads one entry in full; archive/zip checks its size and CRC at the end.
func entryBytes(entry *zip.File) ([]byte, error) {
	r, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// isSignatureFile reports a signature file of a signed jar: META-INF/<NAME>.<ext> with one of
// signatureExts in any case, directly under META-INF/.
func isSignatureFile(name string) bool {
	base, ok := strings.CutPrefix(name, metaInf)
	if !ok || strings.Contains(base, "/") {
		return false
	}
	upper := strings.ToUpper(base)
	for _, ext := range signatureExts {
		if strings.HasSuffix(upper, ext) {
			return true
		}
	}
	return false
}

// withoutDigests is a manifest without the digests of its per-entry sections: the main section as it is
// (line ends read alike), then every per-entry section that still holds an attribute beside its Name once its *-Digest
// attributes are dropped (a continuation line belongs to the attribute before it), in sorted order. A
// section that only gave a digest is dropped whole: the set of entry names is compared by itself.
func withoutDigests(manifest []byte) []byte {
	text := strings.ReplaceAll(string(manifest), "\r\n", "\n")
	main, rest, _ := strings.Cut(text, "\n\n")
	var kept []string
	for _, section := range strings.Split(rest, "\n\n") {
		var attributes []string
		others := 0
		for _, line := range strings.Split(section, "\n") {
			switch {
			case line == "":
			case strings.HasPrefix(line, " ") && len(attributes) > 0:
				attributes[len(attributes)-1] += line[1:]
			default:
				attributes = append(attributes, line)
			}
		}
		var lines []string
		for _, attribute := range attributes {
			if digestAttribute.MatchString(attribute) {
				continue
			}
			if !strings.HasPrefix(attribute, manifestEntryName) {
				others++
			}
			lines = append(lines, attribute)
		}
		if others > 0 {
			kept = append(kept, strings.Join(lines, "\n"))
		}
	}
	sort.Strings(kept)
	return []byte(main + "\n\n" + strings.Join(kept, "\n\n"))
}
