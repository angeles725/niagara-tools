// Package loopback decides whether a listen or dial address stays on this
// machine. lab-sim's control port has no authentication, which is acceptable
// only because it never leaves the loopback interface.
package loopback

import (
	"fmt"
	"net"
)

// Check returns nil when addr is host:port with a loopback IP address or the
// name localhost. An empty host (every interface), any other address and any
// other name are refused: a name is never resolved to find out.
func Check(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%q is not host:port", addr)
	}
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("%q is not a loopback address (127.0.0.1, [::1] or localhost)", addr)
}
