#!/bin/sh
set -e

# Re-apply git/dolt config on every start so env var changes take effect
# even when the home volume already exists from a previous run.
if [ -n "$GIT_USER" ] && [ -n "$GIT_EMAIL" ]; then
    git config --global user.name "$GIT_USER"
    git config --global user.email "$GIT_EMAIL"
    git config --global credential.helper store
    dolt config --global --add user.name "$GIT_USER"
    dolt config --global --add user.email "$GIT_EMAIL"
fi

if [ ! -f /gt/mayor/town.json ]; then
    echo "Initializing Gas Town workspace at /gt..."
    /app/gastown/gt install /gt --git
else
    echo "Refreshing Gas Town workspace at /gt..."
    /app/gastown/gt install /gt --git --force
fi

cd /gt

# Configure default agent (e.g., opencode) if specified
if [ -n "$GT_DEFAULT_AGENT" ]; then
    echo "Setting default agent to $GT_DEFAULT_AGENT..."
    /app/gastown/gt config set default_agent "$GT_DEFAULT_AGENT" 2>/dev/null || true
fi

# Write opencode config if model/provider env vars are set
if [ -n "$OPENCODE_MODEL" ] && [ -n "$OPENCODE_PROVIDER_BASE_URL" ]; then
    echo "Writing opencode config to /gt/mayor/opencode.json..."
    mkdir -p /gt/mayor
    cat > /gt/mayor/opencode.json << EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "$OPENCODE_MODEL",
  "provider": {
    "${OPENCODE_PROVIDER_NAME:-deepseek}": {
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "$OPENCODE_PROVIDER_BASE_URL",
        "apiKey": "${OPENCODE_API_KEY:-}"
      }
    }
  }
}
EOF
fi

if [ -n "${START_DASHBOARD:-1}" ] && [ "${START_DASHBOARD}" != "0" ]; then
    echo "Starting dashboard in background..."
    /app/gastown/gt dashboard --bind 0.0.0.0 --port 8080 >/dev/null 2>&1 &
fi

exec "$@"
