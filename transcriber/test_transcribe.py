"""Tests for transcribe.py's audio handling with the real decoder (PyAV) and
a stand-in for the Whisper model. Skipped where PyAV isn't installed.
python -m unittest -v"""

import json
import os
import subprocess
import sys
import tempfile
import types
import unittest

try:
    import av  # noqa: F401
    import numpy as np
except ImportError:  # pragma: no cover
    av = None

import transcribe


def make_mp3(path, seconds):
    """Writes a silent-ish MP3 of the given length with PyAV."""
    with av.open(path, "w") as out:
        stream = out.add_stream("mp3", rate=16000)
        stream.layout = "mono"
        samples = np.zeros((1, 1152), dtype=np.int16)
        n = 0
        while n < seconds * 16000:
            frame = av.AudioFrame.from_ndarray(samples, format="s16", layout="mono")
            frame.sample_rate = 16000
            frame.pts = n
            for packet in stream.encode(frame):
                out.mux(packet)
            n += samples.shape[1]
        for packet in stream.encode(None):
            out.mux(packet)


class FakeModel:
    """Says one line per piece, detecting Sinhala on the first piece."""

    def __init__(self):
        self.languages = []

    def transcribe(self, samples, beam_size, vad_filter, language):
        self.languages.append(language)
        seconds = samples.size / transcribe.SAMPLE_RATE
        seg = types.SimpleNamespace(start=1.0, end=min(3.0, seconds), text="  hello  ")
        empty = types.SimpleNamespace(start=3.0, end=3.5, text="  ")
        return iter([seg, empty]), types.SimpleNamespace(language="si")


@unittest.skipIf(av is None, "PyAV isn't installed")
class TranscribeTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def test_long_audio_is_handled_in_ten_minute_pieces(self):
        path = os.path.join(self.tmp.name, "long.mp3")
        make_mp3(path, 25 * 60)
        pieces = [(start, samples.size / 16000) for start, samples in transcribe.audio_chunks(path)]
        self.assertEqual([round(s) for s, _ in pieces], [0, 600, 1200])
        self.assertAlmostEqual(sum(d for _, d in pieces), 1500, delta=1)

        model = FakeModel()
        result = transcribe.transcribe(path, model)
        self.assertEqual(result["language"], "si")
        self.assertEqual(model.languages, [None, "si", "si"])  # detected once, then kept
        # Pieces end on whole audio frames, so a start can be a few
        # hundredths past the round number.
        for got, expected in zip([s["start"] for s in result["segments"]], [1.0, 601.0, 1201.0], strict=True):
            self.assertAlmostEqual(got, expected, delta=0.1)
        self.assertTrue(all(s["text"] == "hello" for s in result["segments"]))
        self.assertAlmostEqual(result["durationSec"], 1500, delta=1)

    def test_a_language_setting_skips_detection(self):
        path = os.path.join(self.tmp.name, "short.mp3")
        make_mp3(path, 5)
        model = FakeModel()
        result = transcribe.transcribe(path, model, language="en")
        self.assertEqual(model.languages, ["en"])
        self.assertEqual(result["language"], "en")

    def test_a_file_that_isnt_audio_exits_2_without_loading_the_model(self):
        path = os.path.join(self.tmp.name, "notes.txt")
        with open(path, "w") as f:
            f.write("not audio at all")
        env = dict(os.environ, MODEL="no-such-model", MODEL_OFFLINE="1")
        proc = subprocess.run([sys.executable, transcribe.__file__, path, os.path.join(self.tmp.name, "o.json")],
                              capture_output=True, text=True, env=env)
        self.assertEqual(proc.returncode, transcribe.EXIT_UNDECODABLE)
        self.assertIn("couldn't read the audio", proc.stderr)


if __name__ == "__main__":
    unittest.main()
