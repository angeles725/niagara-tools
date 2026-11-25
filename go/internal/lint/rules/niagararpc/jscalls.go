package niagararpc

// The `baja.rpc(` calls of the *.js files under the scanned root.

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

func isJS(f *javasrc.File) bool { return strings.HasSuffix(f.Path, ".js") }

// firstBajaRpc is "<path>:<line>" of the first `baja.rpc(` call in files' *.js sources, "" when there is none. A
// comment, a string and a template are no call (jssrc.Tokens); baja.rpc sends `sys/rpc` over BOX
// (bajaScript-ux rc/sys.js:613-643, rc/comm.js:729; the station side is box-rt BSysChannel.java:200-201).
func firstBajaRpc(files []*javasrc.File) string {
	for _, f := range files {
		if !isJS(f) {
			continue
		}
		t := jssrc.Tokens(f.Raw)
		for i := 0; i+3 < len(t); i++ {
			if t[i].Kind == jssrc.Ident && t[i].Text == "baja" && t[i+1].Kind == jssrc.Punct && t[i+1].Text == "." && t[i+2].Kind == jssrc.Ident && t[i+2].Text == "rpc" && t[i+3].Kind == jssrc.Punct && t[i+3].Text == "(" {
				return f.Path + ":" + strconv.Itoa(t[i].Line+1)
			}
		}
	}
	return ""
}
