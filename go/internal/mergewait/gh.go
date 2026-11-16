package mergewait

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Gh runs one `gh <args>` and returns its stdout; a non-zero exit is an error carrying its stderr.
type Gh func(args ...string) (string, error)

// ExecGh is the real gh, looked up on PATH; repo, when set, is passed as GH_REPO (gh reads it for the
// pr commands and for the {owner}/{repo} placeholders of gh api).
func ExecGh(repo string, env []string) Gh {
	return func(args ...string) (string, error) {
		cmd := exec.Command("gh", args...)
		cmd.Env = env
		if repo != "" {
			cmd.Env = append(cmd.Env, "GH_REPO="+repo)
		}
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		if err := cmd.Run(); err != nil {
			return out.String(), fmt.Errorf("gh %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
		}
		return out.String(), nil
	}
}

// pr is the part of `gh pr view --json` the wait reads.
type pr struct {
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
	HeadRefOid  string `json:"headRefOid"`
	BaseRefName string `json:"baseRefName"`
	MergeCommit *struct {
		Oid string `json:"oid"`
	} `json:"mergeCommit"`
}

// mergeOid is the merge commit of a merged PR ("" when gh gave none).
func (p pr) mergeOid() string {
	if p.MergeCommit == nil {
		return ""
	}
	return p.MergeCommit.Oid
}

// checkRun is one entry of the check-runs API of a commit.
type checkRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	DetailsURL string `json:"details_url"`
}

func viewPR(gh Gh, number string) (pr, error) {
	var p pr
	err := ghJSON(gh, &p, "pr", "view", number, "--json", "state,isDraft,headRefOid,baseRefName,mergeCommit")
	if err == nil && (p.HeadRefOid == "" || p.BaseRefName == "") {
		err = fmt.Errorf("gh pr view %s: no head commit or base branch in the reply", number)
	}
	return p, err
}

// behindBy is how many commits of base the head lacks.
func behindBy(gh Gh, base, sha string) (int, error) {
	var c struct {
		BehindBy *int `json:"behind_by"`
	}
	if err := ghJSON(gh, &c, "api", "repos/{owner}/{repo}/compare/"+base+"..."+sha); err != nil {
		return 0, err
	}
	if c.BehindBy == nil {
		return 0, fmt.Errorf("gh api compare %s...%s: no behind_by in the reply", base, sha)
	}
	return *c.BehindBy, nil
}

// checkRuns lists the latest check run of every check of the exact commit sha.
func checkRuns(gh Gh, sha string) ([]checkRun, error) {
	var r struct {
		CheckRuns []checkRun `json:"check_runs"`
	}
	err := ghJSON(gh, &r, "api", "repos/{owner}/{repo}/commits/"+sha+"/check-runs?filter=latest&per_page=100")
	return r.CheckRuns, err
}

func ghJSON(gh Gh, v any, args ...string) error {
	out, err := gh(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("gh %s: unreadable reply: %v", strings.Join(args, " "), err)
	}
	return nil
}
