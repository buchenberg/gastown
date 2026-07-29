# Azure VM Deployment Walkthrough (opencode)

Your VM: `132.196.64.129` (already provisioned via portal)

## Step 1: SSH into the VM

```powershell
ssh azureuser@132.196.64.129
```

## Step 2: Install Docker

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
```

## Step 3: Clone the repo

```bash
git clone <your-repo-url> gastown
cd gastown
```

## Step 4: Create secrets file

```bash
cat > .env.azure << 'EOF'
DEEPSEEK_API_KEY=sk-your-key-here
EOF
```

## Step 5: Start the container

```bash
docker compose -f docker-compose.azure.yml --env-file .env.azure up -d
```

On startup the entrypoint will:
1. Run `gt install /gt` to initialize the workspace
2. Set `default_agent: opencode` in town settings
3. Write `/gt/mayor/opencode.json` with the DeepSeek provider/model config
4. Start the dashboard on port 3000

## Step 6: Verify

```bash
# Check the config was written
docker compose -f docker-compose.azure.yml exec gastown cat /gt/mayor/opencode.json

# Attach the mayor with opencode
docker compose -f docker-compose.azure.yml exec gastown gt mayor attach
```

Dashboard: `http://132.196.64.129:3000`

## Override provider/model at deploy time

```bash
docker compose -f docker-compose.azure.yml --env-file .env.azure up -d \
  -e OPENCODE_MODEL=openai/gpt-4o \
  -e OPENCODE_PROVIDER_NAME=openai \
  -e OPENCODE_PROVIDER_BASE_URL=https://api.openai.com/v1 \
  -e OPENCODE_API_KEY=$OPENAI_API_KEY
```

## Cost cleanup

```powershell
az group delete --name gastown-rg --yes --no-wait
```
