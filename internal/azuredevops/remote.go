package azuredevops

import (
	"fmt"
	"strings"
)

// RemoteInfo holds parsed Azure DevOps remote URL components.
type RemoteInfo struct {
	Org     string
	Project string
	Repo    string
}

// ParseAzureDevOpsRemote extracts org, project, and repo from an Azure DevOps git URL.
//
// Supports:
//   - https://dev.azure.com/{org}/{project}/_git/{repo}
//   - https://{org}.visualstudio.com/{project}/_git/{repo}
//   - git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
func ParseAzureDevOpsRemote(raw string) (*RemoteInfo, error) {
	raw = strings.TrimSpace(raw)

	if info := parseDevAzureCom(raw); info != nil {
		return info, nil
	}
	if info := parseVisualStudio(raw); info != nil {
		return info, nil
	}
	if info := parseSSH(raw); info != nil {
		return info, nil
	}

	return nil, fmt.Errorf("azuredevops: not an Azure DevOps URL: %s", raw)
}

func parseDevAzureCom(raw string) *RemoteInfo {
	const prefix = "dev.azure.com/"
	idx := strings.Index(raw, prefix)
	if idx < 0 {
		return nil
	}
	rest := raw[idx+len(prefix):]
	rest = strings.TrimSuffix(rest, ".git")
	rest = strings.TrimSuffix(rest, "/")

	parts := strings.SplitN(rest, "/", 4)
	if len(parts) < 3 {
		return nil
	}
	org := parts[0]
	project := parts[1]
	if parts[2] != "_git" {
		return nil
	}
	repo := ""
	if len(parts) >= 4 {
		repo = parts[3]
	}
	if org == "" || project == "" {
		return nil
	}
	return &RemoteInfo{Org: org, Project: project, Repo: repo}
}

func parseVisualStudio(raw string) *RemoteInfo {
	idx := strings.Index(raw, ".visualstudio.com/")
	if idx < 0 {
		return nil
	}
	org := ""
	dotIdx := strings.Index(raw, ".visualstudio.com")
	if dotIdx > 0 {
		prefix := raw[:dotIdx]
		if last := strings.LastIndex(prefix, "/"); last >= 0 {
			org = prefix[last+1:]
		} else if last := strings.LastIndex(prefix, "@"); last >= 0 {
			org = prefix[last+1:]
		} else {
			org = prefix
		}
	}
	rest := raw[idx+len(".visualstudio.com/"):]
	rest = strings.TrimSuffix(rest, ".git")
	rest = strings.TrimSuffix(rest, "/")

	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return nil
	}
	project := parts[0]
	if parts[1] != "_git" && !strings.HasPrefix(rest, project+"/_git/") {
		return nil
	}
	repo := ""
	if len(parts) >= 3 {
		repo = parts[2]
	}
	if org == "" || project == "" {
		return nil
	}
	return &RemoteInfo{Org: org, Project: project, Repo: repo}
}

func parseSSH(raw string) *RemoteInfo {
	const prefix = "ssh.dev.azure.com:v3/"
	idx := strings.Index(raw, prefix)
	if idx < 0 {
		return nil
	}
	rest := raw[idx+len(prefix):]
	rest = strings.TrimSuffix(rest, ".git")
	rest = strings.TrimSuffix(rest, "/")

	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 {
		return nil
	}
	return &RemoteInfo{Org: parts[0], Project: parts[1], Repo: parts[2]}
}

// IsAzureDevOpsRemote reports whether the URL appears to be an Azure DevOps repo.
func IsAzureDevOpsRemote(raw string) bool {
	_, err := ParseAzureDevOpsRemote(raw)
	return err == nil
}
