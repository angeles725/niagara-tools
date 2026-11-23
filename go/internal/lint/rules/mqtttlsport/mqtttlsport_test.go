package mqtttlsport

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G87 contract (build-n4-module-kit/ledger.d/2026-10-23-r8-u9-c68-c71.md), one hand-written case
// per testdata/*.txt: a Generic authenticator whose connectionType is absent (the default UserLoginOverSSL), or
// UserLoginOverSSL or AnonymousOverSSL, with brokerPort absent (the default 1883) or 1883 WARNs; brokerPort 8883,
// connectionType Anonymous and an AWS JITP authenticator (port preset 8883) are clean; the module alias of a type
// is resolved (x=abstractMqttDriver counts, a GenericMqttAuthenticator of another module does not); --strict; no
// bog, a malformed bog and usage exit 3.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 7) }
