// Package niagararpc is the lint-niagara-rpc rule: an @NiagaraRpc method (the bajaux `baja.rpc` channel over fox,
// box or web) must not be unrestricted, and one that writes must check write permission. The browser is never the
// security boundary (types/security.md §2.7, types/dashboard.md RBAC, types/wb-widgets.md). Annotation attributes
// (transports, permissions, protectedTargets, isSecure; default permissions "I"): javax/baja/rpc/NiagaraRpc.java:15-23.
// Enforcement: com/tridium/util/NiagaraRpcUtil.java:138-171 (organized/baja/baja/decompiled).
//
//	unrestricted     FAIL  permissions = "unrestricted" (any case: equalsIgnoreCase, NiagaraRpcUtil.java:139, :162) or
//	                       permissions = "" (the same "no requirement", :138-151) on the @NiagaraRpc or a nested @Protected,
//	                       on the line of the literal.
//	unguarded-write  FAIL  an RPC whose body calls a write with no write check before it, on the line of the first write.
//	no-context       FAIL  an RPC whose last parameter is not typed Context (or a qualified ....Context), on the line of
//	                       that parameter (of the "(" when there is none). NiagaraRpcUtil.rpc appends the caller's Context
//	                       to the arguments and finds the method with getMethod over their types, a Context as Context.class,
//	                       so any other shape throws NoSuchMethodException (NiagaraRpcUtil.java:91-123, :189-192): a
//	                       subtype, an array, a Context that is not last and a method with no parameter are never called.
//	no-box-transport WARN  an RPC whose literal `transports` lists no box (TransportType.box; the enum is web, box, fox:
//	                       javax/baja/rpc/TransportType.java) while a *.js file under the same root calls `baja.rpc(`,
//	                       which sends sys/rpc over BOX (bajaScript-ux rc/comm.js:729; box-rt BSysChannel.java:200-201); the
//	                       station then finds no matching transport (NiagaraRpcUtil.java:128-130, :181-187). `transports` has
//	                       no default (NiagaraRpc.java:16), so an absent one is a compile error and a constant is not judged;
//	                       `transports = {}` lists no wire and warns. The row is on the `transports` line.
//	waiver-no-reason FAIL  a `// lint-niagara-rpc: reviewed` marker with no reason after it; the RPC is not waived.
//
// A write is `.set(`, `.invoke(` or one of setBoolean/setInt/setLong/setFloat/setDouble/setString/setFlags/setFacets
// with at least two arguments (a setter whose first argument is a number is JDBC's setInt(1, v), no write), or any
// `doXxx(` call (not java.security's doPrivileged, which the station's own camera RPCs call); a name after a type (a declaration in an anonymous class) is no call. A write check, before the write
// in the body: OPERATOR_WRITE, ADMIN_WRITE, hasOperatorWrite(, hasAdminWrite(, checkWrite(, checkCanWrite(,
// requireTargetWrite( or canWrite(. A string literal `permissions = "..."` holding w or W (also in a nested @Protected)
// is a check the station enforces itself (NiagaraRpcUtil.java:138-171) and passes the RPC; a `permissions` that is a
// constant or a concatenation is not judged. Waiver: a `// lint-niagara-rpc: reviewed <reason>` comment on the line
// above the @NiagaraRpc or on any line from it through the method name waives all of the RPC's rows; the reason is mandatory.
//
// Source is read as the sibling rules read it (javasrc.Tokens over the comment-stripped lines): a comment, a string
// or a text block is no code. Limits, false negatives: a generated Slot-o-Matic setter `c.setFoo(v)`, add(/remove(, a
// write in a helper the RPC calls, a check in a helper not named above. False positives: a reflection `m.invoke(o,
// args)` or a JDBC `setInt(i, v)` with a variable index; waive those. The rule reads the *.js files of the root only
// for the `baja.rpc(` call, as tokens (a comment, string or template is no call); a call in another root is not seen.
package niagararpc

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-niagara-rpc"

// Spec registers the rule with the argument contract of toolbelt/lint-niagara-rpc.sh: [--strict] [--] <src-root>...,
// each root on its own (a WARN row is exit 1 under --strict). It selects *.java and *.js; a root with no *.java
// is the empty-root error, whatever *.js it holds.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:       rule{},
		Mode:       lint.Each,
		Usage:      "usage: lint-niagara-rpc.sh [--help] [--strict] <src-root>...",
		Select:     lint.Selection{Names: []string{"*.java", "*.js"}, Except: []string{"*.min.js"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java sources found (wrong path or empty source tree)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

const see = " (types/security.md §2.7)"

// CheckTree reports, per @NiagaraRpc method of every *.java file in walk order, its shape, its unrestricted
// permissions, its first unguarded write and a bare waiver.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	jsCall := firstBajaRpc(files)
	var out []lint.Finding
	java := 0
	for _, f := range files {
		if isJS(f) {
			continue
		}
		java++
		for _, fd := range checkFile(f, jsCall) {
			fd.Path = f.Path
			out = append(out, fd)
		}
	}
	if java == 0 {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

// checkFile is the rows of one Java file; jsCall is where the root's JS first calls baja.rpc( ("" when it does not).
func checkFile(f *javasrc.File, jsCall string) []lint.Finding {
	toks := javasrc.Tokens(f.Stripped())
	var out []lint.Finding
	for i := 0; i < len(toks); i++ {
		end, ok := rpcAnnotation(toks, i)
		if !ok {
			continue
		}
		annLine := toks[i].Line + 1
		annEnd := end // index of the token after the annotation
		perms := permissions(toks, i, annEnd)
		name, nameAt, sigEnd := signature(toks, annEnd)
		sigLine := annLine
		if nameAt >= 0 {
			sigLine = toks[nameAt].Line + 1
		}
		var rows []lint.Finding
		waived, bare := waiver(f.Raw, annLine, sigLine)
		if !waived && bare > 0 {
			rows = append(rows, lint.Finding{Line: bare, Msg: "waiver-no-reason: lint-niagara-rpc: reviewed needs a reason" + see})
		}
		if !waived {
			rows = append(rows, shapeRows(toks, i, annEnd, name, nameAt, jsCall)...)
			annWrite, nonLit := false, false
			for _, p := range perms {
				v := strings.TrimSpace(p.val)
				switch {
				case !p.lit:
					nonLit = true
				case v == "" || strings.EqualFold(v, "unrestricted"):
					rows = append(rows, lint.Finding{Line: p.line, Msg: "unrestricted: " + name + "() declares permissions = \"" + p.val + "\": no framework permission check runs" + see})
				case strings.ContainsAny(v, "wW"):
					annWrite = true
				}
			}
			if sigEnd >= 0 && toks[sigEnd].Text == "{" && !annWrite && !nonLit {
				close := closeOf(toks, sigEnd)
				if w, call := firstWrite(toks, sigEnd+1, close); w >= 0 && !guarded(toks, sigEnd+1, w) {
					rows = append(rows, lint.Finding{Line: toks[w].Line + 1, Msg: "unguarded-write: " + name + "() calls " + call + "() with no OPERATOR_WRITE check before it" + see})
				}
			}
		}
		out = append(out, rows...)
		if sigEnd > i {
			i = sigEnd // the signature and the body hold no further annotation worth rescanning from inside the annotation
		} else {
			i = annEnd - 1
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Line < out[b].Line })
	return out
}

// shapeRows are the no-context FAIL and the no-box-transport WARN of the RPC whose annotation is toks[at:annEnd).
func shapeRows(toks []javasrc.Token, at, annEnd int, name string, nameAt int, jsCall string) []lint.Finding {
	var out []lint.Finding
	if nameAt >= 0 && toks[nameAt+1].Text == "(" {
		if why, line, ok, judged := lastParam(toks, nameAt+1); judged && !ok {
			out = append(out, lint.Finding{Line: line, Msg: "no-context: " + name + "() " + why + ": " + contextTail})
		}
	}
	if names, line, judged := transports(toks, at, annEnd); judged && jsCall != "" && !hasBox(names) {
		lists := "no transport"
		if len(names) > 0 {
			lists = "transports " + strings.Join(names, ", ")
		}
		out = append(out, lint.Finding{Severity: lint.Warn, Line: line, Msg: "no-box-transport: " + name + "() lists " + lists + " but " + jsCall +
			" calls baja.rpc(, which goes over BOX: the station finds no matching transport for it (NiagaraRpcUtil.java:181-187; types/data-access.md §8)"})
	}
	return out
}
