# Transcriber

Writes transcripts of Spotto audio guides with OpenAI's Whisper **small**
model, run through [faster-whisper](https://github.com/SYSTRAN/faster-whisper)
(CTranslate2, int8 on CPU).

When a publisher uploads an audio guide's MP3 or M4A, the API queues a
transcript. This service asks the API for queued audio every minute,
downloads it through the API, transcribes it, and posts the transcript back.
Publishers review and correct it in the Command Center. Travellers follow
along in the app, and the words are searchable.

```
worker.py      the idle loop (standard library only, ~25 MB): claim → download → transcribe → report
transcribe.py  the only part that loads the model; run as a child process per recording,
               so its memory (~0.6–1 GB) goes back to the system when the file is done
```

Long recordings are decoded and transcribed ten minutes at a time, so an
audiobook needs no more memory than a short guide.

The service needs no database, storage keys or open port: only the API's
address and a shared token.

## Settings

| Variable | Default | |
|---|---|---|
| `TRANSCRIBER_TOKEN` | (required) | Same value as the API's `TRANSCRIBER_TOKEN`. Make one with `openssl rand -hex 32`. |
| `API_URL` | `http://api:8881` | The Spotto API. |
| `POLL_SECONDS` | `60` | How often to ask for work while idle. |
| `JOB_TIMEOUT_SECONDS` | `14400` | Give up on one recording after this long (4 h). |
| `MODEL` | `small` | Whisper size. The image includes `small` only. |
| `DEVICE` / `COMPUTE_TYPE` | `cpu` / `int8` | `cuda` / `float16` on a machine with an NVIDIA GPU. |
| `CPU_THREADS` | `0` | `0` lets CTranslate2 decide. |
| `LANGUAGE` | (detect) | Force a language, e.g. `en`. |

## In the stack

`docker-compose.yml` runs it as the `transcriber` service. GitHub Actions
builds `ghcr.io/malinanu/travellingapp-transcriber` with the model inside, so
it never downloads anything at runtime (about 0.6 GB to pull). It is capped
at 1.5 GB of memory, uses two threads and has a low CPU priority, so a long
audiobook can't slow the API down.

Turning it on:

1. In Portainer's stack environment, set `TRANSCRIBER_TOKEN`. The `api` and
   `transcriber` services both read it.
2. Pull and redeploy.

Without the token, the API doesn't queue transcripts, the internal
endpoints don't exist and the transcriber just waits.

The first time GitHub publishes `travellingapp-transcriber`, the package is
private: make it public (GitHub → Packages → Package settings) like the
others, or Portainer can't pull it.

## On your own computer

The service only needs to reach the API, so it can also run on a PC, for
example one with a GPU for speed:

```sh
cd transcriber
python -m venv .venv && . .venv/bin/activate
pip install -r requirements.txt
API_URL=https://spottoapi.threelogiq.com TRANSCRIBER_TOKEN=… \
  DEVICE=cuda COMPUTE_TYPE=float16 python worker.py
```

The first run downloads the model from Hugging Face. Run only one
transcriber per API if you want jobs in upload order; several are safe,
because each recording is claimed by one of them.

## Tests

```sh
python -m unittest -v
```

`test_worker.py` runs the loop against a fake API and a fake model.
`test_transcribe.py` checks the ten-minute decoding with the real decoder.
It needs `av` and `numpy`, and is skipped without them.
