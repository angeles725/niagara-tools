// Command signing-doctor checks the certificate and private key export a module is re-signed with,
// before the build: a CA certificate or a key export OpenSSL 3 cannot read is refused up front.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/signingdoctor"
)

func main() { cli.Run(signingdoctor.Main) }
