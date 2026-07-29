# Azure DevOps VCS Provider

## Goal

Add Azure DevOps as a configurable VCS provider in gas town, using the `az devops` CLI (same pattern as `gh` CLI for GitHub).

## Architecture Decisions

### Scope: Incremental, following the Bitbucket pattern

The existing codebase has **no unified VCS interface**. GitHub ops are methods on `*git.Git`. Bitbucket has its own package with partial coverage. We follow Bitbucket's pattern: add `internal/azuredevops/`, implement the `PRProvider` interface for refinery merge-queue, then add individual `az` dispatch at each non-refinery call site.

### Token: `AZURE_DEVOPS_EXT_PAT`

The `az devops` CLI authenticates via `AZURE_DEVOPS_EXT_PAT` env var (Personal Access Token). No `az login` needed for CLI operations.

### Remote URL format

Azure DevOps repos use two URL patterns:
- `https://dev.azure.com/{org}/{project}/_git/{repo}`
- `https://{org}.visualstudio.com/{project}/_git/{repo}`
- `git@ssh.dev.azure.com:v3/{org}/{project}/{repo}`

We parse the `org` and `repo` from these, similar to how `githubRepoFromRemoteURL()` parses `github.com/owner/repo`.

### PR numbering

Azure DevOps PRs have integer IDs scoped to the repo. The `az repos pr` commands use `--id <number>` for PR operations.

## Implementation Tasks

### Phase 1: Infrastructure

**1.1 Docker: Install Azure CLI + devops extension**
- File: `Dockerfile`
- Add after the system package installs:
  ```dockerfile
  RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash
  RUN az extension add --name azure-devops
  ```
- Verify: `az devops --help` exits 0

**1.2 Create `internal/azuredevops/` package**
- File: `internal/azuredevops/remote.go` — URL parsing
  - `ParseAzureDevOpsRemote(rawURL string) (org, project, repo string, err error)`
  - Supports https://dev.azure.com/* and git@ssh.dev.azure.com:* patterns
- File: `internal/azuredevops/pr.go` — PR operations via `az repos pr`
  - `FindPullRequest(org, project, repo, branch, headSHA string) (*git.PullRequestInfo, error)`
  - `IsPRApproved(org, project, repo string, prID int) (bool, error)`
  - `MergePR(org, project, repo string, prID int, method string) (string, error)`
  - Each calls `az repos pr {list|show|update}` with `--org`, `--detect false`
- Follow `internal/bitbucket/` package structure as template

**1.3 Config: Add `azuredevops` to vcs_provider enum**
- `internal/refinery/engineer.go:462` — add `case "azuredevops":`
  - Error message: update to `...(supported: github, bitbucket, azuredevops)`
- `internal/config/types.go:1347` — update comment on `VCSProvider` field

### Phase 2: Refinery PR Provider

**2.1 Implement Azure DevOps PR provider**
- File: `internal/refinery/pr_provider_azuredevops.go`
- Struct: `azuredevopsPRProvider` with `org`, `project`, `repo` fields
- Constructor: `newAzureDevOpsPRProvider(g *git.Git) (PRProvider, error)`
  - Calls `g.RemoteURL("origin")` then `azuredevops.ParseAzureDevOpsRemote()`
  - Stores org, project, repo from parsing
- Implements `PRProvider` interface, delegating to `azuredevops.FindPullRequest`, `IsPRApproved`, `MergePR`

### Phase 3: Core PR Operations (non-refinery)

**3.1 PR lookup in `internal/git/pr_lookup.go`**
- Make `runGH/Git` VCS-aware: detect if the remote is Azure DevOps, dispatch to `az repos pr`
- Add `runAZ(args ...string) ([]byte, error)` method on `*Git`
- Add `githubRepo` → `repoIdent` type that holds `(org, project, repo)` for either GitHub or Azure DevOps
- Modify `pullRequestTargetRepo` to parse Azure DevOps URLs via `azuredevops.ParseAzureDevOpsRemote()`
- Add `adoPullRequest` JSON struct mirroring `az repos pr show --output json` output

**3.2 PR approval and merge in `internal/git/git.go`**
- Add `IsAzureDevOpsPRApproved(org, project, repo string, prID int)` calling `az repos pr policy list`
- Add `AzureDevOpsPRMerge(org, project, repo string, prID int, method string)` calling `az repos pr update --status completed`
- These mirror the Bitbucket methods already on `*git.Git`

**3.3 PR creation in `internal/cmd/done.go`**
- The polecat done handler creates PRs via `gh pr create`
- Add conditional: if VCS provider is azuredevops, call `az repos pr create` instead
- Token: reads `AZURE_DEVOPS_EXT_PAT` from env

**3.4 Repo creation in `internal/cmd/gitinit.go`**
- `createGitHubRepo()` calls `gh repo create`
- Add `createAzureDevOpsRepo()` that calls `az repos create --name X --project Y --org Z`
- Route based on remote URL parsing (if remote is dev.azure.com, use AZ path)

**3.5 Identity detection in `internal/config/overseer.go`**
- `detectFromGitHub()` calls `gh api user`
- Add `detectFromAzureDevOps()` that calls `az devops user show --output json`
- Route based on remote URL

**3.6 Web convoy PR fetching in `internal/web/fetcher.go`**
- `fetchPRsForRepo()` calls `gh pr list`
- `gitURLToRepoPath()` hardcodes github.com URL parsing
- Add Azure DevOps branch: parse `dev.azure.com` URLs, call `az repos pr list --org X --project Y --repository Z`

**3.7 Web dashboard PR API in `internal/web/api.go`**
- `runGhCommand()` calls `gh pr view`
- Add Azure DevOps version using `az repos pr show`

**3.8 Formula/sling PR resolution**
- `internal/cmd/formula.go:1108,1115` — `gh pr view` for title and files
- `internal/cmd/sling.go:1349` — `gh pr view` for branch name
- Both are simple pr view calls; add `az repos pr show --id X --output json` equivalents

### Phase 4: Docker & Env Config

**4.1 docker-compose.azure.yml**
- Add `AZURE_DEVOPS_EXT_PAT: ${AZURE_DEVOPS_EXT_PAT:-}` to environment
- Add `AZURE_DEVOPS_ORG: ${AZURE_DEVOPS_ORG:-}` for default org

**4.2 .env.azure**
- Add commented placeholders:
  ```env
  AZURE_DEVOPS_EXT_PAT=
  AZURE_DEVOPS_ORG=
  ```

## Risk Notes

1. **CRLF line endings**: Azure CLI shell scripts may have line-ending issues on Windows builds. Ensure `docker-entrypoint.sh` LF enforcement works for the CLI too.
2. **PAT scope**: The Azure DevOps PAT needs `Code (Read & Write)` scope. Document this.
3. **Org URL format**: Azure DevOps URLs differ between orgs using legacy `visualstudio.com` vs newer `dev.azure.com`. Our parser handles both.
4. **Project context**: Some `az repos` commands need `--org`, `--project`, `--repository`. The project is extracted from the remote URL parsing.
5. **Docker image size**: Azure CLI install adds ~300MB. Consider whether this should be conditional or in a separate image tag.

## Validation

1. **Unit tests**: Add `ParseAzureDevOpsRemote` test table covering all URL formats
2. **Integration**: Set `vcs_provider: azuredevops` in config, verify `gt engineer` initializes without error
3. **PR operations**: Create a test PR in an Azure DevOps repo, verify find/approve/merge works via the provider
4. **Docker build**: `docker compose -f docker-compose.azure.yml build` succeeds
5. **CLI availability**: `docker compose exec gastown az devops --help` works

## Files Affected

| File | Change |
|---|---|
| `Dockerfile` | Install Azure CLI + devops extension |
| `internal/azuredevops/remote.go` | **NEW** — URL parsing |
| `internal/azuredevops/pr.go` | **NEW** — PR lookup/approve/merge |
| `internal/refinery/pr_provider_azuredevops.go` | **NEW** — PRProvider impl |
| `internal/refinery/engineer.go` | Add `azuredevops` to vcs_provider switch |
| `internal/config/types.go` | Update VCSProvider comment |
| `internal/git/pr_lookup.go` | Add VCS-aware PR resolution |
| `internal/git/git.go` | Add Azure DevOps PR methods |
| `internal/cmd/done.go` | Add AZ PR creation |
| `internal/cmd/gitinit.go` | Add AZ repo creation |
| `internal/cmd/formula.go` | Add AZ PR info fetch |
| `internal/cmd/sling.go` | Add AZ PR branch resolution |
| `internal/config/overseer.go` | Add AZ identity detection |
| `internal/web/fetcher.go` | Add AZ PR list + URL parsing |
| `internal/web/api.go` | Add AZ PR show |
| `docker-compose.azure.yml` | Add AZ env vars |
| `.env.azure` | Add AZ token placeholder |
