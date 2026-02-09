#!/usr/bin/env bash
set -euo pipefail

CONTAINER="nemotron"
IMAGE="images:ubuntu/24.04"
PORT=8080

log() { echo -e "\033[1;34m==>\033[0m $*"; }
err() { echo -e "\033[1;31m==>\033[0m $*" >&2; exit 1; }

# --- Preflight checks ---
command -v incus >/dev/null || err "incus not installed: sudo pacman -S incus"
groups | grep -q incus-admin || err "Not in incus-admin group. Run: newgrp incus-admin"

# --- Create or start container ---
if incus info "$CONTAINER" &>/dev/null; then
    state=$(incus list "$CONTAINER" --format csv -c s)
    if [ "$state" != "RUNNING" ]; then
        log "Starting existing container..."
        incus start "$CONTAINER"
    else
        log "Container already running."
    fi
else
    log "Creating container..."
    incus launch "$IMAGE" "$CONTAINER"
    sleep 3
fi

# --- GPU passthrough ---
if ! incus config device show "$CONTAINER" | grep -q "gpu"; then
    log "Adding GPU passthrough..."
    incus config device add "$CONTAINER" gpu gpu
    incus restart "$CONTAINER"
    sleep 3
fi

log "Verifying GPU inside container..."
if ! incus exec "$CONTAINER" -- nvidia-smi &>/dev/null; then
    err "GPU not visible inside container. Check your NVIDIA driver and Incus GPU support."
fi
incus exec "$CONTAINER" -- nvidia-smi --query-gpu=name,memory.total --format=csv,noheader

# --- Install dependencies ---
log "Installing system packages..."
incus exec "$CONTAINER" -- bash -c "
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq python3 python3-pip python3-venv > /dev/null
"

log "Setting up Python venv and installing packages (this takes a few minutes)..."
incus exec "$CONTAINER" -- bash -c "
    python3 -m venv /opt/nemotron
    source /opt/nemotron/bin/activate
    pip install -q --upgrade pip
    pip install -q torch --index-url https://download.pytorch.org/whl/cu121
    pip install -q 'nemo_toolkit[asr]' flask
"

# --- Copy server script ---
log "Copying server script..."
incus file push "$(dirname "$0")/nemotron_server.py" "$CONTAINER/opt/nemotron/server.py"

# --- Download model ---
log "Pre-downloading model (first time takes ~1-2 min)..."
incus exec "$CONTAINER" -- bash -c "
    source /opt/nemotron/bin/activate
    python3 -c \"import nemo.collections.asr as asr; asr.models.ASRModel.from_pretrained('nvidia/parakeet-tdt-0.6b-v2')\"
"

# --- Set up proxy for host access ---
if ! incus config device show "$CONTAINER" | grep -q "proxy${PORT}"; then
    log "Adding port proxy (host:${PORT} -> container:${PORT})..."
    incus config device add "$CONTAINER" "proxy${PORT}" proxy \
        listen=tcp:0.0.0.0:${PORT} \
        connect=tcp:127.0.0.1:${PORT}
fi

# --- Create systemd service inside container ---
log "Creating systemd service..."
incus exec "$CONTAINER" -- bash -c "cat > /etc/systemd/system/nemotron.service" <<'EOF'
[Unit]
Description=Nemotron ASR Server
After=network.target

[Service]
Type=simple
ExecStart=/opt/nemotron/bin/python /opt/nemotron/server.py --port 8080
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

incus exec "$CONTAINER" -- systemctl daemon-reload
incus exec "$CONTAINER" -- systemctl enable --now nemotron

# --- Wait for server to be ready ---
log "Waiting for server to start..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:${PORT}/health &>/dev/null; then
        break
    fi
    sleep 2
done

if curl -sf http://localhost:${PORT}/health &>/dev/null; then
    log "Server is ready!"
    curl -s http://localhost:${PORT}/health | python3 -m json.tool 2>/dev/null || curl -s http://localhost:${PORT}/health
    echo
    log "Configure hyprvoice:"
    echo "  nemotron_url = \"http://localhost:${PORT}\""
    echo
    log "Or run: go run . configure"
else
    err "Server didn't start in time. Check logs: incus exec $CONTAINER -- journalctl -u nemotron -f"
fi
