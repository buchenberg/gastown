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

## Using Azure DevOps as the VCS provider

Set `vcs_provider: azuredevops` in your refinery config:

```json
{
  "merge_queue": {
    "vcs_provider": "azuredevops",
    "merge_strategy": "pr"
  }
}
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
