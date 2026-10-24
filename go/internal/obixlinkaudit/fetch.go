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

// get makes ONE read-only GET of url with basic-auth credentials and returns the body. Like the curl
// call it replaces (-fsS, no -L): a redirect is not followed, a status of 400 or more is an error, and
// the station certificate is verified unless insecure (--insecure, a self-signed station).
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
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("the station answered HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
