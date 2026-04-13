#!/usr/bin/env bash
set -euo pipefail

: "${REPO_URL:?REPO_URL is required (e.g. https://github.com/tmuxpack/tpack)}"
: "${GITHUB_PAT:?GITHUB_PAT is required}"

RUNNER_NAME="${RUNNER_NAME:-$(hostname)}"
RUNNER_LABELS="${RUNNER_LABELS:-aur-publisher}"
RUNNER_WORKDIR="${RUNNER_WORKDIR:-/home/runner}"

if [[ ! "$REPO_URL" =~ ^https://github\.com/([^/]+)/([^/]+)/?$ ]]; then
    echo "error: REPO_URL must look like https://github.com/<owner>/<repo>" >&2
    exit 1
fi
OWNER="${BASH_REMATCH[1]}"
REPO="${BASH_REMATCH[2]}"

echo "Requesting runner registration token for ${OWNER}/${REPO}..."
API_RESPONSE="$(
    curl -fsSL \
        -X POST \
        -H "Accept: application/vnd.github+json" \
        -H "Authorization: Bearer ${GITHUB_PAT}" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "https://api.github.com/repos/${OWNER}/${REPO}/actions/runners/registration-token"
)"

REG_TOKEN="$(echo "$API_RESPONSE" | jq -r '.token // empty')"
if [[ -z "$REG_TOKEN" ]]; then
    API_MESSAGE="$(echo "$API_RESPONSE" | jq -r '.message // "unknown error"')"
    echo "error: failed to obtain registration token: ${API_MESSAGE}" >&2
    exit 1
fi

cd "$RUNNER_WORKDIR"

cleanup() {
    if [[ -n "${RUNNER_PID:-}" ]]; then
        echo "Stopping runner (pid ${RUNNER_PID})..."
        kill -TERM "$RUNNER_PID" 2>/dev/null || true
        wait "$RUNNER_PID" 2>/dev/null || true
    fi
    echo "Deregistering runner..."
    REMOVE_RESPONSE="$(
        curl -fsSL \
            -X POST \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer ${GITHUB_PAT}" \
            -H "X-GitHub-Api-Version: 2022-11-28" \
            "https://api.github.com/repos/${OWNER}/${REPO}/actions/runners/remove-token" 2>/dev/null || true
    )"
    REMOVE_TOKEN="$(echo "$REMOVE_RESPONSE" | jq -r '.token // empty' 2>/dev/null || true)"
    if [[ -n "$REMOVE_TOKEN" ]]; then
        ./config.sh remove --token "$REMOVE_TOKEN" || true
    fi
}
RUNNER_PID=""
trap cleanup SIGINT SIGTERM

./config.sh \
    --unattended \
    --replace \
    --url "$REPO_URL" \
    --token "$REG_TOKEN" \
    --labels "$RUNNER_LABELS" \
    --name "$RUNNER_NAME"

./run.sh &
RUNNER_PID=$!
wait "$RUNNER_PID" || true
