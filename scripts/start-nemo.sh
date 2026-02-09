#!/usr/bin/env bash
# Start hyprvoice with Nemotron ASR (local NVIDIA transcription)
# Starts the Docker container if not already running, waits for the model
# to load, then launches the hyprvoice daemon.

set -euo pipefail

CONTAINER_NAME="nemotron-asr"
IMAGE_NAME="nemotron-asr"
PORT=8080
HEALTH_URL="http://localhost:${PORT}/health"

# --- Colors ---
red()   { printf '\033[1;31m%s\033[0m\n' "$*"; }
green() { printf '\033[1;32m%s\033[0m\n' "$*"; }
cyan()  { printf '\033[1;36m%s\033[0m\n' "$*"; }

# --- Start or reuse the Nemotron Docker container ---
start_container() {
    # Check if container is already running
    if docker ps --format '{{.Names}}' | grep -qw "$CONTAINER_NAME"; then
        green "Nemotron container already running."
        return
    fi

    # Check if a stopped container with that name exists and remove it
    if docker ps -a --format '{{.Names}}' | grep -qw "$CONTAINER_NAME"; then
        cyan "Removing stopped nemotron container..."
        docker rm "$CONTAINER_NAME" >/dev/null
    fi

    cyan "Starting Nemotron ASR container..."
    docker run -d --name "$CONTAINER_NAME" --gpus all -p "${PORT}:8080" "$IMAGE_NAME" >/dev/null
    green "Container started."
}

# --- Wait for the model to be ready ---
wait_for_health() {
    cyan "Waiting for Nemotron model to load..."
    local attempts=0
    local max_attempts=60  # 60 × 2s = 2 minutes
    while ! curl -sf "$HEALTH_URL" >/dev/null 2>&1; do
        attempts=$((attempts + 1))
        if [ "$attempts" -ge "$max_attempts" ]; then
            red "Nemotron server did not become healthy after $((max_attempts * 2))s."
            red "Check logs: docker logs $CONTAINER_NAME"
            exit 1
        fi
        sleep 2
    done
    green "Nemotron server is ready."
}

# --- Main ---
start_container
wait_for_health

cyan "Starting hyprvoice daemon..."
exec hyprvoice serve
