package boxstation

import (
	"fmt"
	"net/http"
	"strings"
)

// This file adds the station's spy pages to the servlets, as a station serves them (measured on
// Niagara 4.15): GET /ord?spy:/<path> answers an HTML page, and an unknown path answers the
// "Spy Not Found" page with HTTP 200.

// SpyPage is one spy page: its HTML and content type ("text/html;charset=utf-8" when ""), or,
// when Status is not 0, that HTTP status instead (e.g. 403 for a user without spy permission).
type SpyPage struct {
	Body        []byte
	ContentType string
	Status      int
}

const spyQuery = "spy:/"

func isSpy(r *http.Request) bool {
	return r.URL.Path == "/ord" && strings.HasPrefix(r.URL.RawQuery, spyQuery)
}

func (sv *Servlets) serveSpy(w http.ResponseWriter, r *http.Request) {
	var ok bool
	sv.St.Do(func(st *Station) { ok = st.authorized(r) })
	status, body, ctype := http.StatusUnauthorized, []byte{}, "text/html"
	if ok {
		sv.Do(func(*Servlets) {
			sv.SpyRequests = append(sv.SpyRequests, r.RequestURI)
			status, body, ctype = sv.spy(strings.Trim(unquote(r.URL.RawQuery[len(spyQuery):]), "/"))
		})
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (sv *Servlets) spy(path string) (int, []byte, string) {
	page, ok := sv.SpyPages[path]
	switch {
	case !ok:
		return http.StatusOK, []byte("<!DOCTYPE html><html><body><table class='spy-nav'><tr><td><a href='/ord?spy:/'>Index</a>" +
			"</td></tr></table><hr><h1>Spy Not Found</h1>spy:/" + path + "</body></html>"), "text/html;charset=utf-8"
	case page.Status != 0:
		return page.Status, []byte{}, "text/html"
	case page.ContentType == "":
		return http.StatusOK, page.Body, "text/html;charset=utf-8"
	}
	return http.StatusOK, page.Body, page.ContentType
}
