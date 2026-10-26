"""Tests for worker.py against a fake Spotto API and a fake transcribe
command (no Whisper model needed): python -m unittest -v"""

import json
import os
import re
import sys
import tempfile
import textwrap
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import worker

TOKEN = "test-token"
AUDIO = b"ID3 fake mp3 bytes, long enough to come in several pieces"
CHUNK = 16  # download piece size in these tests


class FakeAPI:
    """Serves the internal transcript endpoints and the audio file."""

    def __init__(self, job=None, audio_status=200, result_status=200, ranges=True, cut_short=False):
        """ranges: serve Range requests like the API; cut_short: end the
        audio response early, as a write timeout would."""
        self.job, self.audio_status, self.result_status = job, audio_status, result_status
        self.ranges, self.cut_short = ranges, cut_short
        self.calls = []  # (path, body)
        api = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def _send(self, status, body=None):
                raw = b"" if body is None else (body if isinstance(body, bytes) else json.dumps(body).encode())
                self.send_response(status)
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def do_GET(self):
                api.calls.append((self.path, self.headers.get("Range")))
                if not self.path.startswith("/media/raw/"):
                    return self._send(404, {"error": "not found"})
                if api.audio_status != 200:
                    return self._send(api.audio_status, {"error": "x"})
                m = re.fullmatch(r"bytes=(\d+)-(\d+)", self.headers.get("Range") or "")
                if not api.ranges or not m:
                    if api.cut_short:  # promise the whole file, send half
                        self.send_response(200)
                        self.send_header("Content-Length", str(len(AUDIO)))
                        self.end_headers()
                        self.wfile.write(AUDIO[: len(AUDIO) // 2])
                        self.close_connection = True
                        return
                    return self._send(200, AUDIO)
                start, end = int(m.group(1)), min(int(m.group(2)), len(AUDIO) - 1)
                if start >= len(AUDIO):
                    return self._send(416, {"error": "requested range not satisfiable"})
                part = AUDIO[start : end + 1]
                self.send_response(206)
                self.send_header("Content-Range", f"bytes {start}-{end}/{len(AUDIO)}")
                self.send_header("Content-Length", str(len(part)))
                self.end_headers()
                self.wfile.write(part)

            def do_POST(self):
                length = int(self.headers.get("Content-Length") or 0)
                body = json.loads(self.rfile.read(length)) if length else None
                api.calls.append((self.path, body))
                if self.headers.get("Authorization") != "Bearer " + TOKEN:
                    return self._send(401, {"error": "invalid transcriber token"})
                if self.path == "/internal/transcripts/claim":
                    job, api.job = api.job, None
                    return self._send(200, job) if job else self._send(204)
                if self.path.endswith("/result"):
                    return self._send(api.result_status, {"stored": api.result_status == 200})
                if self.path.endswith("/fail"):
                    return self._send(200, {"recorded": True})
                self._send(404, {"error": "not found"})

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=self.server.serve_forever, daemon=True).start()
        self.url = f"http://127.0.0.1:{self.server.server_port}"

    def paths(self):
        return [p.split("?")[0] for p, _ in self.calls]

    def body(self, suffix):
        return next(b for p, b in self.calls if p.endswith(suffix))

    def close(self):
        self.server.shutdown()
        self.server.server_close()


JOB = {"id": "cnt_1", "audioPath": "/media/raw/content/cnt_1/m_a.mp3", "audioUrl": "/media/raw/content/cnt_1/m_a.mp3?token=t", "attempt": 1}


def fake_transcriber(tmpdir, behaviour):
    """A stand-in for transcribe.py: 'ok' writes a transcript after checking
    it got the downloaded audio; 'undecodable' exits 2; 'crash' exits 1."""
    script = os.path.join(tmpdir, "fake_transcribe.py")
    with open(script, "w") as f:
        f.write(textwrap.dedent(f"""
            import json, sys
            audio, out = sys.argv[1], sys.argv[2]
            assert open(audio, "rb").read() == {AUDIO!r}
            mode = {behaviour!r}
            if mode == "undecodable":
                print("couldn't read the audio: Invalid data", file=sys.stderr); sys.exit(2)
            if mode == "crash":
                print("Killed", file=sys.stderr); sys.exit(1)
            json.dump({{"language": "en", "durationSec": 12.5,
                        "segments": [{{"start": 0.0, "end": 4.2, "text": "Welcome to Galle Fort."}}]}}, open(out, "w"))
        """))
    return f"{sys.executable} {script}"


class WorkerTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.work = os.path.join(self.tmp.name, "work")
        os.mkdir(self.work)

    def make(self, api, behaviour="ok", token=TOKEN):
        self.addCleanup(api.close)
        cfg = worker.Config({
            "API_URL": api.url + "/", "TRANSCRIBER_TOKEN": token, "WORK_DIR": self.work,
            "TRANSCRIBE_CMD": fake_transcriber(self.tmp.name, behaviour),
            "HEARTBEAT_FILE": os.path.join(self.tmp.name, "alive"),
            "DOWNLOAD_CHUNK_BYTES": str(CHUNK),
        })
        return worker.Worker(cfg)

    def test_transcribes_a_job_and_reports_it(self):
        api = FakeAPI(job=dict(JOB))
        w = self.make(api)
        self.assertTrue(w.run_once())
        pieces = -(-len(AUDIO) // CHUNK)
        self.assertGreater(pieces, 2)
        self.assertEqual(api.paths(), [
            "/internal/transcripts/claim", *["/media/raw/content/cnt_1/m_a.mp3"] * pieces, "/internal/transcripts/cnt_1/result",
        ])
        self.assertEqual(api.calls[1][1], f"bytes=0-{CHUNK - 1}")
        result = api.body("/result")
        self.assertEqual(result["audioPath"], JOB["audioPath"])
        self.assertEqual(result["language"], "en")
        self.assertEqual(result["segments"][0]["text"], "Welcome to Galle Fort.")
        self.assertEqual(os.listdir(self.work), [], "temp files left behind")

    def test_downloads_whole_when_ranges_arent_served(self):
        api = FakeAPI(job=dict(JOB), ranges=False)
        self.make(api).run_once()
        self.assertEqual(api.body("/result")["language"], "en")  # the fake checks it got all the audio

    def test_a_cut_off_download_is_retried_not_transcribed(self):
        api = FakeAPI(job=dict(JOB), ranges=False, cut_short=True)
        self.make(api).run_once()
        fail = api.body("/fail")
        self.assertTrue(fail["retryable"])
        self.assertIn("couldn't download", fail["error"])
        self.assertNotIn("/internal/transcripts/cnt_1/result", api.paths())

    def test_nothing_to_do(self):
        api = FakeAPI(job=None)
        w = self.make(api)
        self.assertFalse(w.run_once())
        self.assertEqual(api.paths(), ["/internal/transcripts/claim"])

    def test_a_crash_is_reported_as_retryable(self):
        api = FakeAPI(job=dict(JOB))
        self.make(api, "crash").run_once()
        fail = api.body("/fail")
        self.assertTrue(fail["retryable"])
        self.assertIn("Killed", fail["error"])
        self.assertEqual(fail["audioPath"], JOB["audioPath"])

    def test_undecodable_audio_is_not_retried(self):
        api = FakeAPI(job=dict(JOB))
        self.make(api, "undecodable").run_once()
        fail = api.body("/fail")
        self.assertFalse(fail["retryable"])
        self.assertIn("Invalid data", fail["error"])

    def test_removed_audio_is_not_retried(self):
        api = FakeAPI(job=dict(JOB), audio_status=404)
        self.make(api).run_once()
        self.assertFalse(api.body("/fail")["retryable"])
        self.assertNotIn("/internal/transcripts/cnt_1/result", api.paths())

    def test_a_refused_transcript_is_reported_as_a_failure(self):
        api = FakeAPI(job=dict(JOB), result_status=400)
        self.make(api).run_once()
        self.assertFalse(api.body("/fail")["retryable"])

    def test_wrong_token(self):
        api = FakeAPI(job=dict(JOB))
        w = self.make(api, token="nope")
        with self.assertRaises(worker.ApiError) as ctx:
            w.run_once()
        self.assertEqual(ctx.exception.status, 401)

    def test_heartbeat(self):
        api = FakeAPI(job=None)
        w = self.make(api)
        w.heartbeat()
        self.assertTrue(os.path.exists(w.cfg.heartbeat_file))


if __name__ == "__main__":
    unittest.main()
