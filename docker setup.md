 ● Here's the full steps to get it running:

  1. Install NVIDIA Container Toolkit (one-time):
  sudo pacman -S nvidia-container-toolkit
  sudo nvidia-ctk runtime configure --runtime=docker
  sudo systemctl restart docker

  2. Build the Docker image (takes a while — downloads NeMo base image + model):
  cd ~/Work/hyprvoice/scripts
  docker build -f Dockerfile.nemotron -t nemotron-asr .

  3. Run the server:
  docker run --rm --gpus all -p 8080:8080 nemotron-asr

  Wait for "Model ready." and "Starting server on 0.0.0.0:8080" in the logs.

  4. Configure hyprvoice:
  go run . configure
  Pick Nemotron ASR → set URL to http://localhost:8080 → save.

  Or directly edit your config:
  [transcription]
    provider = "nemotron"
    model = "nemotron-speech-0.6b"
    language = "en"
    nemotron_url = "http://localhost:8080"

  5. Test it:
  # Quick health check
  curl http://localhost:8080/health

  # Run hyprvoice
  go run . start

  The NeMo base image is large (~15GB), and model download adds another ~1.2GB, so the first build will take a
  while. After that, container startup is fast (model loads from cache in ~10-15s).

  Start with step 1 — install nvidia-container-toolkit — and let me know when you're ready.

