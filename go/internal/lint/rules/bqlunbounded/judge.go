package bqlunbounded

// Reading one bql: query out of a literal and judging whether it is bounded.

import (
	"regexp"
	"strings"
)

var (
	bqlAt    = regexp.MustCompile(`(?i)(^|[|\s])bql:`)
	selectRe = regexp.MustCompile(`(?i)^(select|from)\b`)
	fromRe   = regexp.MustCompile(`(?i)\sfrom\s`)
	topRe    = regexp.MustCompile(`(?i)(^|\s)top\s+(\d+|%[sd]|\?|\{\d*\})(\s|,|$)`)
	orderRe  = regexp.MustCompile(`(?i)\border\s+by\b`)
	whereRe  = regexp.MustCompile(`(?i)\bwhere\b`)
	orRe     = regexp.MustCompile(`(?i)\bor\b`)
	stampRe  = regexp.MustCompile(`(?i)\btimestamp\s*(>=|<=|>|<|=|in\b)`)
	keyRe    = regexp.MustCompile(`(?i)\b(uuid|source)\s*=`)
	extentRe = regexp.MustCompile(`\bfrom\s+(openAlarms|ackPendingAlarms)\b`) // case-sensitive: AlarmDbConnection.doBqlQuery uses equals
	periodRe = regexp.MustCompile(`(?i)[?;](period|start|end)=`)
	quoted   = regexp.MustCompile(`'[^']*'`)
	spaces   = regexp.MustCompile(`\s+`)
)

// query is a bql: query found in a literal: the ORD before it and the select text, whitespace collapsed.
type query struct{ ord, text string }

// find reads the query of a literal: the text after the first bql: up to a | outside single quotes. ok is false
// when the literal holds no bql: or what follows is not a select (prose that names the scheme).
func find(lit string) (q query, ok bool) {
	loc := bqlAt.FindStringIndex(lit)
	if loc == nil {
		return q, false
	}
	text := spaces.ReplaceAllString(strings.TrimSpace(cut(lit[loc[1]:])), " ")
	if !selectRe.MatchString(text) {
		return q, false
	}
	return query{ord: lit[:loc[0]], text: text}, true
}

// built reports whether a literal that is added to something else is a query put together at run time: its first
// bql: is followed by a select (a query whose tail is added) or by nothing (the select is the added part). Prose
// that names the scheme ("expected a bql: prefix" + name) is not.
func built(lit string) bool {
	loc := bqlAt.FindStringIndex(lit)
	if loc == nil {
		return false
	}
	rest := strings.TrimSpace(cut(lit[loc[1]:]))
	return rest == "" || selectRe.MatchString(rest)
}

// cut is s up to its first | outside single quotes (a |view: suffix is not part of the query).
func cut(s string) string {
	in := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\'':
			in = !in
		case s[i] == '|' && !in:
			return s[:i]
		}
	}
	return s
}

// kind is the ORD scheme whose rows the query reads: "history" for a history:<path> segment (history: alone lists
// the histories themselves), "alarm" for an alarm: one, "" for any other ORD.
func kind(ord string) string {
	for _, seg := range strings.Split(ord, "|") {
		seg = strings.ToLower(strings.TrimSpace(seg))
		switch {
		case strings.HasPrefix(seg, "history:") && strings.TrimSpace(seg[len("history:"):]) != "":
			return "history"
		case strings.HasPrefix(seg, "alarm:"):
			return "alarm"
		}
	}
	return ""
}

// verdict is a finding the query earns: its check name, the text after the quoted query and the doc section.
type verdict struct{ check, text, see string }

// judge returns the verdict of a query, or false when it is bounded. A query is bounded by a top <n>; a history
// query also by a timestamp window (?period=, start=, end= in the ORD, or a timestamp predicate), an alarm query
// also by an index path (from openAlarms or ackPendingAlarms, uuid =, source =, a timestamp range). Any other
// query with an order by and no top is flagged too: order by loads every row before any cap.
func judge(q query) (verdict, bool) {
	text := quoted.ReplaceAllString(q.text, "''")
	head := text
	if loc := fromRe.FindStringIndex(text); loc != nil {
		head = text[:loc[0]]
	}
	if topRe.MatchString(head) {
		return verdict{}, false
	}
	where := ""
	if loc := whereRe.FindStringIndex(text); loc != nil {
		where = text[loc[1]:]
		if o := orderRe.FindStringIndex(where); o != nil {
			where = where[:o[0]]
		}
	}
	ranged := stampRe.MatchString(where) && !orRe.MatchString(where)
	sorted := orderRe.MatchString(text)
	var v verdict
	switch kind(q.ord) {
	case "history":
		if ranged || periodRe.MatchString(q.ord) {
			return orderOnly(sorted)
		}
		v = verdict{"history-no-window", " has no top <n> and no timestamp window (?period=, start=, end= or a timestamp predicate): it scans the whole history", "types/queries.md §3, §4"}
	case "alarm":
		if ranged || extentRe.MatchString(text) || (keyRe.MatchString(where) && !orRe.MatchString(where)) {
			return orderOnly(sorted)
		}
		v = verdict{"alarm-unindexed", " has no top <n> and no index path (from openAlarms, from ackPendingAlarms, uuid =, source = or a timestamp range): it scans the whole alarm database", "types/queries.md §3, §6"}
	default:
		v = verdict{"no-top", " has no top <n>: it returns every row of its extent", "types/queries.md §3"}
	}
	if sorted {
		v.text += "; order by sorts every row first"
	}
	return v, true
}

// orderOnly is the verdict of a query that is bounded but sorts without a top.
func orderOnly(sorted bool) (verdict, bool) {
	if !sorted {
		return verdict{}, false
	}
	return verdict{"order-by-no-top", " has an order by and no top <n>: order by loads every row before any cap", "types/queries.md §3"}, true
}

// snippet is the query as the row quotes it: at most 60 runes.
func snippet(q query) string {
	r := []rune(q.text)
	if len(r) > 60 {
		return string(r[:57]) + "..."
	}
	return q.text
}
