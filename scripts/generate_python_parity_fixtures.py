#!/usr/bin/env python3
# Copyright (C) 2026 Jeremy Yang
# SPDX-License-Identifier: AGPL-3.0-or-later

import argparse
import hashlib
import json
import os
import re
import shutil
import sys
import tempfile
from pathlib import Path
from unittest import mock

import pretty_midi


REPO_ROOT = Path(__file__).resolve().parents[1]
WEB_FLASK_DIR = REPO_ROOT / "legacy" / "web-flask"
PYTHON_RENDERER_DIR = REPO_ROOT / "legacy" / "python-renderer"
DEFAULT_OUTPUT_DIR = REPO_ROOT / "backend" / "testdata" / "python-baseline"
OPAQUE_ID_RE = re.compile(r"\b[0-9a-f]{32}\b")
TIMESTAMP_KEYS = {"created_at", "updated_at", "expires_at"}
OPAQUE_ID_KEYS = {"file_id", "job_id"}

sys.path.insert(0, str(WEB_FLASK_DIR))
sys.path.insert(0, str(PYTHON_RENDERER_DIR))

import app as web_app  # noqa: E402
import midi_to_wave  # noqa: E402
import synthesis_jobs  # noqa: E402
import workspaces  # noqa: E402


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    return sha256_bytes(path.read_bytes())


def write_json(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(payload, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def write_bytes(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(payload)


def write_baseline_readme(output_dir):
    (output_dir / "README.md").write_text(
        """# Python Baseline Fixtures

These fixtures freeze the current Flask backend and Python renderer behavior for
the Go backend migration. They are parity inputs for future Go tests, not a new
runtime dependency.

Regenerate them only when intentionally updating the Python baseline:

```bash
./.venv/bin/python3 scripts/generate_python_parity_fixtures.py
```

The generator rewrites this directory from the current checkout. Routine
post-migration Go tests should read the generated fixture files directly and
must not require Python unless they are explicitly fixture-regeneration tests.

Current fixture groups:

- `api/implemented_routes.json`: Python baseline transcript for routes already
  implemented by the Go backend.
- `api/workspace_flow.json`, `api/error_responses.json`, and
  `api/legacy_jobs.json`: fuller API parity targets for future workspace/job
  implementation.
- `renderer/expectations.json` and `renderer/*.wav`: renderer naming, curve,
  parsed note-event, and WAV-output parity targets.
""",
        encoding="utf-8",
    )


def build_midi_bytes(note_specs):
    midi = pretty_midi.PrettyMIDI()
    instrument = pretty_midi.Instrument(program=0)
    for pitch, start, end, velocity in note_specs:
        instrument.notes.append(pretty_midi.Note(
            velocity=velocity,
            pitch=pitch,
            start=start,
            end=end,
        ))
    midi.instruments.append(instrument)

    with tempfile.TemporaryDirectory() as temp_dir:
        midi_path = Path(temp_dir) / "fixture.mid"
        midi.write(str(midi_path))
        return midi_path.read_bytes()


def render_wav(midi_path, output_path, sample_rate, layers):
    output_path.parent.mkdir(parents=True, exist_ok=True)
    midi_to_wave.midi_to_audio(
        str(midi_path),
        str(output_path),
        sample_rate=sample_rate,
        layers=layers,
    )


def parsed_midi_note_specs(midi_path):
    midi_data = pretty_midi.PrettyMIDI(str(midi_path))
    note_specs = []
    for instrument in midi_data.instruments:
        if instrument.is_drum:
            continue
        for note in instrument.notes:
            note_specs.append({
                "pitch": note.pitch,
                "start": note.start,
                "end": note.end,
                "velocity": note.velocity,
            })
    note_specs.sort(key=lambda note: (note["start"], note["pitch"], note["end"], note["velocity"]))
    return note_specs


class ResponseNormaliser:
    def __init__(self):
        self.replacements = {}
        self.next_index = 1

    def replacement_for(self, value, key=None):
        if value not in self.replacements:
            label = "id"
            if key in OPAQUE_ID_KEYS:
                label = key
            self.replacements[value] = f"<{label}:{self.next_index}>"
            self.next_index += 1
        return self.replacements[value]

    def normalise_text(self, value, key=None):
        def replace(match):
            return self.replacement_for(match.group(0), key)

        return OPAQUE_ID_RE.sub(replace, value)

    def normalise(self, value, key=None):
        if key in TIMESTAMP_KEYS and isinstance(value, (int, float)):
            return "<timestamp>"
        if key in OPAQUE_ID_KEYS and isinstance(value, str):
            return self.replacement_for(value, key)
        if isinstance(value, dict):
            for child_key, child_value in value.items():
                if child_key in OPAQUE_ID_KEYS and isinstance(child_value, str):
                    self.replacement_for(child_value, child_key)
            return {
                child_key: self.normalise(child_value, child_key)
                for child_key, child_value in value.items()
            }
        if isinstance(value, list):
            return [self.normalise(item, key) for item in value]
        if isinstance(value, str):
            return self.normalise_text(value, key)
        return value

    def normalise_headers(self, response):
        headers = {
            "Content-Type": response.headers.get("Content-Type"),
        }
        cookie = response.headers.get("Set-Cookie")
        if cookie:
            normalised_cookie = re.sub(
                r"octabit_workspace=[^;]+",
                "octabit_workspace=<workspace-token>",
                cookie,
            )
            normalised_cookie = re.sub(
                r"Expires=[^;]+",
                "Expires=<timestamp>",
                normalised_cookie,
            )
            headers["Set-Cookie"] = normalised_cookie
        disposition = response.headers.get("Content-Disposition")
        if disposition:
            headers["Content-Disposition"] = disposition
        return {key: value for key, value in headers.items() if value}


def response_record(label, response, normaliser, binary=False):
    record = {
        "label": label,
        "status": response.status_code,
        "headers": normaliser.normalise_headers(response),
    }
    if binary:
        record["body"] = {
            "sha256": sha256_bytes(response.data),
            "size": len(response.data),
            "prefix_hex": response.data[:12].hex(),
        }
    else:
        payload = response.get_json(silent=True)
        record["body"] = normaliser.normalise(payload)
    return record


def workspace_config():
    return {
        "schema": web_app.WORKSPACE_CONFIG_SCHEMA,
        "sample_rate": 48000,
        "layers": [{
            "type": "pulse",
            "duty": 0.5,
            "volume": 1.0,
            "curve_enabled": False,
            "frequency_curve": [
                {
                    "frequency_hz": midi_to_wave.MIN_CURVE_FREQUENCY_HZ,
                    "gain_db": 0.0,
                },
                {
                    "frequency_hz": midi_to_wave.MAX_CURVE_FREQUENCY_HZ,
                    "gain_db": 0.0,
                },
            ],
        }],
    }


def curve_workspace_config():
    return {
        "schema": web_app.WORKSPACE_CONFIG_SCHEMA,
        "sample_rate": 44100,
        "layers": [
            {
                "type": "pulse",
                "duty": 0.25,
                "volume": 0.75,
                "curve_enabled": True,
                "frequency_curve": [
                    {"frequency_hz": 110.0, "gain_db": -6.0},
                    {"frequency_hz": 440.0, "gain_db": 0.0},
                    {"frequency_hz": 880.0, "gain_db": 3.0},
                ],
            },
            {
                "type": "triangle",
                "duty": 0.5,
                "volume": 0.4,
                "curve_enabled": False,
                "frequency_curve": [],
            },
        ],
    }


def generate_renderer_fixtures(output_dir, midi_files, midi_manifest):
    renderer_dir = output_dir / "renderer"
    simple_layers = midi_to_wave.normalise_runtime_layers([])
    curve_layers = [
        {
            "type": "pulse",
            "duty": 0.25,
            "volume": 0.75,
            "frequency_curve": [
                {"frequency_hz": 110.0, "gain_db": -6.0},
                {"frequency_hz": 440.0, "gain_db": 0.0},
                {"frequency_hz": 880.0, "gain_db": 3.0},
            ],
        },
        {
            "type": "triangle",
            "duty": 0.5,
            "volume": 0.4,
            "frequency_curve": [],
        },
    ]

    cases = [
        {
            "name": "simple_pulse_48000",
            "midi": "simple.mid",
            "note_specs": midi_manifest["simple.mid"]["note_specs"],
            "sample_rate": 48000,
            "layers": simple_layers,
            "output_name": midi_to_wave.build_output_filename("simple", simple_layers),
        },
        {
            "name": "curve_mix_44100",
            "midi": "melody.mid",
            "note_specs": midi_manifest["melody.mid"]["note_specs"],
            "sample_rate": 44100,
            "layers": curve_layers,
            "output_name": midi_to_wave.build_output_filename("melody", curve_layers),
            "curve_hash": midi_to_wave.build_curve_payload_hash(
                midi_to_wave.normalise_runtime_layers(curve_layers),
            ),
            "curve_gain_samples": {
                str(frequency): midi_to_wave.evaluate_frequency_curve_gain_db(
                    curve_layers[0]["frequency_curve"],
                    frequency,
                )
                for frequency in (55.0, 220.0, 440.0, 660.0, 1760.0)
            },
        },
    ]

    for case in cases:
        wav_path = renderer_dir / f"{case['name']}.wav"
        render_wav(
            midi_files[case["midi"]],
            wav_path,
            case["sample_rate"],
            case["layers"],
        )
        case["wav"] = {
            "path": f"renderer/{wav_path.name}",
            "sha256": sha256_file(wav_path),
            "size": wav_path.stat().st_size,
        }

    invalid_layer_errors = {}
    invalid_layer_inputs = {
        "duplicate_curve_frequency": [{
            "type": "sine",
            "duty": 0.5,
            "volume": 1.0,
            "frequency_curve": [
                {"frequency_hz": 440.0, "gain_db": 0.0},
                {"frequency_hz": 440.0, "gain_db": -6.0},
            ],
        }],
        "too_many_layers": [
            {"type": "pulse", "duty": 0.5, "volume": 1.0, "frequency_curve": []}
            for _index in range(midi_to_wave.MAX_RENDER_LAYERS + 1)
        ],
    }
    for name, layers in invalid_layer_inputs.items():
        try:
            midi_to_wave.parse_layers_json(json.dumps(layers))
        except ValueError as exc:
            invalid_layer_errors[name] = str(exc)

    expectations = {
        "cases": cases,
        "invalid_layer_errors": invalid_layer_errors,
        "limits": {
            "max_render_seconds": midi_to_wave.MAX_RENDER_SECONDS,
            "max_midi_notes": midi_to_wave.MAX_MIDI_NOTES,
            "max_render_layers": midi_to_wave.MAX_RENDER_LAYERS,
            "normalised_peak": midi_to_wave.NORMALISED_PEAK,
        },
    }
    write_json(renderer_dir / "expectations.json", expectations)
    return expectations


def generate_config_fixtures(output_dir):
    raw_config = {
        "schema": web_app.WORKSPACE_CONFIG_SCHEMA,
        "sample_rate": "48000",
        "layers": [{
            "type": "pulse",
            "duty": 0.333333,
            "volume": 1.23456,
            "curve_enabled": True,
            "frequency_curve": [
                {"frequency_hz": midi_to_wave.MAX_CURVE_FREQUENCY_HZ, "gain_db": 12.0},
                {"frequency_hz": midi_to_wave.MIN_CURVE_FREQUENCY_HZ, "gain_db": -6.123456},
            ],
        }],
    }
    normalised = web_app._normalise_workspace_config(raw_config)
    form_payload = web_app._workspace_config_to_form_payload(normalised)
    from_form = web_app._workspace_config_from_form(form_payload)

    payload = {
        "raw_config": raw_config,
        "normalised_config": normalised,
        "form_payload": form_payload,
        "config_from_form": from_form,
    }
    write_json(output_dir / "config" / "workspace_config_normalisation.json", payload)
    return payload


class FixtureAppConfig:
    def __init__(self):
        self.original = {
            key: web_app.app.config.get(key)
            for key in (
                "SYNTHESISE_JOB_ROOT",
                "SYNTHESISE_JOBS_INLINE",
                "WEB_DOWNLOAD_TTL_SECONDS",
                "MAX_CONTENT_LENGTH",
                "WEB_WORKSPACE_TTL_SECONDS",
                "WEB_WORKSPACE_MAX_QUEUED_FILES",
                "WEB_WORKSPACE_MAX_UPLOAD_BYTES",
                "WEB_WORKSPACE_MAX_CONVERTED_FILES",
                "WEB_RENDER_WORKERS",
                "WEB_RENDER_QUEUE_SIZE",
            )
        }
        self.temp_dir = tempfile.TemporaryDirectory()

    def __enter__(self):
        web_app.app.testing = True
        web_app.app.config["SYNTHESISE_JOB_ROOT"] = self.temp_dir.name
        web_app.app.config["SYNTHESISE_JOBS_INLINE"] = True
        web_app.app.config["WEB_DOWNLOAD_TTL_SECONDS"] = 1800
        web_app.app.config["WEB_WORKSPACE_TTL_SECONDS"] = 86400
        web_app.app.config["WEB_WORKSPACE_MAX_QUEUED_FILES"] = 20
        web_app.app.config["WEB_WORKSPACE_MAX_UPLOAD_BYTES"] = 100 * 1024 * 1024
        web_app.app.config["WEB_WORKSPACE_MAX_CONVERTED_FILES"] = 20
        web_app.app.config["WEB_RENDER_WORKERS"] = 2
        web_app.app.config["WEB_RENDER_QUEUE_SIZE"] = 8
        return self

    def __exit__(self, exc_type, exc, traceback):
        for key, value in self.original.items():
            if value is None:
                web_app.app.config.pop(key, None)
            else:
                web_app.app.config[key] = value
        self.temp_dir.cleanup()


def generate_api_flow_transcript(output_dir, midi_bytes):
    normaliser = ResponseNormaliser()
    records = []
    with FixtureAppConfig():
        client = web_app.app.test_client()

        response = client.get("/api/workspace")
        records.append(response_record("workspace_start", response, normaliser))

        response = client.post(
            "/api/workspace/uploads",
            data={"midi_file": (spooled_file(midi_bytes["simple.mid"]), "first.mid")},
            content_type="multipart/form-data",
        )
        first_upload = response.get_json()["upload"]
        records.append(response_record("upload_first", response, normaliser))

        response = client.post(
            "/api/workspace/uploads",
            data={"midi_file": (spooled_file(midi_bytes["melody.mid"]), "second.mid")},
            content_type="multipart/form-data",
        )
        second_upload = response.get_json()["upload"]
        records.append(response_record("upload_second", response, normaliser))

        response = client.patch(
            "/api/workspace/queue",
            json={"file_ids": [second_upload["file_id"], first_upload["file_id"]]},
        )
        records.append(response_record("reorder_queue", response, normaliser))

        config = curve_workspace_config()
        response = client.put("/api/workspace/config", json=config)
        records.append(response_record("save_config", response, normaliser))

        response = client.post(
            "/api/synthesis-jobs",
            json={"file_id": second_upload["file_id"], "config": config},
        )
        job = response.get_json()
        records.append(response_record("create_synthesis_job", response, normaliser))

        response = client.get(f"/api/synthesis-jobs/{job['job_id']}")
        records.append(response_record("poll_synthesis_job", response, normaliser))

        response = client.get(f"/api/synthesis-jobs/{job['job_id']}/download")
        records.append(response_record("download_synthesis_job", response, normaliser, binary=True))

        response = client.delete(f"/api/synthesis-jobs/{job['job_id']}")
        records.append(response_record("delete_synthesis_job", response, normaliser))

        response = client.delete(f"/api/workspace/uploads/{first_upload['file_id']}")
        records.append(response_record("delete_upload", response, normaliser))

    transcript = {
        "description": "Normalised Python Flask API happy-path transcript.",
        "dynamic_values": "Opaque ids, workspace cookie values, and timestamps are replaced with placeholders.",
        "records": records,
    }
    write_json(output_dir / "api" / "workspace_flow.json", transcript)
    return transcript


def generate_implemented_routes_transcript(output_dir):
    normaliser = ResponseNormaliser()
    records = []
    with FixtureAppConfig():
        client = web_app.app.test_client()

        response = client.get("/api/health")
        records.append(response_record("api_health", response, normaliser))

        response = client.get("/static/previews/pulse_50.wav")
        records.append(response_record("preview_pulse_50", response, normaliser, binary=True))

    transcript = {
        "description": "Normalised Python Flask transcript for routes currently implemented by the Go backend.",
        "dynamic_values": "Binary bodies are recorded by hash, size, and prefix.",
        "records": records,
    }
    write_json(output_dir / "api" / "implemented_routes.json", transcript)
    return transcript


def spooled_file(data):
    file_obj = tempfile.SpooledTemporaryFile()
    file_obj.write(data)
    file_obj.seek(0)
    return file_obj


def generate_api_error_transcript(output_dir, midi_bytes):
    normaliser = ResponseNormaliser()
    records = []
    with FixtureAppConfig():
        client = web_app.app.test_client()
        valid_id = "0123456789abcdef0123456789abcdef"

        response = client.delete(f"/api/workspace/uploads/{valid_id}")
        records.append(response_record("workspace_expired_without_cookie", response, normaliser))

        client.get("/api/workspace").close()

        response = client.post("/api/workspace/uploads", data={}, content_type="multipart/form-data")
        records.append(response_record("missing_upload_file", response, normaliser))

        response = client.post(
            "/api/workspace/uploads",
            data={"midi_file": (spooled_file(midi_bytes["simple.mid"]), "lead.txt")},
            content_type="multipart/form-data",
        )
        records.append(response_record("unsupported_upload_extension", response, normaliser))

        response = client.put("/api/workspace/config", json={"schema": "wrong"})
        records.append(response_record("invalid_workspace_config", response, normaliser))

        response = client.post(
            "/api/synthesis-jobs",
            json={"file_id": "not-a-file", "config": workspace_config()},
        )
        records.append(response_record("invalid_json_file_id", response, normaliser))

        response = client.post(
            "/api/synthesis-jobs",
            data={
                "rate": "16000",
                "midi_file": (spooled_file(midi_bytes["simple.mid"]), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        records.append(response_record("invalid_multipart_sample_rate", response, normaliser))

        response = client.post(
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
                "midi_file": (spooled_file(midi_bytes["simple.mid"]), "lead.mid"),
            },
            content_type="multipart/form-data",
        )
        records.append(response_record("invalid_multipart_layers", response, normaliser))

        upload_response = client.post(
            "/api/workspace/uploads",
            data={"midi_file": (spooled_file(midi_bytes["simple.mid"]), "queue.mid")},
            content_type="multipart/form-data",
        )
        upload_id = upload_response.get_json()["upload"]["file_id"]
        with mock.patch.object(
            workspaces.WorkspaceService,
            "start_job",
            side_effect=synthesis_jobs.RenderQueueFull(
                "The render queue is full. Try again after current jobs finish."
            ),
        ):
            response = client.post(
                "/api/synthesis-jobs",
                json={"file_id": upload_id, "config": workspace_config()},
            )
        records.append(response_record("render_queue_full", response, normaliser))

    transcript = {
        "description": "Normalised Python Flask API error response transcript.",
        "dynamic_values": "Opaque ids, workspace cookie values, and timestamps are replaced with placeholders.",
        "records": records,
    }
    write_json(output_dir / "api" / "error_responses.json", transcript)
    return transcript


def generate_legacy_transcript(output_dir, midi_bytes):
    normaliser = ResponseNormaliser()
    records = []
    with FixtureAppConfig():
        client = web_app.app.test_client()

        response = client.post(
            "/synthesise/jobs",
            data={
                "rate": "44100",
                "midi_file": (spooled_file(midi_bytes["simple.mid"]), "legacy.mid"),
            },
            content_type="multipart/form-data",
        )
        job = response.get_json()
        records.append(response_record("create_legacy_job", response, normaliser))

        response = client.get(f"/synthesise/jobs/{job['job_id']}")
        records.append(response_record("poll_legacy_job", response, normaliser))

        response = client.get(f"/synthesise/jobs/{job['job_id']}/download")
        records.append(response_record("download_legacy_job", response, normaliser, binary=True))

        response = client.delete(f"/synthesise/jobs/{job['job_id']}")
        records.append(response_record("delete_legacy_job", response, normaliser))

    transcript = {
        "description": "Normalised Python Flask legacy compatibility route transcript.",
        "dynamic_values": "Opaque ids and timestamps are replaced with placeholders.",
        "records": records,
    }
    write_json(output_dir / "api" / "legacy_jobs.json", transcript)
    return transcript


def generate_fixtures(output_dir):
    if output_dir.exists():
        shutil.rmtree(output_dir)
    output_dir.mkdir(parents=True)
    write_baseline_readme(output_dir)

    midi_note_specs = {
        "simple.mid": [(69, 0.0, 0.5, 100)],
        "melody.mid": [
            (60, 0.0, 0.25, 96),
            (64, 0.3, 0.55, 104),
            (67, 0.6, 0.9, 112),
        ],
    }
    midi_bytes = {
        name: build_midi_bytes(note_specs)
        for name, note_specs in midi_note_specs.items()
    }
    midi_files = {}
    midi_manifest = {}
    for name, data in midi_bytes.items():
        path = output_dir / "midi" / name
        write_bytes(path, data)
        midi_files[name] = path
        midi_manifest[name] = {
            "path": f"midi/{name}",
            "note_specs": parsed_midi_note_specs(path),
            "sha256": sha256_bytes(data),
            "size": len(data),
        }

    renderer_expectations = generate_renderer_fixtures(output_dir, midi_files, midi_manifest)
    config_expectations = generate_config_fixtures(output_dir)
    implemented_routes = generate_implemented_routes_transcript(output_dir)
    api_flow = generate_api_flow_transcript(output_dir, midi_bytes)
    api_errors = generate_api_error_transcript(output_dir, midi_bytes)
    legacy_api = generate_legacy_transcript(output_dir, midi_bytes)

    artifact_hashes = {
        "README.md": sha256_file(output_dir / "README.md"),
        "api/implemented_routes.json": sha256_file(output_dir / "api" / "implemented_routes.json"),
        "api/workspace_flow.json": sha256_file(output_dir / "api" / "workspace_flow.json"),
        "api/error_responses.json": sha256_file(output_dir / "api" / "error_responses.json"),
        "api/legacy_jobs.json": sha256_file(output_dir / "api" / "legacy_jobs.json"),
        "config/workspace_config_normalisation.json": sha256_file(
            output_dir / "config" / "workspace_config_normalisation.json"
        ),
        "renderer/expectations.json": sha256_file(output_dir / "renderer" / "expectations.json"),
    }
    for case in renderer_expectations["cases"]:
        artifact_hashes[case["wav"]["path"]] = sha256_file(output_dir / case["wav"]["path"])

    manifest = {
        "schema": "octabit.python_baseline_fixtures.v1",
        "source": "Generated from the current Flask backend and Python renderer.",
        "regenerate_command": "./.venv/bin/python3 scripts/generate_python_parity_fixtures.py",
        "python_runtime_only": True,
        "midi": midi_manifest,
        "artifacts": artifact_hashes,
        "summary": {
            "renderer_cases": [case["name"] for case in renderer_expectations["cases"]],
            "config_cases": list(config_expectations.keys()),
            "implemented_route_records": len(implemented_routes["records"]),
            "api_flow_records": len(api_flow["records"]),
            "api_error_records": len(api_errors["records"]),
            "legacy_api_records": len(legacy_api["records"]),
        },
    }
    write_json(output_dir / "manifest.json", manifest)


def main(argv=None):
    parser = argparse.ArgumentParser(
        description="Regenerate Python baseline fixtures for the Go backend migration.",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=DEFAULT_OUTPUT_DIR,
        help=f"Fixture output directory. Default: {DEFAULT_OUTPUT_DIR}",
    )
    args = parser.parse_args(argv)

    generate_fixtures(args.output_dir)
    print(f"Generated Python parity fixtures in {args.output_dir}")


if __name__ == "__main__":
    main()
