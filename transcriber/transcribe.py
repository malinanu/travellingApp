"""Transcribes one audio file with faster-whisper (Whisper, CTranslate2).

Run by worker.py in a child process, so the model's memory is handed back
to the system as soon as the file is done:

    python transcribe.py <audio file> <output.json>

Writes {"language", "durationSec", "segments": [{"start", "end", "text"}]}
(times in seconds). The audio is decoded and transcribed ten minutes at a
time, so a long audiobook needs no more memory than a short guide.

Exit codes: 0 done, 2 the file isn't audio we can decode (no point
retrying), anything else a failure worth retrying.

Settings (environment): MODEL (default "small"), DEVICE ("cpu"),
COMPUTE_TYPE ("int8"), CPU_THREADS (0 = library default), MODEL_DIR (where
the model is cached), MODEL_OFFLINE=1 (never download), LANGUAGE (skip
detection, e.g. "en"), BEAM_SIZE (5).
"""

import json
import os
import sys

SAMPLE_RATE = 16000
CHUNK_SECONDS = 600

EXIT_UNDECODABLE = 2


class NoAudio(Exception):
    """The file has no audio stream."""


def audio_chunks(path, chunk_seconds=CHUNK_SECONDS):
    """Yields (start_seconds, float32 mono 16 kHz samples) pieces of the file."""
    import av
    import numpy as np

    resampler = av.audio.resampler.AudioResampler(format="s16", layout="mono", rate=SAMPLE_RATE)
    chunk_len = chunk_seconds * SAMPLE_RATE
    pending, have, start = [], 0, 0.0

    def flush():
        return np.concatenate(pending).astype(np.float32) / 32768.0

    with av.open(path) as container:
        if not container.streams.audio:
            raise NoAudio("the file has no audio")
        frames = container.decode(audio=0)
        for frame in frames:
            for out in resampler.resample(frame):
                arr = out.to_ndarray().reshape(-1)
                pending.append(arr)
                have += arr.size
                if have >= chunk_len:
                    samples = flush()
                    yield start, samples
                    start += samples.size / SAMPLE_RATE
                    pending, have = [], 0
        for out in resampler.resample(None):
            pending.append(out.to_ndarray().reshape(-1))
            have += pending[-1].size
    if have:
        yield start, flush()


def transcribe(path, model, language=None, beam_size=5):
    segments, total, detected = [], 0.0, language
    for offset, samples in audio_chunks(path):
        total = offset + samples.size / SAMPLE_RATE
        pieces, info = model.transcribe(samples, beam_size=beam_size, vad_filter=True, language=detected)
        # The first piece decides the language, so a long recording reads
        # consistently even if a later piece is mostly music or silence.
        detected = detected or info.language
        for sg in pieces:
            text = sg.text.strip()
            if text:
                segments.append({"start": round(offset + sg.start, 2), "end": round(offset + sg.end, 2), "text": text})
    return {"language": detected or "", "durationSec": round(total, 2), "segments": segments}


def load_model():
    from faster_whisper import WhisperModel

    return WhisperModel(
        os.environ.get("MODEL", "small"),
        device=os.environ.get("DEVICE", "cpu"),
        compute_type=os.environ.get("COMPUTE_TYPE", "int8"),
        cpu_threads=int(os.environ.get("CPU_THREADS") or 0),
        download_root=os.environ.get("MODEL_DIR") or None,
        local_files_only=os.environ.get("MODEL_OFFLINE") == "1",
    )


def main(argv):
    if len(argv) != 3:
        print("usage: transcribe.py <audio file> <output.json>", file=sys.stderr)
        return 1
    audio, out = argv[1], argv[2]
    import av

    # Turn away a file that isn't audio before loading the model for it.
    try:
        with av.open(audio) as container:
            if not container.streams.audio:
                raise NoAudio("the file has no audio")
    except (av.error.FFmpegError, NoAudio) as e:
        print(f"couldn't read the audio: {e}", file=sys.stderr)
        return EXIT_UNDECODABLE

    model = load_model()
    try:
        result = transcribe(
            audio,
            model,
            language=os.environ.get("LANGUAGE") or None,
            beam_size=int(os.environ.get("BEAM_SIZE") or 5),
        )
    except (av.error.FFmpegError, NoAudio) as e:
        print(f"couldn't read the audio: {e}", file=sys.stderr)
        return EXIT_UNDECODABLE
    with open(out, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
