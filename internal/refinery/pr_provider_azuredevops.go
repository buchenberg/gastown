package refinery

import (
	"fmt"

	"github.com/steveyegge/gastown/internal/azuredevops"
	"github.com/steveyegge/gastown/internal/git"
)

type azuredevopsPRProvider struct {
	info *azuredevops.RemoteInfo
}

func newAzureDevOpsPRProvider(g *git.Git) (PRProvider, error) {
	remoteURL, err := g.RemoteURL("origin")
	if err != nil {
		return nil, fmt.Errorf("azuredevops provider: failed to get origin remote URL: %w", err)
	}
	info, err := azuredevops.ParseAzureDevOpsRemote(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("azuredevops provider: %w", err)
	}
	return &azuredevopsPRProvider{info: info}, nil
}

func (p *azuredevopsPRProvider) FindPullRequest(branch, _ string, _ int, headSHA string) (*git.PullRequestInfo, error) {
	return azuredevops.FindPullRequest(p.info, branch, headSHA)
}

func (p *azuredevopsPRProvider) IsPRApproved(pr *git.PullRequestInfo) (bool, error) {
	if pr == nil || pr.Number == 0 {
		return false, fmt.Errorf("azuredevops: pull request identity is missing")
	}
	return azuredevops.IsPRApproved(p.info, pr.Number)
}

func (p *azuredevopsPRProvider) MergePR(pr *git.PullRequestInfo, method string) (string, error) {
	if pr == nil || pr.Number == 0 {
		return "", fmt.Errorf("azuredevops: pull request identity is missing")
	}
	return azuredevops.MergePR(p.info, pr.Number, method)
}
