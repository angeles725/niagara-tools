package obixlinkaudit

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// getTimeout bounds the whole GET, as curl --max-time 20 did.
const getTimeout = 20 * time.Second

// maxBody bounds the response read (a component dump is kilobytes; this is far past any real one).
const maxBody = 32 << 20

// get makes ONE read-only GET of url with basic-auth credentials and returns the body. Like the curl
// call it replaces (-fsS, no -L) a redirect is not followed, but unlike curl a 3xx is an error as a status
// of 400 or more is; the body is capped at maxBody, and the station certificate is verified unless
// insecure (--insecure, a self-signed station).
// The credentials go only into the Authorization header; no error text carries them.
func get(url, user, pass string, insecure bool) ([]byte, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		// Mutation: OLA-insecure -- verifying anyway refuses a self-signed station; skipping always accepts any.
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // opt-in per run only
	}
	client := &http.Client{
		Transport:     tr,
		Timeout:       getTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("the station answered HTTP %s", resp.Status)
	case resp.StatusCode >= 300:
		// Mutation: OLA-redirect -- reading a 3xx body as the component audits a login or error page.
		return nil, fmt.Errorf("redirect not followed: HTTP %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err == nil && len(body) > maxBody {
		return nil, fmt.Errorf("response larger than %d MiB", maxBody>>20)
	}
	return body, err
}
