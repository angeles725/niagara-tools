package readtools

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// The optional parts of n4_read_history: summary=true and cov_hint=true, both computed from ONE
// machine-encoded read of the range (asking for both costs one read, not two).

// historyExtras holds what the reply adds for summary and cov_hint, and their latency.
type historyExtras struct {
	summary, cov         any
	summaryNote, covNote string
	wantCov              bool
	ms                   int
}

// historyExtras reads the range once when summary or cov_hint asks for it.
func (t *Tools) historyExtras(args *pyjson.Object, id, params string, timeout int, total any) historyExtras {
	wantSummary := truthyArg(args, "summary")
	ex := historyExtras{wantCov: truthyArg(args, "cov_hint")}
	if !wantSummary && !ex.wantCov {
		return ex
	}
	read := t.readSummaryRows(id, params, intArg(args, "summary_max_rows", summaryDefaultRows), timeout)
	ex.ms = read.elapsed
	if wantSummary {
		var ms int
		ex.summary, ex.summaryNote, ms = t.summaryOf(id, params, read, timeout, total)
		ex.ms += ms
	}
	if ex.wantCov {
		ex.cov, ex.covNote = covHint(read)
	}
	return ex
}

// set adds summary, summary_note, cov_hint and cov_hint_note to out, each only when asked for.
func (ex historyExtras) set(out *pyjson.Object) {
	if ex.summary != nil || ex.summaryNote != "" {
		out.Set("summary", ex.summary)
	}
	if ex.summaryNote != "" {
		out.Set("summary_note", ex.summaryNote)
	}
	if ex.wantCov {
		out.Set("cov_hint", ex.cov)
		if ex.covNote != "" {
			out.Set("cov_hint_note", ex.covNote)
		}
	}
}

// truthyArg is whether args holds key with a true value.
func truthyArg(args *pyjson.Object, key string) bool {
	v, ok := args.Get(key)
	return ok && pyjson.Truthy(v)
}
