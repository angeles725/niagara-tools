package boxstation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

// ProductVersion is what GET /obix/about/ reports unless SetVersion changes it.
const ProductVersion = "4.14.0.162"

// Start serves the station on 127.0.0.1 until the test ends; URL is set.
func (st *Station) Start(t testing.TB) *Station {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(st.serve)}
	go func() { _ = srv.Serve(ln) }()
	st.URL = "http://" + ln.Addr().String()
	t.Cleanup(func() { _ = srv.Close() })
	return st
}

func (st *Station) authorized(r *http.Request) bool {
	want := base64.StdEncoding.EncodeToString([]byte(st.User + ":" + st.Password))
	return r.Header.Get("Authorization") == "Basic "+want
}

func (st *Station) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		st.serveGet(w, r)
		return
	}
	raw, err := io.ReadAll(r.Body)
	st.Do(func(*Station) { st.requests++ })
	if !st.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="station"`)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var frame map[string]any
	if err == nil {
		err = json.Unmarshal(raw, &frame)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if st.Hook != nil {
		if status, body := st.Hook(frame); status != 0 {
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.WriteHeader(status)
			_, _ = w.Write(body)
			return
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := json.Marshal(st.handleFrame(frame))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(out)))
	_, _ = w.Write(out)
}

// serveGet answers GET /obix/about/ in the shape of a live 4.14 answer; any other path, or a
// station without a version, is 404.
func (st *Station) serveGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/xml")
	var version string
	st.Do(func(*Station) { version = st.version })
	switch {
	case !st.authorized(r):
		w.WriteHeader(http.StatusUnauthorized)
	case strings.TrimRight(r.URL.Path, "/") != "/obix/about" || version == "":
		w.WriteHeader(http.StatusNotFound)
	default:
		fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<obj is=\"obix:About\" href=\"/obix/about/\">\n"+
			"  <str name=\"obixVersion\" val=\"1.1\"/>\n  <str name=\"serverName\" val=\"fake-host\"/>\n"+
			"  <str name=\"vendorName\" val=\"Tridium\"/>\n  <str val=\"%s\" name=\"productVersion\"/>\n</obj>\n",
			version)
	}
}
