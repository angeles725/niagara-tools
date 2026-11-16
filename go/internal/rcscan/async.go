package rcscan

import "strings"

// fetchOutside (module map modes only) reports a fetch( call in JavaScript (a .js file or an inline <script>)
// outside an rc/js/services/ file: only the services call the network (modularity-gaps Δ2). The exemption is
// anchored at rc/js/services/ below the artifact dir: a services/ segment elsewhere (js/pages/services/,
// rc/services/) or above the module is not it.
func (s *fileScan) fetchOutside() {
	if !s.o.mapped() || thirdParty(s.src.rc) || strings.HasPrefix(s.src.rc, "js/services/") ||
		!fetchCall.MatchString(s.line) || s.allowed("fetch-outside-services") {
		return
	}
	s.emit(s.o.sevMap(), "fetch-outside-services", "fetch( outside rc/js/services/ -- call the API through"+
		" js/services/api.js (types/frontend-standard.md § Component layout)")
}

// fetch reads a fetch( call's argument list (up to 15 lines) for a signal; the 3 lines before it count
// too (the fetchT helper idiom).
func (s *fileScan) fetch() {
	if s.fPend {
		s.fetchFeed(s.line)
		return
	}
	if m := fetchCall.FindStringIndex(s.line); m != nil && !s.allowed("fetch-no-signal") {
		s.fPend, s.fNR, s.fDepth, s.fLns = true, s.nr, 0, 0
		s.fSig = strings.Contains(s.prev[0]+"\n"+s.prev[1]+"\n"+s.prev[2], "signal")
		s.fetchFeed(s.line[m[0]:])
	}
}

func (s *fileScan) fetchFeed(t string) {
	s.fSig = s.fSig || strings.Contains(t, "signal")
	s.fDepth += strings.Count(t, "(") - strings.Count(t, ")")
	s.fLns++
	if s.fDepth <= 0 || s.fLns >= 15 {
		s.fetchResolve()
	}
}

func (s *fileScan) fetchResolve() {
	if !s.fSig {
		s.emitAt(s.fNR, s.o.sevW(), "fetch-no-signal", fetchMsg)
	}
	s.fPend = false
}

// interval reports setInterval over an inline async function or a name the file declares async.
func (s *fileScan) interval() {
	if s.allowed("setinterval-async") {
		return
	}
	if intervalAsync.MatchString(s.line) {
		s.emit(s.o.sevW(), "setinterval-async", "setInterval over an async function"+pollMsg)
	} else if m := intervalName.FindString(s.line); m != "" {
		if nm := intervalS.ReplaceAllString(intervalP.ReplaceAllString(m, ""), ""); s.isAsync[nm] {
			s.emit(s.o.sevW(), "setinterval-async", "setInterval("+nm+") where "+nm+" is async"+pollMsg)
		}
	}
}
