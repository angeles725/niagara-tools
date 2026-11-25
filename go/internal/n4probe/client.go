package n4probe

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	requestTimeout = 10 * time.Second
	maxBody        = 1 << 20
)

// client does one request at a time with the probe's basic-auth credentials.
type client struct {
	http       *http.Client
	user, pass string
}

// reply is one request's result; err is a transport failure (no HTTP status).
type reply struct {
	status int
	body   []byte
	took   time.Duration
	err    error
}

func newClient(user, pass string, insecure bool) *client {
	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // --insecure-tls: a self-signed station
	}
	return &client{user: user, pass: pass, http: &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
		// A redirect is never followed: the credentials go to the URL the operator named only.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// do sends one request and times it. body nil sends none.
func (c *client) do(method, target string, body []byte) reply {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, target, rd)
	if err != nil {
		return reply{err: errors.New("bad request")}
	}
	req.SetBasicAuth(c.user, c.pass)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	t0 := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return reply{took: time.Since(t0), err: bare(err)}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	r := reply{status: resp.StatusCode, body: data, took: time.Since(t0)}
	if err != nil {
		r.err = bare(err)
	}
	return r
}

// bare drops the URL a *url.Error repeats, so no query ever reaches the output.
func bare(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// ok is a 2xx answer.
func (r reply) ok() bool { return r.err == nil && r.status >= 200 && r.status < 300 }

// every calls fn at rate hz, one call at a time, until fn returns true or d has passed since start. A slow
// call is not made up for with a burst.
func every(start time.Time, rate float64, d time.Duration, fn func(since time.Duration) bool) {
	interval := time.Duration(float64(time.Second) / rate)
	next := time.Now()
	for {
		if fn(time.Since(start)) {
			return
		}
		next = next.Add(interval)
		if now := time.Now(); next.Before(now) {
			next = now
		}
		if next.Sub(start) >= d {
			return
		}
		time.Sleep(time.Until(next))
	}
}
