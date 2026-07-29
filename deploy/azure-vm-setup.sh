#!/bin/bash
set -euo pipefail

# Azure VM deployment for Gas Town
# Prerequisites:
#   - Azure CLI installed and logged in (az login)
#   - Docker installed locally
#   - Sufficient Azure credits

RESOURCE_GROUP="${RESOURCE_GROUP:-gastown-rg}"
LOCATION="${LOCATION:-eastus}"
VM_NAME="${VM_NAME:-gastown-vm}"
ADMIN_USERNAME="${ADMIN_USERNAME:-azureuser}"
IMAGE="Ubuntu2204"
SIZE="Standard_B2s"
SSH_KEY_PATH="${SSH_KEY_PATH:-$HOME/.ssh/id_rsa.pub}"

echo "Creating resource group: $RESOURCE_GROUP"
az group create --name "$RESOURCE_GROUP" --location "$LOCATION"

echo "Deploying VM: $VM_NAME"
az vm create \
  --resource-group "$RESOURCE_GROUP" \
  --name "$VM_NAME" \
  --image "$IMAGE" \
  --size "$SIZE" \
  --admin-username "$ADMIN_USERNAME" \
  --ssh-key-values "$SSH_KEY_PATH" \
  --public-ip-sku Standard \
  --output json

PUBLIC_IP=$(az vm show --resource-group "$RESOURCE_GROUP" --name "$VM_NAME" --show-details --query publicIps -o tsv)
echo "VM public IP: $PUBLIC_IP"

echo "Opening port 22 (SSH) and 3000 (dashboard)"
az vm open-port --port 22 --resource-group "$RESOURCE_GROUP" --name "$VM_NAME" --priority 100
az vm open-port --port 3000 --resource-group "$RESOURCE_GROUP" --name "$VM_NAME" --priority 101

echo ""
echo "Next steps:"
echo "  1. SSH: ssh $ADMIN_USERNAME@$PUBLIC_IP"
echo "  2. On VM, install Docker:"
echo "       curl -fsSL https://get.docker.com | sudo sh"
echo "       sudo usermod -aG docker $ADMIN_USERNAME"
echo "  3. Clone your repo:"
echo "       git clone <your-repo-url> gastown"
echo "       cd gastown"
echo "  4. Deploy:"
echo "       docker compose up -d"
echo "  5. Dashboard: http://$PUBLIC_IP:3000"
