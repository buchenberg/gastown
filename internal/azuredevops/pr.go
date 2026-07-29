package azuredevops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/steveyegge/gastown/internal/git"
)

// TokenEnv is the environment variable for the Azure DevOps PAT.
const TokenEnv = "AZURE_DEVOPS_EXT_PAT"

// OrgEnv is the default org URL override.
const OrgEnv = "AZURE_DEVOPS_ORG"

func requireToken() error {
	if os.Getenv(TokenEnv) == "" {
		return fmt.Errorf("%s is required for Azure DevOps PR operations", TokenEnv)
	}
	return nil
}

// OrgURL returns the full org URL for az devops commands.
// e.g., https://dev.azure.com/myorg
func (r *RemoteInfo) OrgURL() string {
	return "https://dev.azure.com/" + r.Org
}

// azRun executes az with the given arguments and returns stdout.
func azRun(args ...string) ([]byte, error) {
	cmd := exec.Command("az", args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("az %s failed: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr.String()), err)
	}
	return out, nil
}

// ADOPullRequest mirrors the JSON returned by az repos pr show / list.
type ADOPullRequest struct {
	PullRequestID int    `json:"pullRequestId"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	SourceRefName string `json:"sourceRefName"`
	ClosedDate    string `json:"closedDate"`
	LastMergeSourceCommit *struct {
		CommitID string `json:"commitId"`
	} `json:"lastMergeSourceCommit"`
	Repository *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"repository"`
	URL string `json:"url"`
}

func (p *ADOPullRequest) toInfo(org string) *git.PullRequestInfo {
	info := &git.PullRequestInfo{
		Number:       p.PullRequestID,
		URL:          p.URL,
		State:        mapStatus(p.Status),
		MergedAt:     p.ClosedDate,
		LookupSource: "azuredevops",
	}
	if strings.HasPrefix(p.SourceRefName, "refs/heads/") {
		info.HeadRefName = strings.TrimPrefix(p.SourceRefName, "refs/heads/")
	}
	if p.LastMergeSourceCommit != nil {
		info.HeadSHA = strings.TrimSpace(p.LastMergeSourceCommit.CommitID)
	}
	if p.Repository != nil {
		info.BaseRepo = org + "/" + p.Repository.Name
	}
	return info
}

func mapStatus(s string) string {
	switch strings.ToUpper(s) {
	case "ACTIVE":
		return "OPEN"
	case "COMPLETED":
		return "MERGED"
	case "ABANDONED":
		return "CLOSED"
	default:
		return strings.ToUpper(s)
	}
}

// FindPullRequest returns the active PR for the given branch in the Azure DevOps repo.
func FindPullRequest(info *RemoteInfo, branch, headSHA string) (*git.PullRequestInfo, error) {
	if info.Org == "" || info.Project == "" || info.Repo == "" {
		return nil, fmt.Errorf("azuredevops: incomplete remote info (org=%q project=%q repo=%q)", info.Org, info.Project, info.Repo)
	}
	if err := requireToken(); err != nil {
		return nil, err
	}

	args := []string{
		"repos", "pr", "list",
		"--org", info.OrgURL(),
		"--project", info.Project,
		"--repository", info.Repo,
		"--source-branch", branch,
		"--status", "active",
		"--top", "1",
		"--output", "json",
	}
	out, err := azRun(args...)
	if err != nil {
		return nil, fmt.Errorf("azuredevops: find PR for branch %s in %s/%s: %w", branch, info.Project, info.Repo, err)
	}

	var prs []ADOPullRequest
	if err := json.Unmarshal(bytes.TrimSpace(out), &prs); err != nil {
		return nil, fmt.Errorf("azuredevops: parse pr list: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}

	pr := prs[0]
	result := pr.toInfo(info.Org)

	if headSHA != "" && result.HeadSHA != headSHA {
		return nil, fmt.Errorf("azuredevops: PR %d head SHA mismatch: expected %s, got %s", pr.PullRequestID, headSHA, result.HeadSHA)
	}
	return result, nil
}

// GetPullRequest returns a PR by ID.
func GetPullRequest(info *RemoteInfo, prID int) (*git.PullRequestInfo, error) {
	if err := requireToken(); err != nil {
		return nil, err
	}

	args := []string{
		"repos", "pr", "show",
		"--id", fmt.Sprintf("%d", prID),
		"--org", info.OrgURL(),
		"--output", "json",
	}
	out, err := azRun(args...)
	if err != nil {
		return nil, fmt.Errorf("azuredevops: get PR %d: %w", prID, err)
	}

	var pr ADOPullRequest
	if err := json.Unmarshal(bytes.TrimSpace(out), &pr); err != nil {
		return nil, fmt.Errorf("azuredevops: parse pr show %d: %w", prID, err)
	}
	return pr.toInfo(info.Org), nil
}

// IsPRApproved checks whether a PR has at least one approving reviewer.
func IsPRApproved(info *RemoteInfo, prID int) (bool, error) {
	if err := requireToken(); err != nil {
		return false, err
	}

	args := []string{
		"repos", "pr", "policy", "list",
		"--id", fmt.Sprintf("%d", prID),
		"--org", info.OrgURL(),
		"--output", "json",
	}
	out, err := azRun(args...)
	if err != nil {
		return false, fmt.Errorf("azuredevops: check approvals for PR %d: %w", prID, err)
	}

	var evaluations []struct {
		EvaluationID string `json:"evaluationId"`
		Status       string `json:"status"`
		IsBlocking   bool   `json:"isBlocking"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &evaluations); err != nil {
		return false, fmt.Errorf("azuredevops: parse pr policy list for PR %d: %w", prID, err)
	}

	allApproved := len(evaluations) > 0
	for _, eval := range evaluations {
		if eval.IsBlocking && eval.Status != "approved" {
			allApproved = false
			break
		}
	}
	return allApproved, nil
}

// MergePR merges a PR using the specified method (squash, mergeCommit, rebase).
// Returns the merge commit SHA on success.
func MergePR(info *RemoteInfo, prID int, method string) (string, error) {
	if err := requireToken(); err != nil {
		return "", err
	}

	strategy := mapMergeStrategy(method)

	args := []string{
		"repos", "pr", "update",
		"--id", fmt.Sprintf("%d", prID),
		"--status", "completed",
		"--merge-strategy", strategy,
		"--org", info.OrgURL(),
		"--output", "json",
	}
	out, err := azRun(args...)
	if err != nil {
		return "", fmt.Errorf("azuredevops: merge PR %d: %w", prID, err)
	}

	var pr ADOPullRequest
	if err := json.Unmarshal(bytes.TrimSpace(out), &pr); err != nil {
		return "", fmt.Errorf("azuredevops: parse merge response for PR %d: %w", prID, err)
	}
	if pr.LastMergeSourceCommit != nil {
		return pr.LastMergeSourceCommit.CommitID, nil
	}
	return "", nil
}

func mapMergeStrategy(method string) string {
	switch strings.ToLower(method) {
	case "rebase":
		return "rebase"
	case "merge", "mergecommit":
		return "noFastForward"
	case "squash":
		fallthrough
	default:
		return "squash"
	}
}
