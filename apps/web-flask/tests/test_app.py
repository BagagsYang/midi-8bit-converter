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

    def test_index_falls_back_to_english(self):
        response = self.client.get("/", headers={"Accept-Language": "de-DE,de;q=0.9"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn('<html lang="en"', response.get_data(as_text=True))
        self.assertIn("MIDI Queue", response.get_data(as_text=True))

    def test_index_uses_browser_language_for_french(self):
        response = self.client.get("/", headers={"Accept-Language": "fr-FR,fr;q=0.9,en;q=0.8"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        body = response.get_data(as_text=True)
        self.assertIn('<html lang="fr"', body)
        self.assertIn("File MIDI", body)
        self.assertIn("Synthétiser la file", body)
        self.assertIn('value="fr" selected', body)
        self.assertIn("Français", body)

    def test_index_uses_browser_language_for_simplified_chinese(self):
        response = self.client.get("/", headers={"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        body = response.get_data(as_text=True)
        self.assertIn('<html lang="zh-CN"', body)
        self.assertIn("MIDI 队列", body)
        self.assertIn("采样率", body)

    def test_index_query_parameter_overrides_browser_language(self):
        response = self.client.get("/?lang=fr", headers={"Accept-Language": "zh-CN,zh;q=0.9"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn("File MIDI", response.get_data(as_text=True))
        self.assertIn("web_locale=fr", response.headers.get("Set-Cookie", ""))

    def test_index_includes_language_switch_state_preservation_script(self):
        response = self.client.get("/")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertIn("persistLanguageSwitchState", body)
        self.assertIn("restoreLanguageSwitchState", body)
        self.assertIn("pendingLanguageSwitchState", body)
        self.assertIn("SUPPORTED_LOCALES.includes(selectedLocale)", body)

    def test_supported_locale_catalogs_have_matching_keys(self):
        base_keys = set(self._load_catalog(web_app.DEFAULT_LOCALE))
        self.assertEqual(
            {"en", "fr", "zh-CN"},
            set(web_app.SUPPORTED_LOCALES),
        )

        for locale in web_app.SUPPORTED_LOCALES:
            with self.subTest(locale=locale):
                self.assertEqual(base_keys, set(self._load_catalog(locale)))

    def test_synthesise_localises_missing_file_error_from_cookie(self):
        self.client.set_cookie(web_app.LOCALE_COOKIE_NAME, "zh-CN")
        response = self.client.post(
            "/synthesise",
            data={},
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertEqual("未上传 MIDI 文件", response.get_json()["error"])

    def test_synthesise_localises_missing_file_error_from_french_cookie(self):
        self.client.set_cookie(web_app.LOCALE_COOKIE_NAME, "fr")
        response = self.client.post(
            "/synthesise",
            data={},
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertEqual("Aucun fichier MIDI envoyé", response.get_json()["error"])

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
        expected_name = web_app.midi_to_wave.build_output_filename("lead", runtime_layers)

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

    def test_synthesise_accepts_four_layer_mix(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "16000",
                "layers_json": json.dumps([
                    {
                        "type": "pulse",
                        "duty": 0.5,
                        "volume": 1.0,
                        "frequency_curve": [],
                    },
                    {
                        "type": "sine",
                        "duty": 0.5,
                        "volume": 0.5,
                        "frequency_curve": [],
                    },
                    {
                        "type": "triangle",
                        "duty": 0.5,
                        "volume": 0.5,
                        "frequency_curve": [],
                    },
                    {
                        "type": "sawtooth",
                        "duty": 0.5,
                        "volume": 0.5,
                        "frequency_curve": [],
                    },
                ]),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertEqual(b"RIFF", response.data[:4])
        self.assertIn("lead_mix.wav", response.headers["Content-Disposition"])

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

    def _load_catalog(self, locale):
        catalog_path = Path(web_app.I18N_DIR) / f"{locale}.json"
        with catalog_path.open(encoding="utf-8") as file:
            return json.load(file)


if __name__ == "__main__":
    unittest.main()
