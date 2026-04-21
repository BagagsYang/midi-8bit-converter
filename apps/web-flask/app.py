import json
import os
import sys
import tempfile
import webbrowser
from functools import lru_cache
from threading import Timer

from flask import (
    Flask,
    after_this_request,
    jsonify,
    make_response,
    render_template,
    request,
    send_file,
    send_from_directory,
)

APP_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.dirname(os.path.dirname(APP_DIR))
PYTHON_RENDERER_DIR = os.path.join(REPO_ROOT, "core", "python-renderer")
PREVIEW_ASSETS_DIR = os.path.join(REPO_ROOT, "assets", "previews")
I18N_DIR = os.path.join(APP_DIR, "i18n")
SUPPORTED_LOCALES = ("en", "fr", "zh-CN")
DEFAULT_LOCALE = "en"
LOCALE_COOKIE_NAME = "web_locale"
SUPPORTED_LOCALE_LOOKUP = {
    locale.lower(): locale for locale in SUPPORTED_LOCALES
}
LANGUAGE_FALLBACKS = {
    "zh": "zh-CN",
}

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


def _normalise_locale(raw_locale):
    if not raw_locale:
        return None

    locale = raw_locale.strip()
    if not locale:
        return None

    lowered = locale.replace("_", "-").lower()
    if lowered in SUPPORTED_LOCALE_LOOKUP:
        return SUPPORTED_LOCALE_LOOKUP[lowered]

    base_language = lowered.split("-", maxsplit=1)[0]
    if base_language in SUPPORTED_LOCALE_LOOKUP:
        return SUPPORTED_LOCALE_LOOKUP[base_language]
    if base_language in LANGUAGE_FALLBACKS:
        return LANGUAGE_FALLBACKS[base_language]

    return None


@lru_cache(maxsize=None)
def _load_translations(locale):
    with open(os.path.join(I18N_DIR, "en.json"), encoding="utf-8") as file:
        translations = json.load(file)

    if locale != DEFAULT_LOCALE:
        locale_path = os.path.join(I18N_DIR, f"{locale}.json")
        if os.path.exists(locale_path):
            with open(locale_path, encoding="utf-8") as file:
                translations.update(json.load(file))

    return translations


def _resolve_locale():
    requested_locale = _normalise_locale(request.args.get("lang"))
    if requested_locale:
        return requested_locale, True

    cookie_locale = _normalise_locale(request.cookies.get(LOCALE_COOKIE_NAME))
    if cookie_locale:
        return cookie_locale, False

    for accepted_locale, _quality in request.accept_languages:
        matched_locale = _normalise_locale(accepted_locale)
        if matched_locale:
            return matched_locale, False

    return DEFAULT_LOCALE, False


def _get_locale_context():
    locale, is_explicit_override = _resolve_locale()
    return locale, _load_translations(locale), is_explicit_override


def _parse_layers_from_request(form):
    layers_json = (form.get("layers_json") or "").strip()
    if not layers_json:
        layers_json = "[]"

    parsed_layers = midi_to_wave.parse_layers_json(layers_json)
    runtime_layers = midi_to_wave.normalise_runtime_layers(parsed_layers)
    return parsed_layers, runtime_layers


@app.route("/")
def index():
    locale, translations, is_explicit_override = _get_locale_context()
    response = make_response(render_template(
        "index.html",
        default_locale=DEFAULT_LOCALE,
        locale=locale,
        locale_cookie_name=LOCALE_COOKIE_NAME,
        supported_locales=SUPPORTED_LOCALES,
        translations=translations,
    ))

    if is_explicit_override:
        response.set_cookie(
            LOCALE_COOKIE_NAME,
            locale,
            max_age=60 * 60 * 24 * 365,
            samesite="Lax",
        )

    return response


@app.route("/static/previews/<path:filename>")
def preview_asset(filename):
    return send_from_directory(preview_assets_dir, filename)


@app.route("/synthesise", methods=["POST"])
def synthesise():
    _locale, translations, _is_explicit_override = _get_locale_context()

    if "midi_file" not in request.files:
        return jsonify({"error": translations["errors.no_midi_file_uploaded"]}), 400

    file = request.files["midi_file"]
    if file.filename == "":
        return jsonify({"error": translations["errors.no_selected_file"]}), 400

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
            download_name=midi_to_wave.build_output_filename(original_filename, runtime_layers),
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
