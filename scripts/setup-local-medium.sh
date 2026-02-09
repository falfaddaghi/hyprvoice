#!/usr/bin/env bash
set -euo pipefail

echo "=== Hyprvoice Local Setup (whisper-cpp medium) ==="

# Check whisper-cli is installed
if ! command -v whisper-cli &>/dev/null; then
  echo "whisper-cli not found. Installing whisper.cpp..."
  if command -v pacman &>/dev/null; then
    sudo pacman -S --needed whisper.cpp
  else
    echo "Error: whisper-cli not found and no supported package manager detected."
    echo "Install manually: https://github.com/ggerganov/whisper.cpp"
    exit 1
  fi
fi

echo "whisper-cli found: $(command -v whisper-cli)"

# Download medium model if not already present
echo "Downloading medium model (1.5GB)..."
hyprvoice model download medium

# Write config
CONFIG_DIR="$HOME/.config/hyprvoice"
CONFIG_FILE="$CONFIG_DIR/config.toml"
mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_FILE" <<'EOF'
[transcription]
  provider = "whisper-cpp"
  model = "medium"
  language = ""
  threads = 0

[llm]
  enabled = false

[injection]
  backends = ["ydotool", "wtype", "clipboard"]

[notifications]
  enabled = true
  type = "desktop"
EOF

echo "Config written to $CONFIG_FILE"

# Restart service
systemctl --user restart hyprvoice.service 2>/dev/null \
  && echo "hyprvoice service restarted." \
  || echo "Service not running. Start with: systemctl --user enable --now hyprvoice.service"

echo ""
echo "Done! Use 'hyprvoice toggle' or bind SUPER+R to start dictating."
