#!/usr/bin/env python3
"""Nemotron Speech ASR inference server for hyprvoice.

Loads NVIDIA parakeet-tdt-0.6b-v2 and exposes a /transcribe endpoint
that accepts multipart WAV audio uploads and returns transcribed text.

Usage:
    python nemotron_server.py [--port 8080] [--host 0.0.0.0]
"""

import argparse
import io
import logging
import tempfile
import os

import torch
import nemo.collections.asr as nemo_asr
from flask import Flask, request, jsonify

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("nemotron-server")

app = Flask(__name__)
model = None


def load_model():
    global model
    log.info("Loading parakeet-tdt-0.6b-v2 model...")
    model = nemo_asr.models.ASRModel.from_pretrained("nvidia/parakeet-tdt-0.6b-v2")
    model.eval()
    if torch.cuda.is_available():
        model = model.cuda()
        log.info("Model loaded on GPU: %s", torch.cuda.get_device_name(0))
    else:
        log.warning("CUDA not available, running on CPU (will be slow)")
    log.info("Model ready.")


@app.route("/transcribe", methods=["POST"])
def transcribe():
    if "audio" not in request.files:
        return jsonify({"error": "no 'audio' file in request"}), 400

    audio_file = request.files["audio"]
    # Save to a temp file since NeMo needs a file path
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        audio_file.save(tmp)
        tmp_path = tmp.name

    try:
        transcriptions = model.transcribe([tmp_path])
        # NeMo returns different formats depending on version
        if isinstance(transcriptions, list):
            if len(transcriptions) > 0 and isinstance(transcriptions[0], str):
                text = transcriptions[0]
            elif hasattr(transcriptions[0], "text"):
                text = transcriptions[0].text
            else:
                text = str(transcriptions[0])
        else:
            text = str(transcriptions)

        log.info("Transcribed: %r", text)
        return jsonify({"text": text})
    except Exception as e:
        log.exception("Transcription failed")
        return jsonify({"error": str(e)}), 500
    finally:
        os.unlink(tmp_path)


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok", "model": "parakeet-tdt-0.6b-v2", "gpu": torch.cuda.is_available()})


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Nemotron ASR server")
    parser.add_argument("--host", default="0.0.0.0", help="Bind address")
    parser.add_argument("--port", type=int, default=8080, help="Port")
    args = parser.parse_args()

    load_model()
    log.info("Starting server on %s:%d", args.host, args.port)
    app.run(host=args.host, port=args.port, threaded=False)
