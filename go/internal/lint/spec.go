package lint

import (
	"errors"
	"strings"
)

// Spec registers a rule with the command-line contract of the toolbelt wrapper that runs it alone.
// Exactly one of Rule, Tree and Root is set, and that value implements only its own interface;
// Command panics on any other spec (a registration bug).
type Spec struct {
	Rule      Rule      // a per-file rule, or
	Tree      TreeRule  // a whole-tree rule, or
	Root      RootRule  // a rule that reads its root itself
	Mode      Mode      // how the rule's arguments are parsed when it runs alone
	Usage     string    // printed verbatim on a usage error when it runs alone
	UsageExit int       // the exit code of that usage error; 0 means 3
	Select    Selection // the files under a root the rule reads
	// ShortScanError words an unreadable file as awk_lint_each did ("<lint>: cannot scan <path>: <reason>")
	// instead of awk_lint_run's awk error text (lint-agent-on-shape).
	ShortScanError bool
	// CheckFirst prints a finding's check column before its severity (`<check>  <SEV>  <path>:<line>  <msg>`),
	// the row of lint-wb-threading; the empty-root ERROR row keeps the standard shape.
	CheckFirst bool
	// Joined gives a TreeRule the files of every root in one CheckTree call, after the last root, for a lint
	// whose report spans its roots (lint-manual-labels: one UI haystack and one SUMMARY row over every <ui-dir>).
	Joined bool
	// EmptyOK opts a rule out of the empty-root error for a root where it selects no file: no ERROR row,
	// exit 0. Only for a lint whose contract says so (lint-lexicon-ascii LA5: a module with no *.lexicon
	// has nothing to check, and report-module runs it on every module); a walk error still exits 3. A RootRule
	// that returns no ErrEmptyRoot for such a root (lint-jasmine-ux) sets it to record the same exception.
	EmptyOK bool
	// OptionalInAll makes a root where the rule selects no file clean for it under n4lint --all (no ERROR
	// row), for a rule whose files belong to one profile only (lint-spa-poll-no-recovery reads the *.html/*.js
	// of a -ux root, and --all also runs on -rt roots that have none). Run by name or in a rule list it keeps
	// the empty-root error.
	OptionalInAll bool
	// FileRoot lets a RootRule run alone take a regular file it accepts as its root (a template module
	// .jar), as well as a directory. Nil, the default, accepts none: a file root is "not a directory" (exit 3).
	FileRoot func(path string) bool
	// ModuleRoot marks a rule whose roots are module roots, not source roots (lint-duplicate-helper reads
	// every part's src/): n4lint --all, which runs on source roots, leaves it out as it leaves out a
	// RootRule; it runs by name or in a rule list.
	ModuleRoot bool
	// Option is the value option Mode Valued parses (e.g. "--wiring-map"), or the noun of the file operand of
	// Mode FileFirst ("manual"); the rule reads its value (Args.Option) by implementing Configurer.
	Option string
	// Options are further Mode Valued value options, each repeatable (every value in Args.Values), and
	// ValuedStrict accepts --strict in Mode Valued.
	Options      []string
	ValuedStrict bool
	// Flags are Mode Valued boolean flags (e.g. "--no-default-allow"), anywhere in the arguments; the rule
	// reads Args.Flags by implementing Configurer.
	Flags []string
	// ValuedNamed gives Mode Valued lint-write-path's named errors instead of the usage line alone: "<name>:
	// unknown option: <arg>", "<name>: <option> requires a file argument", "<name>: unexpected argument:
	// <arg>"; and an empty argument leaves the root unset, so a later one takes its place (`[ -z "$ROOT" ]`).
	ValuedNamed bool
	// ContinueOnWalkError keeps scanning the files that were listed when the walk could not read a
	// directory (env line, rows, next root, exit 3), as awk_lint_run does. Its zero value stops the rule
	// (env line only, no rows, no later root, exit 3), as scan_files_or_exit does. Set it only with a
	// comment naming the reason, normally that the bash lint being ported used awk_lint_run.
	ContinueOnWalkError bool
	// EmptyError shapes the row of a root where the rule selects no file. Such a root is always an
	// environment error (a wrong path or an empty scaffold, never a clean pass), as the K20 guard of the
	// bash lints made it: the rule prints "ERROR  <check>  <root>  <msg>" on stdout and stops (no later
	// root, exit 3). It is decided per root after the walk, so a walk error wins and prints no ERROR row.
	// There is no opt-out: every registered rule reads source files, and none has a meaning on a root
	// without them. A RootRule decides emptiness itself and returns ErrEmptyRoot; its spec sets Msg. A TreeRule may
	// return ErrEmptyRoot too, when the files it selects hold nothing it checks.
	EmptyError EmptyRow
	// BarePathRow prints a file-level finding (Line 0) as "<path>  <msg>", two spaces and no colon, as
	// lint-structure printed its file and directory rows; a line finding keeps "<path>:<line>  <msg>".
	BarePathRow bool
	// BareUsage prints only Usage on an unknown flag (no "<lint>: unknown flag: <flag>" line), as a bash
	// lint with its own `-*) usage; exit 3` case does; the exit code stays UsageExit.
	BareUsage bool
	// StrictFails prints a WARN row as FAIL under --strict, for a bash lint that rewrote the severity
	// column itself (`v = strict ? "FAIL" : "WARN"`). The exit code is the same either way (1).
	StrictFails bool
	// RelPaths prints a per-file rule's row path with the "<root>/" prefix removed, as a bash lint that
	// printed ${file#"$root/"} did (a root given with a trailing "/" keeps the whole path, as there). Env
	// lines on stderr keep the walk path.
	RelPaths bool
	// IntFlags are the "<name> <n>" options (n a non-negative integer, e.g. "--max-class 800") the rule takes
	// when it runs alone, as a bash lint's AL_FLAG_HOOK took them: among the leading flags, before "--" and
	// the roots, any order with --strict, the last one given winning. A value that is missing or not all
	// digits prints "<lint>: <name> needs a non-negative integer" and the usage line (exit 3); an unknown flag
	// before it stays Parse's unknown-flag error. Configure (required with IntFlags) builds the rule from
	// the values, every flag not given at its Default; in a rule list or under --all the rule runs as
	// registered (its defaults).
	IntFlags  []IntFlag
	Configure func(vals map[string]int) Rule
}

// IntFlag is one Spec.IntFlags option: its name as typed ("--max-class") and its value when not given.
type IntFlag struct {
	Name    string
	Default int
}

// EmptyRow is the ERROR row of Spec.EmptyError.
type EmptyRow struct {
	Check string // the check column; empty shows the rule ID
	Msg   string // the message; empty shows "no <pattern>[ or <pattern>...] sources found (wrong path or empty source tree)"
}

// emptyMsg is the message of the spec's ERROR row on a root where it selects no file.
func (sp Spec) emptyMsg() string {
	if sp.EmptyError.Msg != "" {
		return sp.EmptyError.Msg
	}
	return "no " + strings.Join(sp.Select.Names, " or ") + " sources found (wrong path or empty source tree)"
}

// ID is the registered rule's ID.
func (sp Spec) ID() string { return sp.rule().ID() }

func (sp Spec) severity() Severity { return sp.rule().Severity() }

// rule is the one rule value the spec sets.
func (sp Spec) rule() Configured {
	switch {
	case sp.Tree != nil:
		return sp.Tree
	case sp.Root != nil:
		return sp.Root
	}
	return sp.Rule
}

func (sp Spec) usageExit() int {
	if sp.UsageExit != 0 {
		return sp.UsageExit
	}
	return 3
}

// validate enforces that a spec holds exactly one rule kind, and a value of only that kind.
func (sp Spec) validate() error {
	if count(sp.Rule != nil, sp.Tree != nil, sp.Root != nil) != 1 {
		return errors.New("lint: a Spec sets exactly one of Rule, Tree and Root")
	}
	r := sp.rule()
	_, isRule := r.(Rule)
	_, isTree := r.(TreeRule)
	_, isRoot := r.(RootRule)
	if count(isRule, isTree, isRoot) != 1 {
		return errors.New("lint: rule " + r.ID() + " implements more than one of Rule, TreeRule and RootRule")
	}
	if sp.Mode == FileFirst && sp.Option == "" {
		return errors.New("lint: rule " + r.ID() + " has Mode FileFirst without an Option noun")
	}
	if len(sp.IntFlags) > 0 && (sp.Configure == nil || sp.Rule == nil) {
		return errors.New("lint: rule " + r.ID() + " has IntFlags without a per-file Rule and Configure")
	}
	return nil
}

func count(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}
