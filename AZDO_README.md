# Gas Town on Azure DevOps

Run the Gas Town multi-agent workspace on an Azure VM with opencode as the
mayor and Azure DevOps as the VCS backend.

## Prerequisites

- **Azure account** with credits
- **SSH key** at `~/.ssh/id_rsa.pub` (or `ssh-keygen -t rsa -b 4096` to create one)
- **Azure DevOps org** with a PAT token (Code Read & Write scope) for repo/PR operations

## 1. Provision the Azure VM

Go to https://portal.azure.com → **Virtual machines** → **Create** → **Azure virtual machine**.

| Setting | Value |
|---|---|
| Subscription | Your credits subscription |
| Resource group | `gastown-rg` (create new) |
| VM name | `gastown-vm` |
| Region | East US 2 |
| Image | Ubuntu Server 22.04 LTS |
| Size | Standard B2s or D2s v5 |
| Authentication | SSH public key |
| Username | `azureuser` |
| SSH key | Paste from `~/.ssh/id_rsa.pub` |
| Public inbound ports | SSH (22), HTTP (80) |
| OS disk | **128 GiB** (Standard SSD) |

After creation, add port 80: VM → **Networking** → **Add inbound port rule** →
Destination port `80`, Priority `101`, Name `dashboard`.

## 2. SSH in and install

```bash
ssh azureuser@<VM_PUBLIC_IP>
```

```bash
# Install Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker

# Clone the repo
git clone https://github.com/buchenberg/gastown.git gastown
cd gastown
```

## 3. Configure secrets

```bash
cat > .env.azure << 'EOF'
DEEPSEEK_API_KEY=sk-your-key-here
AZURE_DEVOPS_EXT_PAT=your-ado-pat-token
EOF
```

## 4. Start the stack

```bash
docker compose -f docker-compose.azure.yml --env-file .env.azure up -d
```

Dashboard at `http://<VM_PUBLIC_IP>`.

## 5. Attach the mayor

```bash
./start-mayor.sh
```

This starts the container (if not running) and attaches opencode as the mayor
in an interactive chat session. Type `/exit` to quit.

## 6. Daily workflow

```bash
ssh azureuser@<VM_IP>
cd gastown
./start-mayor.sh
```

The container persists between SSH sessions. Running `start-mayor.sh` again
re-attaches to the running mayor session.

## Create rigs

A rig is a project container — each rig has its own repo, beads database,
crew, polecats, witness, and refinery.

### From an existing repo

```bash
docker compose -f docker-compose.azure.yml exec gastown gt rig add myrig https://github.com/user/myrig
# or Azure DevOps
docker compose -f docker-compose.azure.yml exec gastown gt rig add myrig https://dev.azure.com/myorg/myproject/_git/myrepo
```

### Create a new Azure DevOps repo + rig in one step

```bash
docker compose -f docker-compose.azure.yml exec gastown gt rig add myrig --azure
```

### Adopt an existing directory

```bash
docker compose -f docker-compose.azure.yml exec gastown gt rig add myrig --adopt
```

### Rig flags

| Flag | Purpose |
|---|---|
| `--prefix <p>` | Beads issue prefix (default: derived from name) |
| `--adopt` | Register existing directory without cloning |
| `--force` | Force overwrite if directory exists |
| `--push-url` | Push URL for forked rigs |
| `--upstream-url` | Upstream URL for forked rigs |

### Rig lifecycle

```bash
gt rig list                    # List all rigs
gt rig boot myrig              # Start witness + refinery
gt rig start myrig             # Resume a parked rig
gt rig stop myrig              # Stop all agents
gt rig remove myrig            # Remove from registry (keeps files)
```

The rig's refinery config is at `<town-root>/<rig-name>/config.json`. Set
`vcs_provider` and `merge_strategy` there to control PR operations.

## Using Azure DevOps as the VCS provider

Set `vcs_provider: azuredevops` in your rig's refinery config:

**Path:** `<town-root>/<rig-name>/config.json`

**Example** (`gastown/myrig/config.json`):

```json
{
  "merge_queue": {
    "vcs_provider": "azuredevops",
    "merge_strategy": "pr"
  }
}
```

Valid `vcs_provider` values: `github` (default), `bitbucket`, `azuredevops`.

Inside the container, the town root is `/gt`, so a rig named `myrig` would be at
`/gt/myrig/config.json`.

To edit from the host (since `/gt` is bind-mounted from the cloned repo):

```bash
nano ~/gastown/myrig/config.json
```

Create repos on Azure DevOps:

```bash
docker compose -f docker-compose.azure.yml exec gastown gt install --git \
  --azure=myorg/myproject/myrepo
```

PR operations (create, approve, merge) auto-detect Azure DevOps repos by
parsing `dev.azure.com` remote URLs.

## Ports

| Port | Service |
|---|---|
| 22 | SSH |
| 80 | Gas Town dashboard |

## Updating

```bash
cd gastown
git pull
docker compose -f docker-compose.azure.yml --env-file .env.azure up -d --build
```

## Cleanup

Delete the resource group to stop all charges:

```powershell
az group delete --name gastown-rg --yes --no-wait
```

Or from the Azure Portal: Resource groups → `gastown-rg` → Delete.
