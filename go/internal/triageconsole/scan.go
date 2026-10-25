package triageconsole

import (
	"fmt"
	"strings"
)

// group is one own row: a grouping key's count, first/last timestamp (by tsKey) and the display of its first hit.
type group struct {
	cnt                        int
	first, firstK, last, lastK string
	level, msg, frame          string
}

// siteGroup is one SITE row: a class + digit-normalized message, with its distinct raw messages counted.
type siteGroup struct {
	group
	cls, tag string
	dist     int
}

// block is the WARNING/SEVERE header being read and what its continuation lines added.
type block struct {
	level, ts, tag, hdrMsg string
	exClass, exMsg         string
	ownFKey, ownFDisp      string
}

// scanner is the awk program of the bash triage-console.sh: one record per console line, a pending block
// flushed into a channel (C1 own frame, C3a [sys] fatal, C2 own tag, C2b own path, or a site class) when the
// next header, the next file or the end arrives.
type scanner struct {
	pkg, ownPath   string
	ownTag, modPfx map[string]bool
	site           bool

	groups map[string]*group
	order  []string // insertion order of groups, for gawk's for-in order
	sites  map[string]*siteGroup
	raw    map[string]bool

	hasBlk bool
	blk    block
}

func newScanner(pkg, tags, prefixes string, site bool) *scanner {
	return &scanner{
		pkg: pkg, ownPath: strings.ReplaceAll(pkg, ".", "/"), ownTag: set(tags), modPfx: set(prefixes), site: site,
		groups: map[string]*group{}, sites: map[string]*siteGroup{}, raw: map[string]bool{},
	}
}

// file feeds one console: its first record flushes a block left open by the previous file.
func (s *scanner) file(content string) {
	if content == "" {
		return // awk reads no record from an empty file (and runs no FNR == 1 rule)
	}
	s.flush()
	for _, line := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		s.line(line)
	}
}

var months = map[string]int{
	"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6, "Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
	"Ene": 1, "Abr": 4, "Ago": 8, "Set": 9, "Dic": 12,
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	"ene": 1, "abr": 4, "ago": 8, "set": 9, "dic": 12,
}

// isLevel: the bash also compared substr(s, 1, 8) with the 9-byte "INFORMACI", which never matches, so an
// INFORMACIÓN line is no header (it continues the open block); the port keeps that.
func isLevel(s string) bool {
	switch s {
	case "SEVERE", "WARNING", "INFO", "CONFIG", "FINE", "FINER", "FINEST", "GRAVE", "ADVERTENCIA":
		return true
	}
	return false
}

func normLevel(lv string) string {
	switch lv {
	case "GRAVE":
		return "SEVERE"
	case "ADVERTENCIA":
		return "WARNING"
	}
	return lv
}

func severe(lv string) bool { return lv == "WARNING" || lv == "SEVERE" }

func (s *scanner) line(line string) {
	f1, f2 := fields12(line)
	if isLevel(f1) && strings.HasPrefix(f2, "[") {
		s.flush()
		lvn := normLevel(f1)
		rest := from(line, len(f1)+3) // skips "LEVEL [" assuming one blank, as the awk did
		closeTs := strings.IndexByte(rest, ']')
		if closeTs < 0 {
			return
		}
		ts, rest2 := rest[:closeTs], rest[closeTs+1:]
		if !strings.HasPrefix(rest2, "[") {
			return
		}
		closeTag := strings.IndexByte(rest2, ']')
		if closeTag < 0 {
			return
		}
		tag, msg := rest2[1:closeTag], from(rest2, closeTag+3) // one byte after "]" is skipped
		switch {
		case tag == "sys.xml" && severe(lvn): // C3b, single line
			s.record("WARNING"+subsep+norm(msg)+subsep+"[sys.xml]", ts, "WARNING", msg, "[sys.xml]")
		case tag == "sys.registry" && lvn == "SEVERE": // C3c, single line
			s.registry(ts, msg)
		case severe(lvn):
			s.hasBlk, s.blk = true, block{level: lvn, ts: ts, tag: tag, hdrMsg: msg}
		}
		return
	}
	if !s.hasBlk {
		return
	}
	cont := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(cont, "at ") {
		if s.blk.ownFKey == "" && strings.HasPrefix(cont[3:], s.pkg) {
			s.blk.ownFKey, s.blk.ownFDisp = frameKey(cont[3:]), frameDisp(cont[3:])
		}
		return
	}
	if s.blk.exClass != "" {
		return
	}
	if cp := strings.IndexByte(cont, ':'); cp >= 0 {
		if cand := cont[:cp]; exceptionClass(cand) {
			s.blk.exClass = cand
			s.blk.exMsg = strings.TrimLeft(from(cont, cp+3), " \t\n\v\f\r") // the byte after ':' is skipped
		}
	} else if exceptionClass(cont) {
		s.blk.exClass = cont
	}
}

func exceptionClass(c string) bool {
	return c != "" && !strings.Contains(c, " ") &&
		(strings.HasSuffix(c, "Exception") || strings.HasSuffix(c, "Error") || strings.HasSuffix(c, "Throwable") ||
			strings.HasPrefix(c, "java.") || strings.HasPrefix(c, "javax."))
}

// frameKey: "pkg.Cls.mth(File.java:N)" -> "Cls.mth" (the grouping key, no line number).
func frameKey(at string) string {
	clsMth := at
	if p := strings.IndexByte(at, '('); p >= 0 {
		clsMth = at[:p]
	}
	return lastTwo(clsMth)
}

// frameDisp: "pkg.Cls.mth(File.java:N)" -> "Cls.mth(File.java:N)" (the display keeps the site).
func frameDisp(at string) string {
	p := strings.IndexByte(at, '(')
	if p < 0 {
		return at
	}
	return lastTwo(at[:p]) + at[p:]
}

func lastTwo(clsMth string) string {
	if parts := split(clsMth, "."); len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return clsMth
}

func simpleCls(fqn string) string {
	parts := split(fqn, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// tsKey makes "HH:MM:SS DD-Mon-YY ..." sortable as YYMMDDHHMMSS (an unknown month is 00).
func tsKey(ts string) string {
	return substr(ts, 17, 2) + fmt.Sprintf("%02d", months[substr(ts, 13, 3)]) + substr(ts, 10, 2) +
		substr(ts, 1, 2) + substr(ts, 4, 2) + substr(ts, 7, 2)
}

// at counts one hit of g at ts, keeping the earliest and latest timestamps (ties keep the first seen).
func (g *group) at(ts string) {
	k := tsKey(ts)
	if g.cnt == 0 {
		g.first, g.firstK, g.last, g.lastK = ts, k, ts, k
	}
	g.cnt++
	if k < g.firstK {
		g.first, g.firstK = ts, k
	}
	if k > g.lastK {
		g.last, g.lastK = ts, k
	}
}

func (s *scanner) record(key, ts, level, msg, frame string) {
	g, ok := s.groups[key]
	if !ok {
		g = &group{level: level, msg: msg, frame: frame}
		s.groups[key] = g
		s.order = append(s.order, key)
	}
	g.at(ts)
}

// registry is C3c: [sys.registry] Missing class( for)? "?Prefix:Type with Prefix an own module prefix.
func (s *scanner) registry(ts, msg string) {
	p := strings.Index(msg, "Missing class")
	if p < 0 {
		return
	}
	rest := strings.TrimLeft(msg[p+len("Missing class"):], " \t")
	if strings.HasPrefix(rest, "for") {
		rest = strings.TrimLeft(rest[3:], " \t")
	}
	rest = strings.TrimPrefix(rest, `"`)
	colon := strings.IndexByte(rest, ':')
	if colon < 0 || !s.modPfx[rest[:colon]] {
		return
	}
	s.record("SEVERE"+subsep+norm(msg)+subsep+"[sys.registry]", ts, "SEVERE", msg, "[sys.registry]")
}

func (s *scanner) flush() {
	if !s.hasBlk {
		return
	}
	s.hasBlk = false
	b, lvn := s.blk, s.blk.level
	switch {
	case b.ownFKey != "": // C1 own frame
		msg := b.hdrMsg
		if b.exClass != "" {
			msg = simpleCls(b.exClass)
			if b.exMsg != "" {
				msg += ": " + b.exMsg
			}
		}
		s.record(lvn+subsep+b.exClass+subsep+norm(b.exMsg)+subsep+b.ownFKey, b.ts, lvn, msg, b.ownFDisp)
	case b.tag == "sys" && lvn == "SEVERE" && b.exClass != "": // C3a [sys] fatal
		msg := simpleCls(b.exClass)
		if b.exMsg != "" {
			msg += ": " + b.exMsg
		}
		s.record(lvn+subsep+b.exClass+subsep+norm(b.exMsg)+subsep+"[sys]", b.ts, lvn, msg, "[sys]")
	case severe(lvn) && (s.ownTag[b.tag] || s.ownPath != "" && strings.Contains(b.hdrMsg, s.ownPath)): // C2, C2b
		s.record(lvn+subsep+norm(b.hdrMsg)+subsep+b.tag, b.ts, lvn, b.hdrMsg, "["+b.tag+"]")
	case s.site:
		s.siteRecord(b)
	}
}
