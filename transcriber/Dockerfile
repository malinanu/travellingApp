# Spotto transcriber — writes transcripts of audio guides with OpenAI's
# Whisper "small" model through faster-whisper (CTranslate2, int8 on CPU).
# See README.md. The model is downloaded here, at build time, so the
# container never needs Hugging Face (or any internet) at runtime.

FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    HF_HUB_DISABLE_TELEMETRY=1 \
    MODEL=small \
    MODEL_DIR=/models \
    DEVICE=cpu \
    COMPUTE_TYPE=int8

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

# Bake the model (~460 MB) into the image.
RUN python -c "import os; from faster_whisper.utils import download_model; download_model(os.environ['MODEL'], cache_dir=os.environ['MODEL_DIR'])"

RUN useradd --system --uid 10001 --home-dir /app transcriber && chown -R transcriber /models
COPY worker.py transcribe.py ./
USER transcriber

# Use only the baked-in model; touch this file while alive (worker.py).
ENV MODEL_OFFLINE=1 \
    HEARTBEAT_FILE=/tmp/transcriber-alive

# Healthy while the loop (or a running job) touched the heartbeat file in
# the last 10 minutes.
HEALTHCHECK --interval=2m --timeout=10s --start-period=1m \
    CMD python -c "import os,sys,time; f=os.environ['HEARTBEAT_FILE']; sys.exit(0 if os.path.exists(f) and time.time()-os.path.getmtime(f) < 600 else 1)"

CMD ["python", "worker.py"]
