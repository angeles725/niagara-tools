package box

// #986 D18 (live, 4.14.0.162 lab station): a user without the admin role logs in over BOX but may
// not read oBIX, so the version probe GET /obix/about/ answers 403. Like /file/ and spy pages, a
// 403 there is a denied permission of an authenticated user, not a failed login: About answers an
// HttpStatusError and never latches, so n4_connect does not pause the session it just opened.
// A 401 still latches.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func aboutWith(t *testing.T, code int) (error, []*Error) {
	t.Helper()
	st := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}))
	t.Cleanup(st.Close)
	c := newClient(t, st.URL, nil)
	var latched []*Error
	c.OnAuthError = func(e *Error) { latched = append(latched, e) }
	_, err := c.About(0, 0)
	return err, latched
}

func TestAbout403IsADeniedPermissionNotAFailedLogin(t *testing.T) {
	err, latched := aboutWith(t, 403)
	var be *Error
	if !errors.As(err, &be) || be.Kind != KindHTTPStatus || be.Code != 403 || be.IsAuth() {
		t.Fatalf("403: %#v", err)
	}
	if len(latched) != 0 {
		t.Errorf("a 403 on /obix/about/ latched: %v", latched[0].Msg)
	}
}

func TestAbout401StillLatches(t *testing.T) {
	err, latched := aboutWith(t, 401)
	var be *Error
	if !errors.As(err, &be) || !be.IsAuth() {
		t.Fatalf("401: %#v", err)
	}
	if len(latched) != 1 {
		t.Errorf("a 401 on /obix/about/ latched %d times, want 1", len(latched))
	}
}
