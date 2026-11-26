// Command signing-inventory lists the signing profiles, keystores and aliases a developer has (subject,
// issuer, days left, code-signing or not, the file) and, with --module, what a module records about its signer.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/signinginventory"
)

func main() { cli.Run(signinginventory.Main) }
