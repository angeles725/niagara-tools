package rcscan

import "strings"

// floorRow reports the features f (each with a leading space) above the Chromium 83 panel floor.
func (s *fileScan) floorRow(f string) {
	if f != "" && !s.allowed("browser-floor") {
		s.emit(s.o.sevFloor(), "browser-floor", f[1:]+" above the Chromium 83 panel floor")
	}
}

func cssTokens(t string) string {
	f := ""
	for _, c := range []struct {
		hit  bool
		name string
	}{{insetDecl.MatchString(t), "inset"}, {aspect.MatchString(t), "aspect-ratio"}, {minCall.MatchString(t), "min()"},
		{maxCall.MatchString(t), "max()"}, {clampCall.MatchString(t), "clamp()"}, {strings.Contains(t, "backdrop-filter"), "backdrop-filter"}} {
		if c.hit {
			f += " " + c.name
		}
	}
	return f
}

func (s *fileScan) jsTokens(t string) string {
	f := ""
	for _, c := range [][2]string{{"??=", "??="}, {"||=", "||="}, {"&&=", "&&="}, {".replaceAll(", ".replaceAll()"},
		{".at(", ".at()"}, {"structuredClone(", "structuredClone()"}} {
		if strings.Contains(t, c[0]) {
			f += " " + c[1]
		}
	}
	if s.src.ext == "js" && topAwait.MatchString(t) {
		f += " top-level-await"
	}
	return f
}

// cssFeed reads a CSS line rule by rule: a { starts a rule, a } closes it.
func (s *fileScan) cssFeed(text string) {
	for {
		i := strings.IndexAny(text, "{}")
		if i < 0 {
			break
		}
		s.cssSeg(text[:i])
		if text[i] == '{' {
			s.blk, s.gapNR = "", 0
		} else {
			s.cssClose()
		}
		text = text[i+1:]
	}
	s.cssSeg(text)
}

func (s *fileScan) cssSeg(seg string) {
	s.blk += " " + seg
	if s.gapNR == 0 && gapDecl.MatchString(seg) && !s.allowed("browser-floor") {
		s.gapNR = s.nr
	}
}

// cssClose ends a rule: a gap in a rule that declares no display:grid is a flex gap (Chromium 84).
func (s *fileScan) cssClose() {
	if s.gapNR > 0 && !gridDecl.MatchString(s.blk) {
		s.emitAt(s.gapNR, s.o.sevFloor(), "browser-floor", "gap in a non-grid (flex) rule is above the Chromium 83 panel floor (flex gap = C84)")
	}
	s.blk, s.gapNR = "", 0
}
