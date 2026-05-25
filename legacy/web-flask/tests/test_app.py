import io
import json
import sqlite3
import sys
import tempfile
import threading
import time
import unittest
from contextlib import closing
from pathlib import Path
from unittest import mock

import pretty_midi

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import app as web_app


class WebFlaskSynthesiseTests(unittest.TestCase):
    def setUp(self):
        web_app.app.testing = True
        self._original_job_config = {
            "SYNTHESISE_JOB_ROOT": web_app.app.config.get("SYNTHESISE_JOB_ROOT"),
            "SYNTHESISE_JOBS_INLINE": web_app.app.config.get("SYNTHESISE_JOBS_INLINE"),
            "WEB_DOWNLOAD_TTL_SECONDS": web_app.app.config.get("WEB_DOWNLOAD_TTL_SECONDS"),
            "MAX_CONTENT_LENGTH": web_app.app.config.get("MAX_CONTENT_LENGTH"),
            "WEB_WORKSPACE_TTL_SECONDS": web_app.app.config.get("WEB_WORKSPACE_TTL_SECONDS"),
            "WEB_WORKSPACE_MAX_QUEUED_FILES": web_app.app.config.get("WEB_WORKSPACE_MAX_QUEUED_FILES"),
            "WEB_WORKSPACE_MAX_UPLOAD_BYTES": web_app.app.config.get("WEB_WORKSPACE_MAX_UPLOAD_BYTES"),
            "WEB_WORKSPACE_MAX_CONVERTED_FILES": web_app.app.config.get("WEB_WORKSPACE_MAX_CONVERTED_FILES"),
            "WEB_RENDER_WORKERS": web_app.app.config.get("WEB_RENDER_WORKERS"),
            "WEB_RENDER_QUEUE_SIZE": web_app.app.config.get("WEB_RENDER_QUEUE_SIZE"),
        }
        self.job_root = tempfile.TemporaryDirectory()
        self.addCleanup(self.job_root.cleanup)
        self.addCleanup(self._restore_job_config)
        web_app.app.config["SYNTHESISE_JOB_ROOT"] = self.job_root.name
        web_app.app.config["SYNTHESISE_JOBS_INLINE"] = True
        web_app.app.config["WEB_DOWNLOAD_TTL_SECONDS"] = 1800
        web_app.app.config["WEB_WORKSPACE_TTL_SECONDS"] = 86400
        web_app.app.config["WEB_WORKSPACE_MAX_QUEUED_FILES"] = 20
        web_app.app.config["WEB_WORKSPACE_MAX_UPLOAD_BYTES"] = 100 * 1024 * 1024
        web_app.app.config["WEB_WORKSPACE_MAX_CONVERTED_FILES"] = 20
        web_app.app.config["WEB_RENDER_WORKERS"] = 2
        web_app.app.config["WEB_RENDER_QUEUE_SIZE"] = 8
        self.client = web_app.app.test_client()

    def _restore_job_config(self):
        for key, value in self._original_job_config.items():
            if value is None:
                web_app.app.config.pop(key, None)
            else:
                web_app.app.config[key] = value

    def test_api_health_returns_ok(self):
        response = self.client.get("/api/health")
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertEqual(
            {
                "status": "ok",
                "service": "octabit-web",
            },
            response.get_json(),
        )

    def test_openapi_contract_lists_browser_and_compatibility_routes(self):
        contract_path = Path(__file__).resolve().parents[3] / "docs" / "openapi.yaml"
        contract = contract_path.read_text(encoding="utf-8")

        self.assertIn("openapi: 3.1.0", contract)
        for route in (
            "/api/health:",
            "/api/workspace:",
            "/api/workspace/uploads:",
            "/api/workspace/uploads/{file_id}:",
            "/api/workspace/queue:",
            "/api/workspace/config:",
            "/api/synthesis-jobs:",
            "/api/synthesis-jobs/{job_id}:",
            "/api/synthesis-jobs/{job_id}/download:",
            "/static/previews/{filename}:",
            "/synthesise:",
            "/synthesise/jobs:",
            "/synthesise/jobs/{job_id}:",
            "/synthesise/jobs/{job_id}/download:",
        ):
            with self.subTest(route=route):
                self.assertIn(route, contract)

        self.assertIn("code: render_queue_full", contract)

    def test_api_workspace_creates_reuses_and_replaces_expired_workspace(self):
        response = self.client.get("/api/workspace")
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn(f"{web_app.WORKSPACE_COOKIE_NAME}=", response.headers.get("Set-Cookie", ""))
        self.assertIn("HttpOnly", response.headers.get("Set-Cookie", ""))
        payload = response.get_json()
        self.assertEqual([], payload["uploads"])
        self.assertEqual([], payload["converted_files"])
        self.assertEqual("octabit.workspace_config.v1", payload["config"]["schema"])
        self.assertEqual(20, payload["limits"]["max_queued_files"])
        self.assertNotIn("id", payload["workspace"])

        second_response = self.client.get("/api/workspace")
        self.addCleanup(second_response.close)
        self.assertEqual(200, second_response.status_code)
        self.assertNotIn(web_app.WORKSPACE_COOKIE_NAME, second_response.headers.get("Set-Cookie", ""))

        db_path = Path(self.job_root.name) / "workspaces.sqlite3"
        with closing(sqlite3.connect(db_path)) as connection:
            with connection:
                connection.execute("UPDATE workspaces SET expires_at = ?", (time.time() - 1,))

        replacement_response = self.client.get("/api/workspace")
        self.addCleanup(replacement_response.close)
        self.assertEqual(200, replacement_response.status_code)
        self.assertIn(f"{web_app.WORKSPACE_COOKIE_NAME}=", replacement_response.headers.get("Set-Cookie", ""))
        self.assertEqual([], replacement_response.get_json()["uploads"])

    def test_api_workspace_upload_restore_reorder_delete_and_quota(self):
        self.client.get("/api/workspace").close()

        first_response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "first.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(first_response.close)
        second_response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "second.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(second_response.close)

        self.assertEqual(201, first_response.status_code)
        self.assertEqual(201, second_response.status_code)
        first_upload = first_response.get_json()["upload"]
        second_upload = second_response.get_json()["upload"]
        self.assertEqual(32, len(first_upload["file_id"]))
        self.assertNotIn("id", first_upload)
        self.assertTrue(self._workspace_upload_path(first_upload["file_id"]).exists())

        reorder_response = self.client.patch(
            "/api/workspace/queue",
            json={"file_ids": [second_upload["file_id"], first_upload["file_id"]]},
        )
        self.addCleanup(reorder_response.close)
        self.assertEqual(200, reorder_response.status_code)
        self.assertEqual(
            [second_upload["file_id"], first_upload["file_id"]],
            [upload["file_id"] for upload in reorder_response.get_json()["uploads"]],
        )

        restored_response = self.client.get("/api/workspace")
        self.addCleanup(restored_response.close)
        self.assertEqual(
            ["second.mid", "first.mid"],
            [upload["name"] for upload in restored_response.get_json()["uploads"]],
        )

        delete_response = self.client.delete(f"/api/workspace/uploads/{first_upload['file_id']}")
        self.addCleanup(delete_response.close)
        self.assertEqual(204, delete_response.status_code)
        self.assertEqual([], self._workspace_upload_paths(first_upload["file_id"]))

        web_app.app.config["WEB_WORKSPACE_MAX_QUEUED_FILES"] = 1
        quota_response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "third.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(quota_response.close)
        self.assertEqual(409, quota_response.status_code)
        self.assertEqual("workspace_queue_limit", quota_response.get_json()["error"]["code"])

    def test_api_workspace_upload_total_bytes_limit(self):
        self.client.get("/api/workspace").close()
        web_app.app.config["WEB_WORKSPACE_MAX_UPLOAD_BYTES"] = 4

        response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "too-large.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(413, response.status_code)
        self.assertEqual("workspace_upload_bytes_limit", response.get_json()["error"]["code"])

    def test_api_workspace_config_round_trips_canonical_json(self):
        self.client.get("/api/workspace").close()
        config = self._workspace_config()
        config["sample_rate"] = 44100
        config["layers"].append({
            "type": "sine",
            "duty": 0.5,
            "volume": 0.75,
            "curve_enabled": True,
            "frequency_curve": [
                {
                    "frequency_hz": web_app.midi_to_wave.MIN_CURVE_FREQUENCY_HZ,
                    "gain_db": -6.0,
                },
                {
                    "frequency_hz": web_app.midi_to_wave.MAX_CURVE_FREQUENCY_HZ,
                    "gain_db": 0.0,
                },
            ],
        })

        response = self.client.put("/api/workspace/config", json=config)
        self.addCleanup(response.close)
        self.assertEqual(200, response.status_code)
        saved_config = response.get_json()["config"]
        self.assertEqual(44100, saved_config["sample_rate"])
        self.assertEqual(2, len(saved_config["layers"]))
        self.assertTrue(saved_config["layers"][1]["curve_enabled"])

        restored_response = self.client.get("/api/workspace")
        self.addCleanup(restored_response.close)
        self.assertEqual(saved_config, restored_response.get_json()["config"])

        invalid_response = self.client.put("/api/workspace/config", json={"schema": "wrong"})
        self.addCleanup(invalid_response.close)
        self.assertEqual(422, invalid_response.status_code)
        self.assertEqual("invalid_workspace_config", invalid_response.get_json()["error"]["code"])

    def test_api_workspace_resource_routes_require_active_workspace(self):
        valid_id = "0" * 32
        for method, path in (
            ("delete", f"/api/workspace/uploads/{valid_id}"),
            ("get", f"/api/synthesis-jobs/{valid_id}"),
            ("delete", f"/api/synthesis-jobs/{valid_id}"),
            ("get", f"/api/synthesis-jobs/{valid_id}/download"),
        ):
            with self.subTest(method=method, path=path):
                client = web_app.app.test_client()
                response = getattr(client, method)(path)
                response.close()
                self.assertEqual(410, response.status_code)
                self.assertEqual("workspace_expired", response.get_json()["error"]["code"])

    def test_api_synthesis_job_accepts_workspace_file_id_and_enforces_ownership(self):
        self.client.get("/api/workspace").close()
        upload_response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(upload_response.close)
        file_id = upload_response.get_json()["upload"]["file_id"]

        response = self.client.post(
            "/api/synthesis-jobs",
            json={
                "file_id": file_id,
                "config": self._workspace_config(),
            },
        )
        self.addCleanup(response.close)

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("ready", payload["status"])
        self.assertEqual("lead_pulse.wav", payload["download_name"])
        self.assertNotIn("workspace_id", payload)
        self.assertNotIn("id", payload)
        output_path = self._workspace_output_path(payload["job_id"])
        self.assertTrue(output_path.exists())

        other_client = web_app.app.test_client()
        other_client.get("/api/workspace").close()
        not_owned_response = other_client.get(f"/api/synthesis-jobs/{payload['job_id']}")
        not_owned_response.close()
        self.assertEqual(404, not_owned_response.status_code)
        self.assertEqual("not_found", not_owned_response.get_json()["error"]["code"])

        download_response = self.client.get(payload["download_url"])
        self.addCleanup(download_response.close)
        self.assertEqual(200, download_response.status_code)
        self.assertEqual(b"RIFF", download_response.data[:4])

        delete_response = self.client.delete(payload["delete_url"])
        self.addCleanup(delete_response.close)
        self.assertEqual(204, delete_response.status_code)
        self.assertFalse(output_path.exists())

    def test_api_workspace_converted_file_limit(self):
        self.client.get("/api/workspace").close()
        web_app.app.config["WEB_WORKSPACE_MAX_CONVERTED_FILES"] = 1
        upload_response = self.client.post(
            "/api/workspace/uploads",
            data={"midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid")},
            content_type="multipart/form-data",
        )
        self.addCleanup(upload_response.close)
        file_id = upload_response.get_json()["upload"]["file_id"]

        first_response = self.client.post(
            "/api/synthesis-jobs",
            json={"file_id": file_id, "config": self._workspace_config()},
        )
        self.addCleanup(first_response.close)
        self.assertEqual(202, first_response.status_code)

        quota_response = self.client.post(
            "/api/synthesis-jobs",
            json={"file_id": file_id, "config": self._workspace_config()},
        )
        self.addCleanup(quota_response.close)
        self.assertEqual(409, quota_response.status_code)
        self.assertEqual("workspace_converted_limit", quota_response.get_json()["error"]["code"])

    def test_workspace_sqlite_connection_pragmas(self):
        service = web_app._workspace_service()
        with service.connect() as connection:
            foreign_keys = connection.execute("PRAGMA foreign_keys").fetchone()[0]
            journal_mode = connection.execute("PRAGMA journal_mode").fetchone()[0]
            busy_timeout = connection.execute("PRAGMA busy_timeout").fetchone()[0]

        self.assertEqual(1, foreign_keys)
        self.assertEqual("wal", journal_mode.lower())
        self.assertEqual(5000, busy_timeout)

    def test_api_synthesis_job_returns_api_status_download_and_delete_links(self):
        response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
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

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("ready", payload["status"])
        self.assertEqual("lead_sine.wav", payload["download_name"])
        self.assertIn("/api/synthesis-jobs/", payload["download_url"])
        self.assertIn("/api/synthesis-jobs/", payload["delete_url"])
        self.assertNotIn("/synthesise/jobs/", payload["download_url"])
        self.assertNotIn("/synthesise/jobs/", payload["delete_url"])

        status_response = self.client.get(f"/api/synthesis-jobs/{payload['job_id']}")
        self.addCleanup(status_response.close)
        self.assertEqual(200, status_response.status_code)
        status_payload = status_response.get_json()
        self.assertEqual("ready", status_payload["status"])
        self.assertIn("/api/synthesis-jobs/", status_payload["download_url"])

        download_response = self.client.get(payload["download_url"])
        self.addCleanup(download_response.close)
        self.assertEqual(200, download_response.status_code)
        self.assertEqual(b"RIFF", download_response.data[:4])
        self.assertIn("lead_sine.wav", download_response.headers["Content-Disposition"])

        output_path = self._workspace_output_path(payload["job_id"])
        self.assertTrue(output_path.exists())
        delete_response = self.client.delete(payload["delete_url"])
        self.addCleanup(delete_response.close)
        self.assertEqual(204, delete_response.status_code)
        self.assertFalse(output_path.exists())

        expired_response = self.client.get(f"/api/synthesis-jobs/{payload['job_id']}")
        self.addCleanup(expired_response.close)
        self.assertEqual(404, expired_response.status_code)
        self.assertEqual("not_found", expired_response.get_json()["error"]["code"])

    def test_api_synthesis_job_returns_structured_validation_errors(self):
        missing_response = self.client.post(
            "/api/synthesis-jobs",
            data={},
            content_type="multipart/form-data",
        )
        self.addCleanup(missing_response.close)
        self.assertEqual(400, missing_response.status_code)
        self.assertEqual("missing_midi_file", missing_response.get_json()["error"]["code"])

        extension_response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.txt"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(extension_response.close)
        self.assertEqual(415, extension_response.status_code)
        self.assertEqual("unsupported_file_type", extension_response.get_json()["error"]["code"])

        rate_response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "16000",
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(rate_response.close)
        self.assertEqual(422, rate_response.status_code)
        self.assertEqual("invalid_sample_rate", rate_response.get_json()["error"]["code"])

        layers_response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
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
        self.addCleanup(layers_response.close)
        self.assertEqual(422, layers_response.status_code)
        self.assertEqual("invalid_layers", layers_response.get_json()["error"]["code"])

        empty_response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(b""), "empty.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(empty_response.close)
        self.assertEqual(400, empty_response.status_code)
        self.assertEqual("empty_midi_file", empty_response.get_json()["error"]["code"])
        self.assertEqual([], list(Path(self.job_root.name).iterdir()))

    def test_api_synthesis_job_marks_oversized_render_as_failed(self):
        response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
                "midi_file": (
                    io.BytesIO(self._build_midi_bytes([
                        (
                            69,
                            web_app.midi_to_wave.MAX_RENDER_SECONDS,
                            web_app.midi_to_wave.MAX_RENDER_SECONDS + 1,
                        ),
                    ])),
                    "too-long.mid",
                ),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("failed", payload["status"])
        self.assertIn("duration", payload["error"])

    def test_api_synthesis_job_rejects_more_than_four_layers(self):
        layers = [
            {
                "type": "pulse",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [],
            }
            for _index in range(web_app.midi_to_wave.MAX_RENDER_LAYERS + 1)
        ]

        response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
                "layers_json": json.dumps(layers),
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(422, response.status_code)
        self.assertEqual("invalid_layers", response.get_json()["error"]["code"])

    def test_api_synthesis_job_rejects_invalid_job_ids(self):
        for method, path in (
            ("get", "/api/synthesis-jobs/not-a-job"),
            ("delete", "/api/synthesis-jobs/not-a-job"),
            ("get", "/api/synthesis-jobs/not-a-job/download"),
        ):
            with self.subTest(method=method, path=path):
                response = getattr(self.client, method)(path)
                self.addCleanup(response.close)
                self.assertEqual(400, response.status_code)
                self.assertEqual("invalid_job_id", response.get_json()["error"]["code"])

    def test_api_synthesis_job_rejects_oversized_upload_with_structured_error(self):
        web_app.app.config["MAX_CONTENT_LENGTH"] = 128
        response = self.client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(b"x" * 512), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(413, response.status_code)
        self.assertEqual("upload_too_large", response.get_json()["error"]["code"])

    def test_index_falls_back_to_english(self):
        response = self.client.get("/", headers={"Accept-Language": "de-DE,de;q=0.9"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn('<html lang="en"', response.get_data(as_text=True))
        self.assertIn("OctaBit", response.get_data(as_text=True))
        self.assertIn("Current File(s)", response.get_data(as_text=True))

    def test_index_uses_browser_language_for_french(self):
        response = self.client.get("/", headers={"Accept-Language": "fr-FR,fr;q=0.9,en;q=0.8"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        body = response.get_data(as_text=True)
        self.assertIn('<html lang="fr"', body)
        self.assertIn("Fichier(s) actuel(s)", body)
        self.assertIn("Traiter et télécharger", body)
        self.assertIn('value="fr" data-i18n="toolbar.language_option.fr" selected', body)
        self.assertIn("Français", body)

    def test_index_uses_browser_language_for_simplified_chinese(self):
        response = self.client.get("/", headers={"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        body = response.get_data(as_text=True)
        self.assertIn('<html lang="zh-CN"', body)
        self.assertIn("当前文件", body)
        self.assertIn("采样率", body)

    def test_index_query_parameter_overrides_browser_language(self):
        response = self.client.get("/?lang=fr", headers={"Accept-Language": "zh-CN,zh;q=0.9"})
        self.addCleanup(response.close)

        self.assertEqual(200, response.status_code)
        self.assertIn("Fichier(s) actuel(s)", response.get_data(as_text=True))
        self.assertIn("web_locale=fr", response.headers.get("Set-Cookie", ""))

    def test_index_renders_theme_select_with_language_select(self):
        response = self.client.get("/")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertIn("OctaBit", body)
        self.assertIn("OctaBit – MIDI to 8-bit Converter", body)
        self.assertIn("https://octabit.cc/", body)
        self.assertIn("Converted Files", body)
        self.assertIn('id="convertedList"', body)
        self.assertIn('id="themeSelect"', body)
        self.assertIn('class="theme-select-frame"', body)
        self.assertIn('id="themeSelectIcon"', body)
        self.assertLess(body.index('id="themeSelectIcon"'), body.index('id="themeSelect"'))
        self.assertIn('aria-label="Theme"', body)
        self.assertIn('id="languageSelect"', body)
        self.assertIn('class="language-select-frame"', body)
        self.assertIn('class="language-select-icon"', body)
        self.assertLess(body.index('class="language-select-icon"'), body.index('id="languageSelect"'))
        self.assertLess(body.index('id="themeSelect"'), body.index('id="languageSelect"'))
        self.assertIn('href="https://github.com/bagags/octabit"', body)
        self.assertIn('class="github-link"', body)
        self.assertIn('class="github-link-icon"', body)
        self.assertIn('aria-label="GitHub repository"', body)
        self.assertLess(body.index('id="languageSelect"'), body.index('class="github-link"'))
        self.assertIn('<option value="system" data-i18n="settings.theme_system">System</option>', body)
        self.assertIn('<option value="light" data-i18n="settings.theme_light">Light</option>', body)
        self.assertIn('<option value="dark" data-i18n="settings.theme_dark">Dark</option>', body)
        self.assertIn("Theme", body)
        self.assertIn("Light", body)
        self.assertIn("Dark", body)
        self.assertIn("System", body)
        self.assertIn('class="module output-module"', body)
        self.assertIn('id="octabit-config"', body)
        self.assertIn('"translationsByLocale"', body)
        self.assertIn('data-i18n="queue.title"', body)
        self.assertIn("/static/js/theme-init.js", body)
        self.assertIn("/static/css/app.css", body)
        self.assertIn("/static/js/lucide-icons.js", body)
        self.assertIn("/static/js/app.js", body)
        self.assertLess(body.index("/static/js/theme-init.js"), body.index("/static/css/app.css"))
        self.assertLess(body.index("/static/js/lucide-icons.js"), body.index("/static/js/app.js"))
        self.assertIn("Process &amp; Download", body)
        self.assertIn(
            "Clearing converted files deletes the temporary WAV files from the server, "
            "so they cannot be retained or downloaded again.",
            body,
        )
        self.assertNotIn("<style>", body)
        self.assertNotIn('class="action-rail"', body)
        self.assertNotIn('id="settingsPlaceholderBtn"', body)
        self.assertNotIn('id="settingsButton"', body)
        self.assertNotIn('id="settingsDialog"', body)
        self.assertNotIn('name="themeChoice"', body)
        self.assertNotIn('id="themeToggle"', body)
        self.assertNotIn('id="followSystemThemeToggle"', body)

    def test_static_browser_script_preserves_interaction_hooks(self):
        response = self.client.get("/static/js/app.js")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn("document.getElementById('octabit-config')", body)
        self.assertIn("translateStaticSurface", body)
        self.assertIn("TRANSLATIONS_BY_LOCALE[nextLocale]", body)
        self.assertIn("window.history.replaceState", body)
        self.assertIn("WORKSPACE_API_URL = '/api/workspace'", body)
        self.assertIn("restoreWorkspace", body)
        self.assertIn("fetch(`${WORKSPACE_API_URL}/uploads`", body)
        self.assertIn("fetch(`${WORKSPACE_API_URL}/config`", body)
        self.assertIn("SYNTHESIS_JOBS_API_URL = '/api/synthesis-jobs'", body)
        self.assertIn("fetch(SYNTHESIS_JOBS_API_URL", body)
        self.assertIn("fetch(`${SYNTHESIS_JOBS_API_URL}/${jobId}`)", body)
        self.assertIn("responseErrorMessage(errorPayload, response.statusText)", body)
        self.assertIn("payload.error.message || payload.error.code", body)
        self.assertNotIn("fetch('/synthesise/jobs'", body)
        self.assertNotIn("fetch(`/synthesise/jobs/${jobId}`)", body)
        self.assertNotIn("persistLanguageSwitchState", body)
        self.assertNotIn("restoreLanguageSwitchState", body)
        self.assertNotIn("pendingLanguageSwitchState", body)
        self.assertNotIn("window.addEventListener('beforeunload'", body)
        self.assertIn("SUPPORTED_LOCALES.includes(selectedLocale)", body)
        self.assertIn("window.confirm(t('converted.clear_confirm'))", body)
        self.assertIn("layer-control-grid", body)
        self.assertIn("document.getElementById('themeSelect')", body)
        self.assertIn("document.getElementById('themeSelectIcon')", body)
        self.assertIn("document.querySelector('.language-select-icon')", body)
        self.assertIn("document.querySelector('.github-link-icon')", body)
        self.assertIn("window.octabitLucideIcons", body)
        self.assertIn("ICONS.svg('github'", body)
        self.assertIn("ICONS.svg('languages'", body)
        self.assertIn("ICONS.svg('x')", body)
        self.assertIn("ICONS.svg('play')", body)
        self.assertIn("ICONS.svg(selectedThemeIconName()", body)
        self.assertIn("activeLayerTypes(layerIndex).has(value)", body)
        self.assertIn("firstUnusedWaveType()", body)
        self.assertIn("${usedTypes.has(value) ? 'disabled' : ''}", body)
        self.assertIn("layers[layerCount].type = unusedType", body)
        self.assertIn("'sun'", body)
        self.assertIn("'moon-star'", body)
        self.assertIn("activeThemeValue() === 'light' ? 'sun' : 'moon-star'", body)
        self.assertIn("octabitTheme", body)
        self.assertIn("localStorage.setItem(THEME_STORAGE_KEY, theme)", body)
        self.assertIn("localStorage.removeItem(THEME_STORAGE_KEY)", body)
        self.assertIn("themeSelect.value === 'system'", body)
        self.assertIn("prefers-color-scheme: dark", body)
        self.assertIn("prefers-color-scheme: light", body)
        self.assertIn("applyTheme();", body)
        self.assertNotIn("htmlElement.setAttribute('data-bs-theme', 'dark')", body)
        self.assertNotIn("themeToggle", body)
        self.assertNotIn("settingsButton", body)
        self.assertNotIn("settingsDialog", body)
        self.assertNotIn("themeChoice", body)
        self.assertNotIn("followSystemThemeToggle", body)
        self.assertNotIn("updateThemeIcon", body)
        self.assertNotIn("removeButton.textContent = 'x'", body)
        self.assertNotIn("play-icon", body)

    def test_lucide_icon_helper_exposes_vendored_inline_svgs(self):
        response = self.client.get("/static/js/lucide-icons.js")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn("window.octabitLucideIcons", body)
        self.assertIn("License: ISC License", body)
        self.assertIn("Hugeicons GitHub icon", body)
        self.assertIn("hugeiconsAttributes", body)
        self.assertIn("github:", body)
        self.assertIn("languages:", body)
        self.assertIn("'moon-star':", body)
        self.assertIn("play:", body)
        self.assertIn("sun:", body)
        self.assertIn("x:", body)
        self.assertIn('aria-hidden="true"', body)
        self.assertIn('focusable="false"', body)

    def test_app_css_uses_lucide_icon_rule_not_css_triangle(self):
        response = self.client.get("/static/css/app.css")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn(".lucide-icon", body)
        self.assertNotIn(".play-icon", body)
        self.assertNotIn("border-left: 10px solid currentColor", body)

    def test_toolbar_theme_and_language_selects_do_not_show_focus_halo(self):
        response = self.client.get("/static/css/app.css")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn(".theme-select:focus,\n.language-select:focus", body)
        self.assertIn("box-shadow: none;", body)
        self.assertIn(".theme-select:focus-visible,\n.language-select:focus-visible", body)
        self.assertIn("outline: none;", body)
        self.assertIn(".control-select:focus", body)
        self.assertIn("box-shadow: 0 0 0 2px var(--accent-ring);", body)

    def test_theme_select_icon_uses_square_control(self):
        response = self.client.get("/static/css/app.css")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn(".theme-select-frame", body)
        self.assertIn("width: 38px;", body)
        self.assertIn("flex: 0 0 38px;", body)
        self.assertIn(".theme-select-icon", body)
        self.assertIn("position: absolute;", body)
        self.assertIn("pointer-events: none;", body)
        self.assertIn("font-size: 0;", body)
        self.assertIn("background-image: none;", body)
        self.assertIn("-webkit-appearance: none;", body)

    def test_language_select_icon_matches_theme_square_position(self):
        response = self.client.get("/static/css/app.css")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn(".theme-select-frame,\n.language-select-frame", body)
        self.assertIn(".theme-select,\n.language-select {\n    width: 100%;", body)
        self.assertIn(".theme-select-icon,\n.language-select-icon", body)
        self.assertIn(".theme-select-icon .lucide-icon,\n.language-select-icon .lucide-icon", body)
        self.assertIn("width: 38px;", body)
        self.assertIn("flex: 0 0 38px;", body)
        self.assertIn("left: 50%;", body)
        self.assertIn("transform: translate(-50%, -50%);", body)
        self.assertIn(".theme-select-frame:hover .theme-select-icon", body)
        self.assertIn(".language-select-frame:hover .language-select-icon", body)
        self.assertIn(".theme-select-frame:focus-within .theme-select-icon", body)
        self.assertIn(".language-select-frame:focus-within .language-select-icon", body)

    def test_theme_init_script_resolves_stored_or_system_theme(self):
        response = self.client.get("/static/js/theme-init.js")
        self.addCleanup(response.close)

        body = response.get_data(as_text=True)
        self.assertEqual(200, response.status_code)
        self.assertIn("octabitTheme", body)
        self.assertIn("THEME_STORAGE_KEY = 'octabitTheme'", body)
        self.assertIn("THEME_VALUES = ['light', 'dark']", body)
        self.assertIn("THEME_VALUES.includes(value)", body)
        self.assertIn("localStorage.getItem(THEME_STORAGE_KEY)", body)
        self.assertIn("prefers-color-scheme: dark", body)
        self.assertIn("prefers-color-scheme: light", body)
        self.assertIn("data-bs-theme", body)

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

    def test_synthesise_rejects_empty_midi_file(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(b""), "empty.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertEqual("Uploaded MIDI file is empty.", response.get_json()["error"])

    def test_synthesise_rejects_oversized_render_before_wav_response(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "44100",
                "midi_file": (
                    io.BytesIO(self._build_midi_bytes([
                        (
                            69,
                            web_app.midi_to_wave.MAX_RENDER_SECONDS,
                            web_app.midi_to_wave.MAX_RENDER_SECONDS + 1,
                        ),
                    ])),
                    "too-long.mid",
                ),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertIn("duration", response.get_json()["error"])

    def test_render_executor_rejects_jobs_when_capacity_is_full(self):
        started = threading.Event()
        release = threading.Event()
        executor = web_app.synthesis_jobs.BoundedRenderExecutor(
            max_workers=1,
            max_queue_size=0,
        )

        def blocking_job():
            started.set()
            release.wait(2)

        future = executor.submit(blocking_job)
        self.assertTrue(started.wait(1))

        try:
            with self.assertRaises(web_app.synthesis_jobs.RenderQueueFull):
                executor.submit(lambda: None)
        finally:
            release.set()
            future.result(timeout=2)

    def test_synthesise_accepts_layers_json_and_returns_wav(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "44100",
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

    def test_synthesise_job_rejects_empty_midi_before_starting_job(self):
        response = self.client.post(
            "/synthesise/jobs?lang=zh-CN",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(b""), "empty.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertEqual(
            "上传的 MIDI 文件为空。请重新添加原始文件后再试。",
            response.get_json()["error"],
        )
        self.assertEqual([], list(Path(self.job_root.name).iterdir()))

    def test_synthesise_job_rejects_unsupported_extension_before_starting_job(self):
        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.txt"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertEqual(
            "Unsupported file type. Upload a .mid or .midi file.",
            response.get_json()["error"],
        )
        self.assertEqual([], list(Path(self.job_root.name).iterdir()))

    def test_synthesise_job_rejects_invalid_rate_before_starting_job(self):
        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "16000",
                "midi_file": (io.BytesIO(self._build_midi_bytes()), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(400, response.status_code)
        self.assertIn("Unsupported sample rate", response.get_json()["error"])
        self.assertEqual([], list(Path(self.job_root.name).iterdir()))

    def test_synthesise_job_rejects_invalid_layers_before_starting_job(self):
        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
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
        self.assertEqual([], list(Path(self.job_root.name).iterdir()))

    def test_synthesise_rejects_oversized_upload(self):
        web_app.app.config["MAX_CONTENT_LENGTH"] = 128
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "44100",
                "midi_file": (io.BytesIO(b"x" * 512), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        self.addCleanup(response.close)

        self.assertEqual(413, response.status_code)
        self.assertEqual("Uploaded file is too large.", response.get_json()["error"])

    def test_synthesise_job_returns_ready_status_and_download(self):
        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
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

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("ready", payload["status"])
        self.assertEqual("lead_sine.wav", payload["download_name"])
        self.assertGreater(payload["size_bytes"], 4)
        self.assertIn("created_at", payload)
        self.assertIn("updated_at", payload)
        self.assertIn("expires_at", payload)
        self.assertIn("/synthesise/jobs/", payload["download_url"])
        self.assertIn("/synthesise/jobs/", payload["delete_url"])

        status_response = self.client.get(f"/synthesise/jobs/{payload['job_id']}")
        self.addCleanup(status_response.close)
        self.assertEqual(200, status_response.status_code)
        self.assertEqual("ready", status_response.get_json()["status"])

        download_response = self.client.get(payload["download_url"])
        self.addCleanup(download_response.close)
        self.assertEqual(200, download_response.status_code)
        self.assertEqual(b"RIFF", download_response.data[:4])
        self.assertIn("attachment;", download_response.headers["Content-Disposition"])
        self.assertIn("lead_sine.wav", download_response.headers["Content-Disposition"])

    def test_synthesise_job_delete_removes_ready_download(self):
        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
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

        payload = response.get_json()
        output_path = Path(web_app._job_dir(payload["job_id"])) / "output.wav"
        self.assertTrue(output_path.exists())

        delete_response = self.client.delete(payload["delete_url"])
        self.addCleanup(delete_response.close)
        self.assertEqual(204, delete_response.status_code)
        self.assertFalse(output_path.exists())

        status_response = self.client.get(f"/synthesise/jobs/{payload['job_id']}")
        self.addCleanup(status_response.close)
        self.assertEqual(410, status_response.status_code)
        self.assertEqual("expired", status_response.get_json()["status"])

        download_response = self.client.get(payload["download_url"])
        self.addCleanup(download_response.close)
        self.assertEqual(410, download_response.status_code)
        self.assertEqual("expired", download_response.get_json()["status"])

    def test_synthesise_job_reports_failed_status(self):
        with (
            mock.patch.object(web_app, "_render_uploaded_wav", side_effect=ValueError("renderer exploded")),
            self.assertLogs(web_app.app.logger.name, level="WARNING") as logs,
        ):
            response = self.client.post(
                "/synthesise/jobs",
                data={
                    "rate": "44100",
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

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("failed", payload["status"])
        self.assertEqual("renderer exploded", payload["error"])
        self.assertTrue(any("renderer exploded" in message for message in logs.output))

        download_response = self.client.get(f"/synthesise/jobs/{payload['job_id']}/download")
        self.addCleanup(download_response.close)
        self.assertEqual(400, download_response.status_code)
        self.assertEqual("failed", download_response.get_json()["status"])

    def test_synthesise_job_reports_exception_type_for_empty_error(self):
        with (
            mock.patch.object(web_app, "_render_uploaded_wav", side_effect=MemoryError()),
            self.assertLogs(web_app.app.logger.name, level="ERROR") as logs,
        ):
            response = self.client.post(
                "/synthesise/jobs",
                data={
                    "rate": "44100",
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

        self.assertEqual(202, response.status_code)
        payload = response.get_json()
        self.assertEqual("failed", payload["status"])
        self.assertEqual("MemoryError: synthesis ran out of memory", payload["error"])
        self.assertTrue(any("Synthesis job" in message for message in logs.output))

    def test_synthesise_job_missing_or_expired_returns_expired(self):
        missing_response = self.client.get(f"/synthesise/jobs/{'0' * 32}")
        self.addCleanup(missing_response.close)
        self.assertEqual(410, missing_response.status_code)
        self.assertEqual("expired", missing_response.get_json()["status"])

        response = self.client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
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
        payload = response.get_json()
        metadata = web_app._read_job_metadata(payload["job_id"])
        metadata["expires_at"] = time.time() - 1
        web_app._write_job_metadata(payload["job_id"], metadata)

        expired_response = self.client.get(f"/synthesise/jobs/{payload['job_id']}")
        self.addCleanup(expired_response.close)
        self.assertEqual(410, expired_response.status_code)
        self.assertEqual("expired", expired_response.get_json()["status"])

    def test_synthesise_rejects_invalid_curve_payload(self):
        response = self.client.post(
            "/synthesise",
            data={
                "rate": "44100",
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
                "rate": "44100",
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
                "rate": "44100",
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
                "rate": "44100",
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

    def _build_midi_bytes(self, note_specs=None):
        note_specs = note_specs or [(69, 0.0, 0.5)]
        midi = pretty_midi.PrettyMIDI()
        instrument = pretty_midi.Instrument(program=0)
        for pitch, start, end in note_specs:
            instrument.notes.append(pretty_midi.Note(
                velocity=100,
                pitch=pitch,
                start=start,
                end=end,
            ))
        midi.instruments.append(instrument)

        with tempfile.TemporaryDirectory() as temp_dir:
            midi_path = Path(temp_dir) / "test.mid"
            midi.write(str(midi_path))
            return midi_path.read_bytes()

    def _workspace_config(self):
        return {
            "schema": "octabit.workspace_config.v1",
            "sample_rate": 48000,
            "layers": [{
                "type": "pulse",
                "duty": 0.5,
                "volume": 1.0,
                "curve_enabled": False,
                "frequency_curve": [
                    {
                        "frequency_hz": web_app.midi_to_wave.MIN_CURVE_FREQUENCY_HZ,
                        "gain_db": 0.0,
                    },
                    {
                        "frequency_hz": web_app.midi_to_wave.MAX_CURVE_FREQUENCY_HZ,
                        "gain_db": 0.0,
                    },
                ],
            }],
        }

    def _workspace_upload_path(self, file_id):
        matches = self._workspace_upload_paths(file_id)
        if len(matches) != 1:
            self.fail(f"Expected one upload path for {file_id}, found {matches}")
        return matches[0]

    def _workspace_upload_paths(self, file_id):
        return list(Path(self.job_root.name).glob(f"workspaces/*/uploads/{file_id}.mid"))

    def _workspace_output_path(self, job_id):
        matches = list(Path(self.job_root.name).glob(f"workspaces/*/jobs/{job_id}/output.wav"))
        if len(matches) != 1:
            self.fail(f"Expected one output path for {job_id}, found {matches}")
        return matches[0]

    def _load_catalog(self, locale):
        catalog_path = Path(web_app.I18N_DIR) / f"{locale}.json"
        with catalog_path.open(encoding="utf-8") as file:
            return json.load(file)


if __name__ == "__main__":
    unittest.main()
