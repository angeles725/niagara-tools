package boxstation

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// This file adds the station's GET servlets to the fake, as a station serves them: GET
// /file/<path> (FileServlet) and GET /ord/<ORD> (a BQL table as CSV); spy_servlet.go adds the
// spy pages. Servlets listens on its
// own address in front of a Station; every other request (BOX POSTs, GET /obix/about/) is the
// station's own answer.

// File is a station file: its bytes and content type, or, when Status is not 0, that HTTP status
// instead (e.g. 403 for a permission denial).
type File struct {
	Body        []byte
	ContentType string
	Status      int
}

// OrdRoute answers GET /ord/ when Needle is in the decoded ORD: Body with ContentType
// ("text/csv" when "").
type OrdRoute struct {
	Needle      string
	Body        []byte
	ContentType string
}

// Servlets serves the station's /file/ and /ord/ GETs in front of a Station. Lock it (through
// Do) to change Files or OrdRoutes, or read the request records, while it serves.
type Servlets struct {
	St        *Station
	URL       string
	Files     map[string]File
	OrdRoutes []OrdRoute
	// FileRequests and OrdRequests are the decoded paths and ORDs of every /file/ and /ord/ GET;
	// GETs counts every GET (FakeStation.about_requests), authorized or not.
	FileRequests, OrdRequests []string
	GETs                      int
	// SpyPages answer GET /ord?spy:/<path> by decoded path ("" is the index), see spy_servlet.go;
	// SpyRequests is the request URI of every spy GET, in order.
	SpyPages    map[string]SpyPage
	SpyRequests []string

	mu sync.Mutex
}

// ServeServlets serves st's servlets on 127.0.0.1 until the test ends; URL is the address the
// server under test should use for st.
func ServeServlets(t testing.TB, st *Station) *Servlets {
	t.Helper()
	sv := &Servlets{St: st, Files: map[string]File{}, FileRequests: []string{}, OrdRequests: []string{},
		SpyPages: map[string]SpyPage{}, SpyRequests: []string{}}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(sv.serve)}
	go func() { _ = srv.Serve(ln) }()
	sv.URL = "http://" + ln.Addr().String()
	t.Cleanup(func() { _ = srv.Close() })
	return sv
}

// Do runs fn with the servlets locked.
func (sv *Servlets) Do(fn func(sv *Servlets)) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	fn(sv)
}

// SetPassword changes the password the station (BOX and servlets) accepts.
func (sv *Servlets) SetPassword(p string) { sv.St.Do(func(st *Station) { st.Password = p }) }

func (sv *Servlets) serve(w http.ResponseWriter, r *http.Request) {
	isFile, isOrd := strings.HasPrefix(r.URL.Path, "/file/"), strings.HasPrefix(r.URL.Path, "/ord/")
	if r.Method != http.MethodGet {
		sv.St.serve(w, r)
		return
	}
	sv.Do(func(*Servlets) { sv.GETs++ })
	if isSpy(r) {
		sv.serveSpy(w, r)
		return
	}
	if !isFile && !isOrd {
		sv.St.serve(w, r)
		return
	}
	var ok bool
	sv.St.Do(func(st *Station) { ok = st.authorized(r) })
	status, body, ctype := http.StatusUnauthorized, []byte{}, "text/xml"
	if ok {
		sv.Do(func(*Servlets) {
			if isFile {
				status, body, ctype = sv.file(unquote(r.RequestURI[len("/file/"):]))
			} else {
				status, body, ctype = sv.ord(unquote(r.RequestURI[len("/ord/"):]))
			}
		})
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// file is FakeStation._file: a .bog is 403, an unknown path 404.
func (sv *Servlets) file(path string) (int, []byte, string) {
	sv.FileRequests = append(sv.FileRequests, path)
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".bog") || strings.HasSuffix(lower, ".bog.gz") {
		return http.StatusForbidden, []byte{}, "text/html"
	}
	f, ok := sv.Files[path]
	switch {
	case !ok:
		return http.StatusNotFound, []byte{}, "text/html"
	case f.Status != 0:
		return f.Status, []byte{}, "text/html"
	}
	return http.StatusOK, f.Body, f.ContentType
}

// ord is FakeStation._ord: the first route whose needle the ORD holds, else 404.
func (sv *Servlets) ord(text string) (int, []byte, string) {
	sv.OrdRequests = append(sv.OrdRequests, text)
	for _, route := range sv.OrdRoutes {
		if strings.Contains(text, route.Needle) {
			ctype := route.ContentType
			if ctype == "" {
				ctype = "text/csv"
			}
			return http.StatusOK, route.Body, ctype
		}
	}
	return http.StatusNotFound, []byte{}, "text/html"
}

// unquote is urllib.parse.unquote, which leaves a malformed escape as it is.
func unquote(s string) string {
	if out, err := url.PathUnescape(s); err == nil {
		return out
	}
	return s
}
