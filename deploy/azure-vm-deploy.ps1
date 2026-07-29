# Azure VM deployment for Gas Town
# Usage: .\deploy\azure-vm-deploy.ps1
param(
    [string]$ResourceGroup = "gastown-rg",
    [string]$Location = "eastus2",
    [string]$VmName = "gastown-vm",
    [string]$AdminUsername = "azureuser",
    [string]$SshKeyPath = "$HOME\.ssh\id_rsa.pub",
    [string]$VmSize = "Standard_D2s_v5"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
    Write-Host "`n=== $msg ===" -ForegroundColor Cyan
}

Write-Step "Creating resource group: $ResourceGroup"
az group create --name $ResourceGroup --location $Location --output none

Write-Step "Ensuring SSH key exists"
$sshDir = Split-Path -Parent $SshKeyPath
$privKey = $SshKeyPath -replace '\.pub$', ''
if (-not (Test-Path $SshKeyPath)) {
    if (-not (Test-Path $sshDir)) {
        New-Item -ItemType Directory -Path $sshDir -Force | Out-Null
    }
    Write-Host "Generating new SSH key at $privKey"
    ssh-keygen -t rsa -b 4096 -f $privKey -N "" -q
} else {
    Write-Host "Using existing SSH key: $SshKeyPath"
}

Write-Step "Deploying VM: $VmName"
az vm create `
  --resource-group $ResourceGroup `
  --name $VmName `
  --image Ubuntu2204 `
  --size $VmSize `
  --admin-username $AdminUsername `
  --ssh-key-values $SshKeyPath `
  --public-ip-sku Standard `
  --output json

Write-Step "Opening ports 22 and 3000"
az vm open-port --port 22 --resource-group $ResourceGroup --name $VmName --priority 100 --output none
az vm open-port --port 3000 --resource-group $ResourceGroup --name $VmName --priority 101 --output none

Write-Step "Getting public IP"
$ip = az vm show `
  --resource-group $ResourceGroup `
  --name $VmName `
  --show-details `
  --query publicIps -o tsv

if (-not $ip) {
    throw "Failed to get public IP for VM $VmName"
}

Write-Host "`nVM public IP: $ip" -ForegroundColor Green
Write-Host "SSH: ssh $AdminUsername@$ip" -ForegroundColor Yellow
Write-Host "Dashboard: http://$ip`:3000" -ForegroundColor Yellow

Write-Host "`nNext steps:"
Write-Host "  1. ssh $AdminUsername@$ip"
Write-Host "  2. curl -fsSL https://get.docker.com | sudo sh"
Write-Host "  3. sudo usermod -aG docker `$USER && newgrp docker"
Write-Host "  4. git clone <repo-url> gastown && cd gastown"
Write-Host "  5. docker compose up -d"
Write-Host "`nTo delete everything when done:"
Write-Host "  az group delete --name $ResourceGroup --yes --no-wait" -ForegroundColor Red
