// Package mqtttlsport is lint-mqtt-tls-port (G87): a Generic MQTT authenticator that connects over TLS and
// leaves the broker port at 1883. The authenticator defaults to connectionType=UserLoginOverSSL while
// brokerPort defaults to 1883 (javap of abstractMqttDriver-rt 4.15.6.20, BAbstractMqttAuthenticator static
// init: sipush 1883; guide abstractMqttDriver-GenericMqttAuthenticator.txt:19), and a TLS broker listens on
// 8883, so the connection fails against a TLS listener (types/cloud-connector.md § 9). The contract is the
// G87 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-u9-c68-c71.md); toolbelt/lint-mqtt-tls-port.sh is the
// wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each station directory: every *.bog under it, read through xmlsrc (a zipped bog
// through its file.xml member). An authenticator is a <p> whose t names GenericMqttAuthenticator of the
// abstractMqttDriver module (its prefix resolved through the m="prefix=module" declarations). Its
// connectionType is read from the v of its child slot, the enum tag as written (a "@range" suffix dropped);
// absent is the default UserLoginOverSSL. Its brokerPort is the v of its child slot; absent is 1883.
package mqtttlsport

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name> convention).
const ID = "lint-mqtt-tls-port"

// Spec registers the rule: [--strict] [--] <station-dir>... (each on its own), WARN rows (exit 1 under
// --strict), left out of n4lint --all (its roots are station directories, not source roots).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-mqtt-tls-port.sh [--strict] <station-dir>...",
		Select:     lint.Selection{Names: []string{"*.bog"}},
		EmptyError: lint.EmptyRow{Msg: "no *.bog under the root (wrong path: pass a station directory)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	driverModule = "abstractMqttDriver"
	authType     = "GenericMqttAuthenticator"
	defaultPort  = "1883"
	msg          = " TLS connection on port 1883: set brokerPort to the broker's TLS port, 8883 (types/cloud-connector.md § 9)"
)

// tlsTypes are the connection types that run over TLS; an absent connectionType is the first.
var tlsTypes = map[string]bool{"UserLoginOverSSL": true, "AnonymousOverSSL": true}

// CheckTree reads each bog; a malformed or unreadable one fails the scan (exit 3) after the other bogs' rows, and
// what Niagara reads although XML 1.0 forbids it (xmlsrc.Doc.Warnings) is one WARN row before the bog's own.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	var bad error
	for _, f := range files {
		doc, err := xmlsrc.Open(f.Path)
		if err != nil {
			bad = err
			continue
		}
		for _, w := range lint.XMLWarnings(doc) {
			w.Path = f.Path
			out = append(out, w)
		}
		out = append(out, check(f.Path, doc)...)
	}
	return out, bad
}

func check(path string, doc *xmlsrc.Doc) []lint.Finding {
	elems := doc.Elements()
	modules := map[string]string{} // prefix -> module
	for _, e := range elems {
		if m, ok := e.Attr("m"); ok {
			for _, decl := range strings.Fields(m) {
				if p, mod, ok := strings.Cut(decl, "="); ok {
					modules[p] = mod
				}
			}
		}
	}
	var out []lint.Finding
	for i, e := range elems {
		t, _ := e.Attr("t")
		prefix, name, ok := strings.Cut(t, ":")
		if e.Name != "p" || !ok || name != authType || modules[prefix] != driverModule {
			continue
		}
		conn, _, _ := strings.Cut(doc.SlotValue(i, "connectionType"), "@")
		if conn == "" {
			conn = "UserLoginOverSSL"
		}
		port := doc.SlotValue(i, "brokerPort")
		if port == "" {
			port = defaultPort
		}
		if tlsTypes[conn] && port == defaultPort {
			out = append(out, lint.Finding{Path: path, Line: e.Line, Msg: doc.SlotPath(i) + msg})
		}
	}
	return out
}
