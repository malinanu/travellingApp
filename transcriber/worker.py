"""Spotto transcriber: writes transcripts of audio guides.

Idles cheaply and does the heavy work only when there is some. Every
POLL_SECONDS it asks the Spotto API for an audio guide waiting for a
transcript (POST /internal/transcripts/claim, with TRANSCRIBER_TOKEN). For
each one it downloads the audio through the API, runs transcribe.py in a
child process (the only place the Whisper model is loaded, so its memory
goes back to the system when the file is done) and posts the transcript
back, or reports why it couldn't.

Standard library only: between jobs this process uses a few tens of MB.
See README.md for the settings.
"""

import http.client
import json
import logging
import os
import pathlib
import re
import signal
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

log = logging.getLogger("transcriber")

HERE = pathlib.Path(__file__).resolve().parent
EXIT_UNDECODABLE = 2  # transcribe.py: the file isn't audio we can decode
CONTENT_RANGE = re.compile(r"bytes (\d+)-(\d+)/(\d+|\*)")


class Config:
    def __init__(self, env=os.environ):
        self.api_url = env.get("API_URL", "http://api:8881").rstrip("/")
        self.token = env.get("TRANSCRIBER_TOKEN", "").strip()
        self.poll_seconds = float(env.get("POLL_SECONDS") or 60)
        self.job_timeout = float(env.get("JOB_TIMEOUT_SECONDS") or 4 * 3600)
        cmd = env.get("TRANSCRIBE_CMD")
        self.transcribe_cmd = cmd.split() if cmd else [sys.executable, str(HERE / "transcribe.py")]
        self.heartbeat_file = env.get("HEARTBEAT_FILE") or ""
        self.work_dir = env.get("WORK_DIR") or None
        # The API ends any one response after 30 s, so the audio comes in
        # pieces small enough for a slow link.
        self.chunk_bytes = int(env.get("DOWNLOAD_CHUNK_BYTES") or 16 << 20)


class ApiError(Exception):
    def __init__(self, status, message):
        super().__init__(f"HTTP {status}: {message}")
        self.status = status


class JobFailed(Exception):
    def __init__(self, message, retryable):
        super().__init__(message)
        self.retryable = retryable


class Stopping(Exception):
    """The worker was asked to stop in the middle of a job."""


class Worker:
    def __init__(self, cfg):
        self.cfg = cfg
        self.stopping = False

    # ---- talking to the API ----

    def _request(self, method, path, body=None, timeout=30):
        data = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(self.cfg.api_url + path, data=data, method=method)
        req.add_header("Authorization", "Bearer " + self.cfg.token)
        if data is not None:
            req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as res:
                raw = res.read()
                return res.status, (json.loads(raw) if raw else None)
        except urllib.error.HTTPError as e:
            raise ApiError(e.code, e.read()[:300].decode(errors="replace")) from None

    def claim(self):
        """The next job, or None when there's nothing to do."""
        status, job = self._request("POST", "/internal/transcripts/claim")
        return job if status == 200 else None

    def report(self, path, body):
        # A transcript can take an hour to make, so try hard to hand it in.
        for attempt in range(5):
            try:
                self._request("POST", path, body, timeout=60)
                return
            except (ApiError, OSError) as e:
                if isinstance(e, ApiError) and e.status < 500:
                    raise
                log.warning("report failed (%s), retrying", e)
                time.sleep(min(60, 5 * 2**attempt))
        log.error("giving up reporting to %s; the job will be picked up again later", path)

    # ---- one job ----

    def download(self, url, dest):
        """Saves the audio to dest, a piece at a time (HTTP Range), checking
        every piece is the one asked for: a cut-off response must never pass
        for the whole recording."""
        if url.startswith("/"):
            url = self.cfg.api_url + url
        have, total = 0, None
        try:
            with open(dest, "wb") as f:
                while total is None or have < total:
                    req = urllib.request.Request(url, headers={"Range": f"bytes={have}-{have + self.cfg.chunk_bytes - 1}"})
                    with urllib.request.urlopen(req, timeout=60) as res:
                        body = res.read()  # raises IncompleteRead when cut short
                        if res.status != 206:  # no ranges: this is the whole file
                            f.write(body)
                            return
                        m = CONTENT_RANGE.fullmatch(res.headers.get("Content-Range", ""))
                        if not m or int(m.group(1)) != have or not body:
                            raise JobFailed("the API sent the wrong part of the audio", retryable=True)
                        f.write(body)
                        have += len(body)
                        if m.group(3) != "*":
                            total = int(m.group(3))
                        elif len(body) < self.cfg.chunk_bytes:
                            return
        except urllib.error.HTTPError as e:
            if e.code == 416 and have == 0:
                raise JobFailed("the audio file is empty", retryable=False) from None
            # 404: the audio was removed; anything else is likely temporary.
            raise JobFailed(f"couldn't download the audio (HTTP {e.code})", retryable=e.code != 404) from None
        except (OSError, http.client.HTTPException) as e:
            raise JobFailed(f"couldn't download the audio ({e!r})", retryable=True) from None

    def run_transcribe(self, audio, out):
        """Runs transcribe.py, keeping the heartbeat fresh while it works."""
        started = time.monotonic()
        with tempfile.TemporaryFile() as errors:
            proc = subprocess.Popen(self.cfg.transcribe_cmd + [audio, out], stdout=subprocess.DEVNULL, stderr=errors)
            while True:
                try:
                    code = proc.wait(timeout=15)
                    break
                except subprocess.TimeoutExpired:
                    self.heartbeat()
                    if self.stopping:
                        proc.terminate()
                        proc.wait()
                        raise Stopping()
                    if time.monotonic() - started > self.cfg.job_timeout:
                        proc.kill()
                        proc.wait()
                        hours = self.cfg.job_timeout / 3600
                        raise JobFailed(f"transcribing took longer than {hours:g} hours", retryable=False)
            errors.seek(0)
            tail = errors.read()[-400:].decode(errors="replace").strip()
        if code == 0:
            with open(out, encoding="utf-8") as f:
                return json.load(f)
        if code == EXIT_UNDECODABLE:
            raise JobFailed(tail or "the file isn't audio that can be transcribed", retryable=False)
        # Killed for memory, a crash, a missing model: worth another try.
        raise JobFailed(tail or f"the transcriber exited with code {code}", retryable=True)

    def process(self, job):
        job_id, audio_path = job["id"], job["audioPath"]
        log.info("transcribing %s (attempt %s)", job_id, job.get("attempt"))
        started = time.monotonic()
        with tempfile.TemporaryDirectory(dir=self.cfg.work_dir) as tmp:
            audio, out = os.path.join(tmp, "audio"), os.path.join(tmp, "transcript.json")
            try:
                self.download(job["audioUrl"], audio)
                result = self.run_transcribe(audio, out)
            except JobFailed as e:
                log.warning("job %s failed (retryable=%s): %s", job_id, e.retryable, e)
                self.report(f"/internal/transcripts/{job_id}/fail",
                            {"audioPath": audio_path, "error": str(e), "retryable": e.retryable})
                return
            except Stopping:
                self.report(f"/internal/transcripts/{job_id}/fail",
                            {"audioPath": audio_path, "error": "the transcriber was restarted", "retryable": True})
                raise
        result["audioPath"] = audio_path
        try:
            self.report(f"/internal/transcripts/{job_id}/result", result)
        except ApiError as e:
            # The API refused the transcript itself; trying again won't help.
            log.error("job %s: transcript refused: %s", job_id, e)
            self.report(f"/internal/transcripts/{job_id}/fail",
                        {"audioPath": audio_path, "error": f"the API refused the transcript ({e})", "retryable": False})
            return
        log.info("job %s done: %d lines, %s, in %.0fs", job_id, len(result.get("segments", [])),
                 result.get("language") or "?", time.monotonic() - started)

    # ---- the loop ----

    def heartbeat(self):
        if self.cfg.heartbeat_file:
            pathlib.Path(self.cfg.heartbeat_file).touch()

    def run_once(self):
        """Does one job if there is one; returns whether it did."""
        job = self.claim()
        if not job:
            return False
        self.process(job)
        return True

    def run(self):
        backoff = self.cfg.poll_seconds
        while not self.stopping:
            self.heartbeat()
            try:
                if self.run_once():
                    backoff = self.cfg.poll_seconds
                    continue  # there may be more waiting
                backoff = self.cfg.poll_seconds
            except Stopping:
                break
            except ApiError as e:
                if e.status in (401, 404):
                    log.error("the API refused the transcriber (%s): check TRANSCRIBER_TOKEN on both sides", e)
                else:
                    log.warning("claim failed: %s", e)
                backoff = min(backoff * 2, 15 * 60)
            except OSError as e:
                log.warning("can't reach the API at %s: %s", self.cfg.api_url, e)
                backoff = min(backoff * 2, 15 * 60)
            self._sleep(backoff)
        log.info("stopped")

    def _sleep(self, seconds):
        end = time.monotonic() + seconds
        while not self.stopping and time.monotonic() < end:
            time.sleep(min(1.0, end - time.monotonic()))


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    cfg = Config()
    worker = Worker(cfg)

    def stop(signum, frame):
        worker.stopping = True

    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    if not cfg.token:
        # Stay up doing nothing rather than exit, so a stack deployed before
        # the token is set doesn't restart this container over and over.
        log.warning("TRANSCRIBER_TOKEN is not set: transcripts are off until it is (and the container is recreated)")
        while not worker.stopping:
            worker.heartbeat()
            worker._sleep(300)
        return 0
    log.info("waiting for audio guides from %s (every %gs)", cfg.api_url, cfg.poll_seconds)
    worker.run()
    return 0


if __name__ == "__main__":
    sys.exit(main())
