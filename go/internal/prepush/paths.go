package prepush

import "regexp"

// Data only: every path class, trailer and tool location the gates read. All patterns are matched
// against one line (a repository-relative path, a log line, a resolver row), never a whole output.

// kitDir is the kit, relative to the toplevel; the three wrappers the hook runs live in its toolbelt.
const (
	kitDir       = "build-n4-module-kit"
	citeResolver = kitDir + "/toolbelt/kit-cite-resolve.sh"
	kitStateTool = "/toolbelt/kit-state.sh"         // under the kit
	sweepTool    = "/toolbelt/sweep-build-state.sh" // under the kit
)

// ciLocalScript is the CI replay whose --changed mode lints the pushed range (cilints.go).
const ciLocalScript = "scripts/ci-local.sh"

// newBranchBases are the refs a new remote branch is measured from, in order.
var newBranchBases = []string{"origin/main", "origin/HEAD"}

// The retro gate's path classes.
var (
	// build-relevant = kit files EXCEPT BUILD-STATE(.src).md, ledger.d/** and retros/**, PLUS scripts/**,
	// PLUS go/** except the built go/bin/ (the toolbelt wrappers exec the Go tools, so a go/ change changes
	// lint behavior like a toolbelt script change). [ev: retro go-javasrc-scanner Δ1]
	relevantRoots = regexp.MustCompile(`^(build-n4-module-kit/|scripts/|go/)`)
	notRelevant   = []*regexp.Regexp{
		regexp.MustCompile(`^go/bin/`),
		regexp.MustCompile(`^build-n4-module-kit/BUILD-STATE(\.src)?\.md$`),
		regexp.MustCompile(`^build-n4-module-kit/ledger\.d/`),
		regexp.MustCompile(`^build-n4-module-kit/retros/`),
	}
	// ledger data = the files the content half checks. [ev: retro r10-path-keyed-gates-skip-data]
	ledgerData = regexp.MustCompile(`^build-n4-module-kit/(BUILD-STATE(\.src)?\.md|ledger\.d/|retros/)`)
	// a ledger update = a ledger.d/ fragment or a BUILD-STATE.src.md edit (BUILD-STATE.md is generated from them)
	ledgerUpdate = regexp.MustCompile(`^build-n4-module-kit/(BUILD-STATE\.src\.md|ledger\.d/[^/]+\.md)$`)
	// a retro = a retros/*.md file other than the generated INDEX.md
	retroFile  = regexp.MustCompile(`^build-n4-module-kit/retros/[^/]+\.md$`)
	retroIndex = regexp.MustCompile(`/INDEX\.md$`)
)

// retroPatchPathspec selects the retro diffs a promotion's header flip is read from.
const retroPatchPathspec = "build-n4-module-kit/retros/*.md"

// The retro trailers (anywhere in a commit message line) and the header lines of a flip in a patch.
var (
	trivialTrailer   = regexp.MustCompile(`Retro: none \(trivial:`)
	promotionTrailer = regexp.MustCompile(`Retro: promotion \(folds`)
)

const (
	patchFileHeader = "diff --git "
	pendingRemoved  = "-<!-- review-status: pending"
	foldedAdded     = "+<!-- review-status: folded"
)

// The live doctrine set: the cards whose every evidence pointer must resolve under `kit-cite-resolve
// --strict`, and the resolver itself (a change to it re-checks the set). These two patterns are the
// only definition (doctrine_test.go pins which paths they take); no shell copy exists.
var (
	liveDoctrine = regexp.MustCompile(`^(build-n4-module-kit/(types/[^/]+|METHODOLOGY|BUILD-LOOP|ORCHESTRATION|SOURCES|build-verify|corpus-index|README|skill/SKILL|toolbelt/INDEX)\.md|go/README\.md|CONTRIBUTING\.md|README\.md)$`)
	liveCiteTool = regexp.MustCompile(`^(go/internal/(citeparse|citeresolve)/|go/cmd/kit-cite-resolve/|build-n4-module-kit/toolbelt/kit-cite-resolve\.sh$)`)
)

// resolverPaths are what the resolver reads or runs besides the cards: the working tree must hold the
// pushed bytes of each.
var resolverPaths = []string{
	"go/internal/citeparse", "go/internal/citeresolve", "go/cmd/kit-cite-resolve", citeResolver,
}

// resolverArgs come before the cards on the resolver's command line.
var resolverArgs = []string{"--kit", kitDir, "--format", "tsv", "--strict"}

// The resolver's output: a TSV row has at least findingMinFields fields and its status in the sixth;
// these statuses are not findings. absentRoot is the one failure that skips the check.
const (
	findingMinFields = 7
	statusField      = 5
	maxFindingRows   = 20
	toolFailureTail  = 5
)

var (
	notFinding = regexp.MustCompile(`^(RESOLVED|HISTORICAL_DRIFT|UNVERIFIABLE_KIND|SKIPPED_TEMPLATE)$`)
	absentRoot = regexp.MustCompile(`^kit-cite-resolve: cannot read (research|modules|clients|memory) root (.*): no such file or directory$`)
)

// The client denylist: $NIAGARA_CLIENT_DENYLIST, else this path under $HOME. It lives outside every repo
// and is never read here, only checked for existence.
const (
	clientGuardTool   = "client-guard"
	denylistEnv       = "NIAGARA_CLIENT_DENYLIST"
	denylistUnderHome = "/.config/niagara-private/client-denylist.txt"
)
