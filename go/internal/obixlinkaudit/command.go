package obixlinkaudit

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// Exit codes of obix-link-audit (unchanged from the bash tool).
const (
	ExitOK     = 0 // no MISMATCH or MISSING row
	ExitIssues = 1 // any MISMATCH or MISSING row
	ExitEnv    = 3 // usage or environment error
)

const usage = "usage: obix-link-audit.sh --map <wiring-map.md> (--xml <component.xml> | --obix <base-url> --component <path>) [--table 1|2] [--insecure]\n"

type options struct{ mapPath, xml, base, comp, table string }

// Main implements `obix-link-audit --map <wiring-map.md> (--xml <component.xml> | --obix <base-url>
// --component <path>) [--table 1|2] [--insecure]`, checks in the bash tool's order.
func Main(args []string, s cli.Stdio) int {
	say := func(format string, a ...any) { fmt.Fprintf(s.Err, "obix-link-audit: "+format+"\n", a...) }
	fail := func(format string, a ...any) int { say(format, a...); return ExitEnv }
	o, insecure, ok := parseArgs(args)
	if !ok || o.mapPath == "" {
		return usageExit(s)
	}
	if !isRegular(o.mapPath) {
		return fail("map not found: %s", o.mapPath)
	}
	if o.table != "1" && o.table != "2" {
		return usageExit(s)
	}
	live := o.xml == ""
	switch {
	case live && (o.base == "" || o.comp == ""), !live && (o.base != "" || o.comp != ""):
		return usageExit(s)
	case !live && !isRegular(o.xml):
		return fail("xml not found: %s", o.xml)
	}
	var body []byte
	if live {
		target, err := liveURL(o.base, o.comp)
		if err != nil {
			return fail("%v", err)
		}
		user, pass := s.Env("OBIX_USER"), s.Env("OBIX_PASS")
		if user == "" || pass == "" {
			return fail("live mode needs OBIX_USER and OBIX_PASS (a read-only oBIX user)")
		}
		// Mutation: OLA-http -- dropping this warning hides clear-text credentials.
		if len(o.base) >= 7 && strings.EqualFold(o.base[:7], "http://") {
			say("WARNING: http:// base sends the oBIX credentials in clear text; use https:// (--insecure for a self-signed certificate)")
		}
		if body, err = get(target, user, pass, insecure); err != nil {
			say("%v", err)
			return fail("GET failed: %s", target)
		}
		o.xml = target
	}
	mapText, err := os.ReadFile(o.mapPath)
	if err != nil {
		return fail("%v", err)
	}
	rows := DeclaredRows(string(mapText), o.table)
	if len(rows) == 0 {
		return fail("no Table %s rows in %s", o.table, o.mapPath)
	}
	var doc *xmlsrc.Doc
	if live {
		doc, err = xmlsrc.Parse(o.xml, body)
	} else {
		doc, err = xmlsrc.Open(o.xml)
	}
	// Mutation: OLA-malformed -- reading a malformed or empty response as no links prints every row MISSING.
	if err != nil {
		return fail("%v", err)
	}
	return Report(s.Out, rows, Links(doc))
}

func usageExit(s cli.Stdio) int {
	fmt.Fprint(s.Err, usage)
	return ExitEnv
}

// liveURL is <base>/config/<component>/ (one leading and one trailing slash of the component trimmed, as
// the bash tool did), or why the arguments cannot name one station GET. The base must be an http(s) URL
// with a host and without credentials, query or fragment, so no URL echoed later carries a password.
func liveURL(base, comp string) (string, error) {
	u, err := url.Parse(base)
	switch {
	case err != nil:
		return "", errors.New("--obix is not a URL") // the parse error would echo the raw base
	// Mutation: OLA-userinfo -- accepting user:password@ lets "GET failed: <url>" print the password.
	case u.User != nil:
		return "", errors.New("--obix must not carry credentials (user:password@); pass them in OBIX_USER / OBIX_PASS")
	case (u.Scheme != "http" && u.Scheme != "https") || u.Host == "":
		return "", fmt.Errorf("--obix needs an http:// or https:// base with a host: %s", base)
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		return "", fmt.Errorf("--obix must not carry a query or fragment: %s", base)
	}
	path := strings.TrimSuffix(strings.TrimPrefix(comp, "/"), "/")
	// Mutation: OLA-component -- a raw .., ?, # or % in the path GETs a different resource than the component.
	if !slotPath(path) {
		return "", fmt.Errorf("--component must be a slot path without .. segments, ?, #, %% or control characters: %q", comp)
	}
	return strings.TrimSuffix(base, "/") + "/config/" + path + "/", nil
}

// slotPath: path is pasted into the URL as is, so it may hold no .. segment, no ?, # or % (which would
// end or re-encode the path) and no control character.
func slotPath(path string) bool {
	for _, r := range path {
		if r < 0x20 || r == 0x7f || strings.ContainsRune("?#%", r) {
			return false
		}
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// parseArgs reads the flags as the bash loop did (a later flag wins); ok is false on an unknown
// argument or a flag without its value.
func parseArgs(args []string) (o options, insecure, ok bool) {
	o.table = "2"
	flags := map[string]*string{"--map": &o.mapPath, "--xml": &o.xml, "--obix": &o.base, "--component": &o.comp, "--table": &o.table}
	for i := 0; i < len(args); i++ {
		if args[i] == "--insecure" {
			insecure = true
			continue
		}
		dst, known := flags[args[i]]
		if !known || i+1 >= len(args) {
			return o, insecure, false
		}
		*dst = args[i+1]
		i++
	}
	return o, insecure, true
}

// isRegular is `[ -f path ]`: a regular file, symlinks followed.
func isRegular(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}
