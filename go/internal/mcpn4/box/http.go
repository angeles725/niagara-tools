package box

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// This file is the one HTTP path of the client: urllib's opener as BoxClient builds it
// (HTTPSHandler plus _NoRedirect). Every request is one connection with "Connection: close",
// no compression and no HTTP/2, the timeout bounds each socket operation (Python's socket
// timeout, not a total), a redirect is refused before any byte reaches its target, and a
// failure becomes the text urllib's exception would print.

// response is a 2xx answer: its body read up to the cap and its headers.
type response struct {
	body   []byte
	header http.Header
}

// httpStatus is a non-2xx answer (urllib's HTTPError): the caller maps code to its own error.
type httpStatus struct{ code int }

func (e *httpStatus) Error() string { return fmt.Sprintf("HTTP %d", e.code) }

// transportFailure is a urllib URLError/OSError/ValueError: text is str(exc).
type transportFailure struct{ text string }

func (e *transportFailure) Error() string { return e.text }

// deadlineConn sets a fresh deadline before every read and write: a socket timeout.
type deadlineConn struct {
	net.Conn
	timeout time.Duration
}

func (c *deadlineConn) Read(b []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.timeout))
	return c.Conn.Read(b)
}

func (c *deadlineConn) Write(b []byte) (int, error) {
	c.SetDeadline(time.Now().Add(c.timeout))
	return c.Conn.Write(b)
}

// do sends one request and reads at most limit bytes of a 2xx body. It returns *response, or an
// error: *httpStatus, *Error (a refused redirect), *transportFailure or *PyError.
func (c *Client) do(method, path string, body []byte, header map[string]string, timeout float64, limit int) (*response, error) {
	if timeout <= 0 {
		timeout = c.timeout
	}
	d := time.Duration(timeout * float64(time.Second))
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := (&net.Dialer{Timeout: d}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			return &deadlineConn{conn, d}, nil
		},
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: c.insecureTLS},
		TLSNextProto:       map[string]func(string, *tls.Conn) http.RoundTripper{},
		DisableKeepAlives:  true,
		DisableCompression: true,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, rd)
	if err != nil {
		return nil, &transportFailure{"<urlopen error " + err.Error() + ">"}
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept-Encoding", "identity")
	wrote := false
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		WroteRequest: func(httptrace.WroteRequestInfo) { wrote = true },
	}))
	resp, err := client.Do(req)
	if err != nil {
		return nil, failure(err, !wrote)
	}
	defer resp.Body.Close()
	switch code := resp.StatusCode; {
	case code >= 200 && code < 300:
	case code == 301 || code == 302 || code == 303 || code == 307 || code == 308:
		if err := redirect(req.URL, code, resp.Header); err != nil {
			return nil, err
		}
		return nil, &httpStatus{code}
	default:
		return nil, &httpStatus{code}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)))
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) { // a short body is read as it is
		return nil, failure(err, false)
	}
	return &response{body: raw, header: resp.Header}, nil
}

// redirect is _NoRedirect: a 3xx with a usable Location (or URI) is refused, naming only the
// target host. Without one urllib raises the plain HTTPError (nil here).
func redirect(base *url.URL, code int, h http.Header) error {
	loc := h.Get("Location")
	if loc == "" {
		loc = h.Get("Uri")
	}
	if loc == "" {
		return nil
	}
	target, err := base.Parse(loc)
	if err != nil {
		return nil
	}
	switch strings.ToLower(target.Scheme) {
	case "http", "https", "ftp", "":
	default:
		return nil
	}
	host := strings.ToLower(target.Hostname())
	if host == "" {
		host = "unknown host"
	}
	return &Error{Kind: KindBox, Msg: fmt.Sprintf("refusing HTTP %d redirect to host %s "+
		"(credentials are never forwarded)", code, host)}
}

// failure words a Go transport error as str() of the exception urllib raises. Before the
// request is sent (connect, TLS, writing) urllib wraps the OSError in URLError, printed as
// "<urlopen error ...>"; while the answer is awaited or read the OSError surfaces bare.
func failure(err error, connecting bool) error {
	var errno syscall.Errno
	var dns *net.DNSError
	var cert *tls.CertificateVerificationError
	text := ""
	switch {
	case errors.Is(err, os.ErrDeadlineExceeded) || isTimeout(err):
		text = "timed out"
	case errors.As(err, &dns):
		text = "[Errno -3] Temporary failure in name resolution"
		if dns.IsNotFound {
			text = "[Errno -2] Name or service not known"
		}
	case errors.As(err, &cert):
		text = "[SSL: CERTIFICATE_VERIFY_FAILED] certificate verify failed: " + cert.Err.Error()
	case errors.As(err, &errno):
		text = fmt.Sprintf("[Errno %d] %s", int(errno), strerror.Text(errno))
	case !connecting && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)):
		return &transportFailure{"Remote end closed connection without response"}
	case strings.Contains(err.Error(), "malformed HTTP"):
		return &PyError{"BadStatusLine"} // http.client.HTTPException: no OSError, not wrapped
	default:
		text = err.Error()
	}
	if connecting {
		return &transportFailure{"<urlopen error " + text + ">"}
	}
	return &transportFailure{text}
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// latin1 is a header value as http.client decodes it: one code point per byte.
func latin1(s string) string {
	r := make([]rune, len(s))
	for i := 0; i < len(s); i++ {
		r[i] = rune(s[i])
	}
	return string(r)
}

// header returns the first value of name and whether it was sent (an empty value counts).
func header(h http.Header, name string) (string, bool) {
	v, ok := h[http.CanonicalHeaderKey(name)]
	if !ok || len(v) == 0 {
		return "", false
	}
	return latin1(v[0]), true
}
