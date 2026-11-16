package ngdeploy

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// errNoRoot is a descriptor whose root element encoding/xml cannot read.
var errNoRoot = errors.New("no root element")

// withoutBuildStamp is META-INF/module.xml without what the vendor's plugin stamps on it at every build: the
// values of the attributes of buildStamps (buildMillis, buildHost) of its root element, wherever they stand
// among its attributes. encoding/xml finds the start tag of the root element and reads its attributes; the
// values of those two are cut out of the tag, and every other byte of the file, before, inside and after
// that tag, is kept as it is. The same names on any other element are content.
func withoutBuildStamp(descriptor []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(descriptor))
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil } // only names matter here
	for {
		start := dec.InputOffset()
		token, err := dec.RawToken()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errNoRoot, err)
		}
		root, is := token.(xml.StartElement)
		if !is {
			continue
		}
		tag := descriptor[start:dec.InputOffset()]
		for _, attr := range root.Attr {
			if attr.Name.Space == "" && buildStamps[attr.Name.Local] {
				tag = withoutValue(tag, attr.Name.Local)
			}
		}
		return append(append(append([]byte{}, descriptor[:start]...), tag...), descriptor[dec.InputOffset():]...), nil
	}
}

// withoutValue is a start tag encoding/xml accepted, without the value of the attribute of that name: the
// quotes stay. Attribute values are walked quote to quote, so a ">" or the name inside a value is not read
// as markup.
func withoutValue(tag []byte, name string) []byte {
	for i := 0; i < len(tag); i++ {
		if tag[i] != '"' && tag[i] != '\'' {
			continue
		}
		end := i + 1 + bytes.IndexByte(tag[i+1:], tag[i])
		if end <= i {
			return tag // no closing quote: not a tag the decoder accepts
		}
		before := bytes.TrimRight(bytes.TrimRight(bytes.TrimRight(tag[:i], " \t\r\n"), "="), " \t\r\n")
		if at := bytes.LastIndexAny(before, " \t\r\n"); string(before[at+1:]) == name {
			return append(append([]byte{}, tag[:i+1]...), tag[end:]...)
		}
		i = end
	}
	return tag
}

// withoutDigests is a manifest without the digests of its per-entry sections: the main section as it is
// (line ends read alike), then every per-entry section that still holds an attribute beside its Name once
// its *-Digest attributes are dropped (a continuation line belongs to the attribute before it), in sorted
// order. A section that only gave a digest is dropped whole: the set of entry names is compared by itself.
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
