import hashlib
import json
import os
import sys
import tempfile
import webbrowser
from threading import Timer

from flask import (
    Flask,
    after_this_request,
    jsonify,
    render_template,
    request,
    send_file,
    send_from_directory,
)

APP_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.dirname(os.path.dirname(APP_DIR))
PYTHON_RENDERER_DIR = os.path.join(REPO_ROOT, "core", "python-renderer")
PREVIEW_ASSETS_DIR = os.path.join(REPO_ROOT, "assets", "previews")

if PYTHON_RENDERER_DIR not in sys.path:
    sys.path.insert(0, PYTHON_RENDERER_DIR)

import midi_to_wave


if getattr(sys, "frozen", False):
    template_folder = os.path.join(sys._MEIPASS, "templates")
    static_folder = os.path.join(sys._MEIPASS, "static")
    preview_assets_dir = os.path.join(static_folder, "previews")
    app = Flask(
        __name__,
        template_folder=template_folder,
        static_folder=static_folder,
    )
else:
    preview_assets_dir = PREVIEW_ASSETS_DIR
    app = Flask(
        __name__,
        template_folder=os.path.join(APP_DIR, "templates"),
        static_folder=os.path.join(APP_DIR, "static"),
    )


def open_browser():
    webbrowser.open_new("http://127.0.0.1:5002")


def _build_curve_payload_hash(layers):
    canonical_payload = json.dumps(layers, sort_keys=True, separators=(",", ":"))
    return hashlib.sha1(canonical_payload.encode("utf-8")).hexdigest()[:8]


def _build_download_name(original_filename, runtime_layers):
    suffix = "mix" if len(runtime_layers) > 1 else runtime_layers[0]["type"]
    if any(layer["frequency_curve"] for layer in runtime_layers):
        suffix = f"{suffix}_{_build_curve_payload_hash(runtime_layers)}"

    return f"{original_filename}_{suffix}.wav"


def _parse_layers_from_request(form):
    layers_json = (form.get("layers_json") or "").strip()
    if not layers_json:
        layers_json = "[]"

    parsed_layers = midi_to_wave.parse_layers_json(layers_json)
    runtime_layers = midi_to_wave.normalise_runtime_layers(parsed_layers)
    return parsed_layers, runtime_layers


@app.route("/")
def index():
    return render_template("index.html")


@app.route("/static/previews/<path:filename>")
def preview_asset(filename):
    return send_from_directory(preview_assets_dir, filename)


@app.route("/synthesise", methods=["POST"])
def synthesise():
    if "midi_file" not in request.files:
        return jsonify({"error": "No MIDI file uploaded"}), 400

    file = request.files["midi_file"]
    if file.filename == "":
        return jsonify({"error": "No selected file"}), 400

    temp_paths = []
    try:
        sample_rate = int(request.form.get("rate", 48000))
        parsed_layers, runtime_layers = _parse_layers_from_request(request.form)

        temp_midi = tempfile.NamedTemporaryFile(delete=False, suffix=".mid")
        temp_wav = tempfile.NamedTemporaryFile(delete=False, suffix=".wav")
        temp_midi.close()
        temp_wav.close()
        temp_paths.extend([temp_midi.name, temp_wav.name])

        file.save(temp_midi.name)
        midi_to_wave.midi_to_audio(
            temp_midi.name,
            temp_wav.name,
            sample_rate,
            parsed_layers,
        )

        original_filename = "output"
        if file.filename:
            original_filename = os.path.splitext(file.filename)[0]

        @after_this_request
        def _cleanup_temp_files(response):
            for temp_path in temp_paths:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass
            return response

        return send_file(
            temp_wav.name,
            as_attachment=True,
            download_name=_build_download_name(original_filename, runtime_layers),
        )
    except ValueError as exc:
        for temp_path in temp_paths:
            try:
                os.unlink(temp_path)
            except FileNotFoundError:
                pass
        return jsonify({"error": str(exc)}), 400
    except Exception as exc:
        for temp_path in temp_paths:
            try:
                os.unlink(temp_path)
            except FileNotFoundError:
                pass
        return jsonify({"error": str(exc)}), 500


if __name__ == "__main__":
    Timer(1, open_browser).start()
    print("DEBUG: SERVER RUNNING ON PORT 5002")
    app.run(host="127.0.0.1", port=5002, debug=False)
