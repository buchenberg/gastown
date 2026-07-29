package azuredevops

import (
	"testing"
)

func TestParseAzureDevOpsRemote(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantOrg string
		wantProj string
		wantRepo string
		wantErr  bool
	}{
		{
			name:    "dev.azure.com HTTPS",
			url:     "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "dev.azure.com HTTPS with .git",
			url:     "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "dev.azure.com HTTPS trailing slash",
			url:     "https://dev.azure.com/myorg/myproject/_git/myrepo/",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "visualstudio.com HTTPS",
			url:     "https://myorg.visualstudio.com/myproject/_git/myrepo",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "visualstudio.com HTTPS with .git",
			url:     "https://myorg.visualstudio.com/myproject/_git/myrepo.git",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "SSH format",
			url:     "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "SSH format with .git",
			url:     "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo.git",
			wantOrg: "myorg",
			wantProj: "myproject",
			wantRepo: "myrepo",
		},
		{
			name:    "not azure devops URL (GitHub)",
			url:     "https://github.com/owner/repo.git",
			wantErr: true,
		},
		{
			name:    "not azure devops URL (Bitbucket)",
			url:     "git@bitbucket.org:workspace/repo.git",
			wantErr: true,
		},
		{
			name:    "empty",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseAzureDevOpsRemote(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.url)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if info.Org != tt.wantOrg {
				t.Errorf("Org = %q, want %q", info.Org, tt.wantOrg)
			}
			if info.Project != tt.wantProj {
				t.Errorf("Project = %q, want %q", info.Project, tt.wantProj)
			}
			if info.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", info.Repo, tt.wantRepo)
			}
		})
	}
}

func TestIsAzureDevOpsRemote(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://dev.azure.com/org/proj/_git/repo", true},
		{"https://github.com/owner/repo", false},
		{"https://bitbucket.org/workspace/repo", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := IsAzureDevOpsRemote(tt.url); got != tt.want {
				t.Errorf("IsAzureDevOpsRemote(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestRemoteInfo_OrgURL(t *testing.T) {
	r := &RemoteInfo{Org: "myorg"}
	if got := r.OrgURL(); got != "https://dev.azure.com/myorg" {
		t.Errorf("OrgURL() = %q, want %q", got, "https://dev.azure.com/myorg")
	}
}
