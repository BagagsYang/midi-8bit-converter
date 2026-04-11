import io
import json
import sys
import tempfile
import unittest
from pathlib import Path

import pretty_midi

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import app as web_app


class WebFlaskSynthesiseTests(unittest.TestCase):
    def setUp(self):
        web_app.app.testing = True
        self.client = web_app.app.test_client()

    def test_synthesise_accepts_layers_json_and_returns_wav(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "16000",
                "layers_json": json.dumps([{
                    "type": "sine",
                    "duty": 0.5,
                    "volume": 1.0,
                    "frequency_curve": [],
                }]),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertEqual(b"RIFF", response.data[:4])
        self.assertIn("attachment;", response.headers["Content-Disposition"])
        self.assertIn("lead_sine.wav", response.headers["Content-Disposition"])

    def test_synthesise_rejects_invalid_curve_payload(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "16000",
                "layers_json": json.dumps([{
                    "type": "sine",
                    "duty": 0.5,
                    "volume": 1.0,
                    "frequency_curve": [
                        {"frequency_hz": 440.0, "gain_db": 0.0},
                        {"frequency_hz": 440.0, "gain_db": -6.0},
                    ],
                }]),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertIn("strictly increasing", response.get_json()["error"])

    def test_synthesise_uses_curve_hash_in_download_name(self):
        layers = [{
            "type": "sine",
            "duty": 0.5,
            "volume": 1.0,
            "frequency_curve": [
                {"frequency_hz": 261.6255653006, "gain_db": -12.0},
                {"frequency_hz": 1046.5022612024, "gain_db": 0.0},
            ],
        }]
        runtime_layers = web_app.midi_to_wave.normalise_runtime_layers(
            web_app.midi_to_wave.parse_layers_json(json.dumps(layers))
        )
        expected_name = web_app._build_download_name("lead", runtime_layers)

        response = self.client.post(
            "/synthesise",
            data={
                "rate": "16000",
                "layers_json": json.dumps(layers),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn(expected_name, response.headers["Content-Disposition"])

    def test_synthesise_accepts_rounded_web_curve_endpoints(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "16000",
                "layers_json": json.dumps([{
                    "type": "sine",
                    "duty": 0.5,
                    "volume": 1.0,
                    "frequency_curve": [
                        {"frequency_hz": 8.1757989156, "gain_db": 0.0},
                        {"frequency_hz": 12543.8539514, "gain_db": 0.0},
                    ],
                }]),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertEqual(b"RIFF", response.data[:4])

    def _build_midi_bytes(self):
        midi = pretty_midi.PrettyMIDI()
        instrument = pretty_midi.Instrument(program=0)
        instrument.notes.append(pretty_midi.Note(
            velocity=100,
            pitch=69,
            start=0.0,
            end=0.5,
        ))
        midi.instruments.append(instrument)

        with tempfile.TemporaryDirectory() as temp_dir:
            midi_path = Path(temp_dir) / "test.mid"
            midi.write(str(midi_path))
            return midi_path.read_bytes()


if __name__ == "__main__":
    unittest.main()
