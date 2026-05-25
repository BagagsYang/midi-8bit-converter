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
from werkzeug.exceptions import RequestEntityTooLarge

APP_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.dirname(os.path.dirname(APP_DIR))
PYTHON_RENDERER_DIR = os.path.join(REPO_ROOT, "legacy", "python-renderer")
PREVIEW_ASSETS_DIR = os.path.join(REPO_ROOT, "assets", "previews")
I18N_DIR = os.path.join(APP_DIR, "i18n")
SUPPORTED_LOCALES = ("en", "fr", "zh-CN")
DEFAULT_LOCALE = "en"
LOCALE_COOKIE_NAME = "web_locale"
WORKSPACE_COOKIE_NAME = "octabit_workspace"
SUPPORTED_LOCALE_LOOKUP = {
    locale.lower(): locale for locale in SUPPORTED_LOCALES
}
LANGUAGE_FALLBACKS = {
    "zh": "zh-CN",
}
DEFAULT_DOWNLOAD_TTL_SECONDS = 30 * 60
DEFAULT_WORKSPACE_TTL_SECONDS = 24 * 60 * 60
DEFAULT_WORKSPACE_MAX_QUEUED_FILES = 20
DEFAULT_WORKSPACE_MAX_UPLOAD_BYTES = 100 * 1024 * 1024
DEFAULT_WORKSPACE_MAX_CONVERTED_FILES = 20
DEFAULT_JOB_ROOT = os.path.join(tempfile.gettempdir(), "octabit-jobs")
DEFAULT_MAX_UPLOAD_BYTES = 20 * 1024 * 1024
DEFAULT_RENDER_WORKERS = 2
DEFAULT_RENDER_QUEUE_SIZE = 8
ALLOWED_MIDI_EXTENSIONS = {".mid", ".midi"}
ALLOWED_SAMPLE_RATES = {44100, 48000, 96000}
WORKSPACE_CONFIG_SCHEMA = "octabit.workspace_config.v1"
LEGACY_SYNTHESIS_JOBS_ROUTE = "/synthesise/jobs"
API_SYNTHESIS_JOBS_ROUTE = "/api/synthesis-jobs"

if PYTHON_RENDERER_DIR not in sys.path:
    sys.path.insert(0, PYTHON_RENDERER_DIR)

import midi_to_wave
import synthesis_jobs
import workspaces


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


def _get_default_max_upload_bytes():
    raw_limit = os.environ.get("WEB_MAX_UPLOAD_BYTES", DEFAULT_MAX_UPLOAD_BYTES)
    try:
        return max(1, int(raw_limit))
    except (TypeError, ValueError):
        return DEFAULT_MAX_UPLOAD_BYTES


app.config["MAX_CONTENT_LENGTH"] = _get_default_max_upload_bytes()


def open_browser(port=5002):
    webbrowser.open_new(f"http://127.0.0.1:{port}")


def _get_server_port(default=5002):
    try:
        return int(os.environ.get("PORT", default))
    except ValueError:
        return default


def _get_download_ttl_seconds():
    raw_ttl = app.config.get(
        "WEB_DOWNLOAD_TTL_SECONDS",
        os.environ.get("WEB_DOWNLOAD_TTL_SECONDS", DEFAULT_DOWNLOAD_TTL_SECONDS),
    )
    try:
        return max(1, int(raw_ttl))
    except (TypeError, ValueError):
        return DEFAULT_DOWNLOAD_TTL_SECONDS


def _get_positive_int_config(config_key, env_key, default):
    raw_value = app.config.get(config_key, os.environ.get(env_key, default))
    try:
        return max(1, int(raw_value))
    except (TypeError, ValueError):
        return default


def _get_workspace_ttl_seconds():
    return _get_positive_int_config(
        "WEB_WORKSPACE_TTL_SECONDS",
        "WEB_WORKSPACE_TTL_SECONDS",
        DEFAULT_WORKSPACE_TTL_SECONDS,
    )


def _get_workspace_max_queued_files():
    return _get_positive_int_config(
        "WEB_WORKSPACE_MAX_QUEUED_FILES",
        "WEB_WORKSPACE_MAX_QUEUED_FILES",
        DEFAULT_WORKSPACE_MAX_QUEUED_FILES,
    )


def _get_workspace_max_upload_bytes():
    return _get_positive_int_config(
        "WEB_WORKSPACE_MAX_UPLOAD_BYTES",
        "WEB_WORKSPACE_MAX_UPLOAD_BYTES",
        DEFAULT_WORKSPACE_MAX_UPLOAD_BYTES,
    )


def _get_workspace_max_converted_files():
    return _get_positive_int_config(
        "WEB_WORKSPACE_MAX_CONVERTED_FILES",
        "WEB_WORKSPACE_MAX_CONVERTED_FILES",
        DEFAULT_WORKSPACE_MAX_CONVERTED_FILES,
    )


def _get_job_root():
    return app.config.get(
        "SYNTHESISE_JOB_ROOT",
        os.environ.get("WEB_SYNTHESISE_JOB_ROOT", DEFAULT_JOB_ROOT),
    )


def _get_render_workers():
    return _get_positive_int_config(
        "WEB_RENDER_WORKERS",
        "WEB_RENDER_WORKERS",
        DEFAULT_RENDER_WORKERS,
    )


def _get_render_queue_size():
    raw_value = app.config.get(
        "WEB_RENDER_QUEUE_SIZE",
        os.environ.get("WEB_RENDER_QUEUE_SIZE", DEFAULT_RENDER_QUEUE_SIZE),
    )
    try:
        return max(0, int(raw_value))
    except (TypeError, ValueError):
        return DEFAULT_RENDER_QUEUE_SIZE


def _job_service():
    return synthesis_jobs.SynthesisJobService(
        job_root=_get_job_root(),
        download_ttl_seconds=_get_download_ttl_seconds(),
        run_inline=bool(app.config.get("SYNTHESISE_JOBS_INLINE")),
        logger=app.logger,
        max_workers=_get_render_workers(),
        max_queue_size=_get_render_queue_size(),
    )


def _workspace_service():
    return workspaces.WorkspaceService(
        job_root=_get_job_root(),
        workspace_ttl_seconds=_get_workspace_ttl_seconds(),
        max_queued_files=_get_workspace_max_queued_files(),
        max_upload_bytes=_get_workspace_max_upload_bytes(),
        max_converted_files=_get_workspace_max_converted_files(),
        default_config=_default_workspace_config(),
        run_inline=bool(app.config.get("SYNTHESISE_JOBS_INLINE")),
        logger=app.logger,
        max_workers=_get_render_workers(),
        max_queue_size=_get_render_queue_size(),
    )


def _json_success(payload=None, status_code=200):
    return jsonify(payload or {}), status_code


def _json_accepted(payload):
    return _json_success(payload, 202)


def _json_api_error(code, message, status_code):
    return jsonify({
        "error": {
            "code": code,
            "message": message,
        }
    }), status_code


def _json_legacy_error(message):
    return jsonify({"error": message})


def _api_error_response(code, message, status_code):
    return _json_api_error(code, message, status_code)


def _api_error(code, message, status_code):
    response = jsonify({
        "error": {
            "code": code,
            "message": message,
        }
    })
    response.status_code = status_code
    return response


def _set_workspace_cookie(response, workspace):
    response.set_cookie(
        WORKSPACE_COOKIE_NAME,
        workspace.token,
        max_age=_get_workspace_ttl_seconds(),
        httponly=True,
        samesite="Lax",
    )
    return response


def _json_workspace(payload, status_code=200, workspace=None):
    response = jsonify(payload)
    response.status_code = status_code
    if workspace is not None and workspace.created:
        _set_workspace_cookie(response, workspace)
    return response


def _active_workspace_or_error():
    workspace = _workspace_service().get_active_workspace(
        request.cookies.get(WORKSPACE_COOKIE_NAME),
    )
    if workspace is None:
        return None, _api_error(
            "workspace_expired",
            "The temporary workspace has expired. Refresh to start a new workspace.",
            410,
        )
    return workspace, None


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


def _load_all_translations():
    return {
        locale: _load_translations(locale)
        for locale in SUPPORTED_LOCALES
    }


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


def _parse_sample_rate(form):
    raw_rate = (form.get("rate") or "48000").strip()
    try:
        sample_rate = int(raw_rate)
    except (TypeError, ValueError) as exc:
        raise ValueError("Unsupported sample rate. Choose 44100, 48000, or 96000.") from exc

    if sample_rate not in ALLOWED_SAMPLE_RATES:
        raise ValueError("Unsupported sample rate. Choose 44100, 48000, or 96000.")

    return sample_rate


def _parse_synthesis_options(form):
    sample_rate = _parse_sample_rate(form)
    parsed_layers, runtime_layers = _parse_layers_from_request(form)
    return {
        "sample_rate": sample_rate,
        "parsed_layers": parsed_layers,
        "runtime_layers": runtime_layers,
    }


def _validate_midi_upload(translations):
    if "midi_file" not in request.files:
        return None, _json_legacy_error(translations["errors.no_midi_file_uploaded"]), 400

    uploaded_file = request.files["midi_file"]
    if uploaded_file.filename == "":
        return None, _json_legacy_error(translations["errors.no_selected_file"]), 400

    extension = os.path.splitext(uploaded_file.filename)[1].lower()
    if extension not in ALLOWED_MIDI_EXTENSIONS:
        return None, _json_legacy_error(
            "Unsupported file type. Upload a .mid or .midi file.",
        ), 400

    return uploaded_file, None, None


def _validate_midi_upload_for_api(translations):
    if "midi_file" not in request.files:
        return None, _api_error_response(
            "missing_midi_file",
            translations["errors.no_midi_file_uploaded"],
            400,
        )

    uploaded_file = request.files["midi_file"]
    if uploaded_file.filename == "":
        return None, _api_error_response(
            "no_selected_file",
            translations["errors.no_selected_file"],
            400,
        )

    extension = os.path.splitext(uploaded_file.filename)[1].lower()
    if extension not in ALLOWED_MIDI_EXTENSIONS:
        return None, _api_error_response(
            "unsupported_file_type",
            "Unsupported file type. Upload a .mid or .midi file.",
            415,
        )

    return uploaded_file, None


def _parse_synthesis_options_for_api(form):
    try:
        return _parse_synthesis_options(form), None
    except ValueError as exc:
        message = str(exc)
        if "sample rate" in message:
            code = "invalid_sample_rate"
        else:
            code = "invalid_layers"
        return None, _api_error_response(code, message, 422)


def _default_workspace_config():
    return {
        "schema": WORKSPACE_CONFIG_SCHEMA,
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


def _workspace_config_error(message):
    return ValueError(message)


def _round_config_number(value, digits):
    rounded = round(float(value), digits)
    return 0.0 if rounded == 0 else rounded


def _normalise_workspace_config(payload):
    if not isinstance(payload, dict):
        raise _workspace_config_error("Workspace config must be a JSON object.")
    if payload.get("schema") != WORKSPACE_CONFIG_SCHEMA:
        raise _workspace_config_error(f"Workspace config schema must be {WORKSPACE_CONFIG_SCHEMA}.")

    sample_rate = payload.get("sample_rate")
    if isinstance(sample_rate, str) and sample_rate.strip().isdigit():
        sample_rate = int(sample_rate)
    if sample_rate not in ALLOWED_SAMPLE_RATES:
        raise _workspace_config_error("Unsupported sample rate. Choose 44100, 48000, or 96000.")

    raw_layers = payload.get("layers")
    if not isinstance(raw_layers, list):
        raise _workspace_config_error("Workspace config layers must be an array.")
    if not 1 <= len(raw_layers) <= 4:
        raise _workspace_config_error("Workspace config supports between 1 and 4 layers.")

    normalised_layers = []
    renderer_validation_layers = []
    for index, raw_layer in enumerate(raw_layers, start=1):
        if not isinstance(raw_layer, dict):
            raise _workspace_config_error(f"Layer {index} must be an object.")

        curve_enabled = raw_layer.get("curve_enabled", False)
        if not isinstance(curve_enabled, bool):
            raise _workspace_config_error(f"Layer {index} curve_enabled must be a boolean.")

        renderer_layer = {
            "type": raw_layer.get("type"),
            "duty": raw_layer.get("duty"),
            "volume": raw_layer.get("volume"),
            "frequency_curve": raw_layer.get("frequency_curve", []),
        }
        try:
            sanitised_layer = midi_to_wave.sanitise_layer(renderer_layer, index)
        except ValueError as exc:
            raise _workspace_config_error(str(exc)) from exc

        if sanitised_layer["volume"] > 2.0:
            raise _workspace_config_error(f"Layer {index} volume must be between 0.0 and 2.0.")

        normalised_layer = {
            "type": sanitised_layer["type"],
            "duty": _round_config_number(sanitised_layer["duty"], 4),
            "volume": _round_config_number(sanitised_layer["volume"], 4),
            "curve_enabled": curve_enabled,
            "frequency_curve": [
                {
                    "frequency_hz": point["frequency_hz"],
                    "gain_db": _round_config_number(point["gain_db"], 4),
                }
                for point in sanitised_layer["frequency_curve"]
            ],
        }
        normalised_layers.append(normalised_layer)
        renderer_validation_layers.append({
            "type": normalised_layer["type"],
            "duty": normalised_layer["duty"],
            "volume": normalised_layer["volume"],
            "frequency_curve": normalised_layer["frequency_curve"],
        })

    # Reuse renderer validation on the canonical layer set so future renderer
    # constraints fail before JSON reaches SQLite.
    midi_to_wave.parse_layers_json(json.dumps(renderer_validation_layers))

    return {
        "schema": WORKSPACE_CONFIG_SCHEMA,
        "sample_rate": sample_rate,
        "layers": normalised_layers,
    }


def _workspace_config_to_form_payload(config):
    renderer_layers = []
    for layer in config["layers"]:
        renderer_layers.append({
            "type": layer["type"],
            "duty": layer["duty"],
            "volume": layer["volume"],
            "frequency_curve": layer["frequency_curve"] if layer["curve_enabled"] else [],
        })
    return {
        "rate": str(config["sample_rate"]),
        "layers_json": json.dumps(renderer_layers),
    }


def _workspace_config_from_form(form):
    options = _parse_synthesis_options(form)
    if not options["parsed_layers"]:
        config = _default_workspace_config()
        config["sample_rate"] = options["sample_rate"]
        return config

    layers = options["parsed_layers"]
    normalised_layers = []
    for layer in layers:
        frequency_curve = layer.get("frequency_curve", [])
        normalised_layers.append({
            "type": layer["type"],
            "duty": _round_config_number(layer["duty"], 4),
            "volume": _round_config_number(layer["volume"], 4),
            "curve_enabled": bool(frequency_curve),
            "frequency_curve": frequency_curve,
        })
    return _normalise_workspace_config({
        "schema": WORKSPACE_CONFIG_SCHEMA,
        "sample_rate": options["sample_rate"],
        "layers": normalised_layers,
    })


def _build_original_filename(uploaded_filename):
    if not uploaded_filename:
        return "output"

    return os.path.splitext(os.path.basename(uploaded_filename))[0] or "output"


def _render_uploaded_wav(uploaded_file, form, output_wav_path, options=None):
    options = options or _parse_synthesis_options(form)

    temp_midi = tempfile.NamedTemporaryFile(delete=False, suffix=".mid")
    temp_midi.close()
    try:
        uploaded_file.save(temp_midi.name)
        if os.path.getsize(temp_midi.name) == 0:
            raise ValueError("Uploaded MIDI file is empty.")
        midi_to_wave.midi_to_audio(
            temp_midi.name,
            output_wav_path,
            options["sample_rate"],
            options["parsed_layers"],
        )
    finally:
        try:
            os.unlink(temp_midi.name)
        except FileNotFoundError:
            pass

    original_filename = _build_original_filename(uploaded_file.filename)
    return midi_to_wave.build_output_filename(original_filename, options["runtime_layers"])


def _is_valid_job_id(job_id):
    return synthesis_jobs.is_valid_job_id(job_id)


def _job_dir(job_id):
    return _job_service().job_dir(job_id)


def _job_metadata_path(job_id):
    return _job_service().metadata_path(job_id)


def _write_job_metadata(job_id, metadata):
    _job_service().write_metadata(job_id, metadata)


def _read_job_metadata(job_id):
    return _job_service().read_metadata(job_id)


def _delete_job(job_id):
    _job_service().delete_job(job_id)


def _cleanup_expired_jobs():
    _job_service().cleanup_expired_jobs()


def _job_payload(metadata):
    return _job_service().job_payload(metadata, route_base=LEGACY_SYNTHESIS_JOBS_ROUTE)


def _initial_job_metadata(job_id, uploaded_filename):
    return _job_service().initial_metadata(job_id, uploaded_filename)


def _start_synthesise_job(job_id, input_path, form_payload, uploaded_filename):
    _job_service().start_job(
        job_id,
        input_path,
        form_payload,
        uploaded_filename,
        _render_uploaded_wav,
    )


@app.errorhandler(RequestEntityTooLarge)
def handle_request_entity_too_large(_error):
    message = "Uploaded file is too large."
    if request.path.startswith("/api/"):
        return _json_api_error("upload_too_large", message, 413)
    return _json_legacy_error(message), 413


@app.route("/api/health")
def api_health():
    return _json_success({
        "status": "ok",
        "service": "octabit-web",
    })


def _save_upload_to_temp(uploaded_file):
    temp_file = tempfile.NamedTemporaryFile(delete=False, suffix=".mid")
    temp_file.close()
    try:
        uploaded_file.save(temp_file.name)
        return temp_file.name
    except Exception:
        try:
            os.unlink(temp_file.name)
        except FileNotFoundError:
            pass
        raise


def _cleanup_temp_path(path):
    try:
        os.unlink(path)
    except FileNotFoundError:
        pass


@app.route("/api/workspace", methods=["GET"])
def api_get_workspace():
    service = _workspace_service()
    workspace = service.get_or_create_workspace(request.cookies.get(WORKSPACE_COOKIE_NAME))
    return _json_workspace(service.state_payload(workspace), workspace=workspace)


@app.route("/api/workspace/uploads", methods=["POST"])
def api_create_workspace_upload():
    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    _locale, translations, _is_explicit_override = _get_locale_context()
    file, validation_error = _validate_midi_upload_for_api(translations)
    if validation_error is not None:
        return validation_error

    temp_path = _save_upload_to_temp(file)
    try:
        if os.path.getsize(temp_path) == 0:
            _cleanup_temp_path(temp_path)
            return _api_error(
                "empty_midi_file",
                translations["errors.empty_midi_file"],
                400,
            )
        service = _workspace_service()
        upload_payload = service.create_upload_from_temp(workspace, temp_path, file.filename)
    except workspaces.WorkspaceError as exc:
        _cleanup_temp_path(temp_path)
        return _api_error(exc.code, exc.message, exc.status_code)
    except Exception as exc:
        _cleanup_temp_path(temp_path)
        return _api_error("internal_error", str(exc), 500)

    return _json_workspace({"upload": upload_payload}, status_code=201)


@app.route("/api/workspace/uploads/<file_id>", methods=["DELETE"])
def api_delete_workspace_upload(file_id):
    if not _is_valid_job_id(file_id):
        return _api_error("invalid_file_id", "Invalid file id.", 400)

    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    deleted = _workspace_service().delete_upload(workspace, file_id)
    if not deleted:
        return _api_error("not_found", "File not found.", 404)
    return "", 204


@app.route("/api/workspace/queue", methods=["PATCH"])
def api_update_workspace_queue():
    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    payload = request.get_json(silent=True)
    if not isinstance(payload, dict) or not isinstance(payload.get("file_ids"), list):
        return _api_error("invalid_queue", "Request body must contain file_ids.", 422)

    file_ids = payload["file_ids"]
    if not all(_is_valid_job_id(file_id) for file_id in file_ids):
        return _api_error("invalid_file_id", "Invalid file id.", 400)

    try:
        uploads = _workspace_service().replace_queue(workspace, file_ids)
    except workspaces.WorkspaceError as exc:
        return _api_error(exc.code, exc.message, exc.status_code)

    return _json_workspace({"uploads": uploads})


@app.route("/api/workspace/config", methods=["PUT"])
def api_save_workspace_config():
    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    payload = request.get_json(silent=True)
    try:
        config = _normalise_workspace_config(payload)
    except ValueError as exc:
        return _api_error("invalid_workspace_config", str(exc), 422)

    saved_config = _workspace_service().save_config(workspace, config)
    return _json_workspace({"config": saved_config})


@app.route("/api/synthesis-jobs", methods=["POST"])
def api_create_synthesis_job():
    _locale, translations, _is_explicit_override = _get_locale_context()

    if request.is_json:
        workspace, error_response = _active_workspace_or_error()
        if error_response is not None:
            return error_response

        service = _workspace_service()
        payload = request.get_json(silent=True) or {}
        file_id = payload.get("file_id")
        if not _is_valid_job_id(file_id):
            return _api_error("invalid_file_id", "Invalid file id.", 400)
        try:
            config = _normalise_workspace_config(payload.get("config"))
        except ValueError as exc:
            return _api_error("invalid_workspace_config", str(exc), 422)

        upload_row = service.get_upload(workspace, file_id)
        if upload_row is None:
            return _api_error("not_found", "File not found.", 404)
        source_path = service.upload_path(workspace.id, file_id)
        source_name = upload_row["original_name"]
        service.save_config(workspace, config)
    else:
        file, error_response = _validate_midi_upload_for_api(translations)
        if error_response is not None:
            return error_response

        _options, error_response = _parse_synthesis_options_for_api(request.form)
        if error_response is not None:
            return error_response

        temp_path = _save_upload_to_temp(file)
        if os.path.getsize(temp_path) == 0:
            _cleanup_temp_path(temp_path)
            return _api_error(
                "empty_midi_file",
                translations["errors.empty_midi_file"],
                400,
            )

        service = _workspace_service()
        workspace = service.get_or_create_workspace(request.cookies.get(WORKSPACE_COOKIE_NAME))
        upload_row = None
        source_path = temp_path
        source_name = file.filename
        try:
            config = _workspace_config_from_form(request.form)
        except ValueError as exc:
            _cleanup_temp_path(temp_path)
            return _api_error("invalid_request", str(exc), 400)

    job_id = None
    try:
        job_id, input_path = service.prepare_job(
            workspace,
            source_path,
            source_name,
            config,
            upload_row,
        )
        if not request.is_json:
            _cleanup_temp_path(source_path)
        service.start_job(
            workspace,
            job_id,
            input_path,
            _workspace_config_to_form_payload(config),
            source_name,
            _render_uploaded_wav,
        )
    except synthesis_jobs.RenderQueueFull as exc:
        if job_id is not None:
            service.delete_job(workspace, job_id)
        if not request.is_json:
            _cleanup_temp_path(source_path)
        return _api_error("render_queue_full", str(exc), 429)
    except workspaces.WorkspaceError as exc:
        if not request.is_json:
            _cleanup_temp_path(source_path)
        return _api_error(exc.code, exc.message, exc.status_code)
    except Exception as exc:
        if not request.is_json:
            _cleanup_temp_path(source_path)
        return _api_error("internal_error", str(exc), 500)

    job_row, _expired = service.get_job(workspace, job_id)
    return _json_workspace(service.job_payload(job_row), status_code=202, workspace=workspace)


@app.route("/api/synthesis-jobs/<job_id>", methods=["GET"])
def api_get_synthesis_job(job_id):
    if not _is_valid_job_id(job_id):
        return _api_error("invalid_job_id", "Invalid job id.", 400)

    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    service = _workspace_service()
    job_row, expired = service.get_job(workspace, job_id)
    if expired:
        return jsonify({"job_id": job_id, "status": "expired"}), 410
    if job_row is None:
        return _api_error("not_found", "Job not found.", 404)

    return jsonify(service.job_payload(job_row))


@app.route("/api/synthesis-jobs/<job_id>", methods=["DELETE"])
def api_delete_synthesis_job(job_id):
    if not _is_valid_job_id(job_id):
        return _api_error("invalid_job_id", "Invalid job id.", 400)

    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    deleted = _workspace_service().delete_job(workspace, job_id)
    if not deleted:
        return _api_error("not_found", "Job not found.", 404)
    return "", 204


@app.route("/api/synthesis-jobs/<job_id>/download", methods=["GET"])
def api_download_synthesis_job(job_id):
    if not _is_valid_job_id(job_id):
        return _api_error("invalid_job_id", "Invalid job id.", 400)

    workspace, error_response = _active_workspace_or_error()
    if error_response is not None:
        return error_response

    service = _workspace_service()
    job_row, expired = service.get_job(workspace, job_id)
    if expired:
        return jsonify({"job_id": job_id, "status": "expired"}), 410
    if job_row is None:
        return _api_error("not_found", "Job not found.", 404)

    payload = service.job_payload(job_row)
    if job_row["status"] == "failed":
        return jsonify(payload), 400
    if job_row["status"] != "ready":
        return jsonify(payload), 409

    output_path = service.job_output_path(workspace.id, job_id)
    if not os.path.exists(output_path):
        service.delete_job(workspace, job_id)
        return jsonify({"job_id": job_id, "status": "expired"}), 410

    return send_file(
        output_path,
        as_attachment=True,
        download_name=job_row["download_name"] or "output.wav",
    )


@app.route("/")
def index():
    locale, translations, is_explicit_override = _get_locale_context()
    response = make_response(render_template(
        "index.html",
        default_locale=DEFAULT_LOCALE,
        locale=locale,
        locale_cookie_name=LOCALE_COOKIE_NAME,
        supported_locales=SUPPORTED_LOCALES,
        translations_by_locale=_load_all_translations(),
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

    file, error_response, status_code = _validate_midi_upload(translations)
    if error_response is not None:
        return error_response, status_code
    try:
        options = _parse_synthesis_options(request.form)
    except ValueError as exc:
        return _json_legacy_error(str(exc)), 400

    temp_paths = []
    try:
        temp_wav = tempfile.NamedTemporaryFile(delete=False, suffix=".wav")
        temp_wav.close()
        temp_paths.append(temp_wav.name)
        download_name = _render_uploaded_wav(file, request.form, temp_wav.name, options)

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
            download_name=download_name,
        )
    except ValueError as exc:
        for temp_path in temp_paths:
            try:
                os.unlink(temp_path)
            except FileNotFoundError:
                pass
        return _json_legacy_error(str(exc)), 400
    except Exception as exc:
        for temp_path in temp_paths:
            try:
                os.unlink(temp_path)
            except FileNotFoundError:
                pass
        return _json_legacy_error(str(exc)), 500


@app.route("/synthesise/jobs", methods=["POST"])
def create_synthesise_job():
    _locale, translations, _is_explicit_override = _get_locale_context()
    file, error_response, status_code = _validate_midi_upload(translations)
    if error_response is not None:
        return error_response, status_code
    try:
        _parse_synthesis_options(request.form)
    except ValueError as exc:
        return _json_legacy_error(str(exc)), 400

    _cleanup_expired_jobs()
    job_service = _job_service()
    job_id, input_path = job_service.prepare_job()

    try:
        file.save(input_path)
        if os.path.getsize(input_path) == 0:
            _delete_job(job_id)
            return _json_legacy_error(translations["errors.empty_midi_file"]), 400
        _write_job_metadata(job_id, _initial_job_metadata(job_id, file.filename))
        _start_synthesise_job(job_id, input_path, request.form.to_dict(flat=True), file.filename)
    except synthesis_jobs.RenderQueueFull as exc:
        _delete_job(job_id)
        return _json_legacy_error(str(exc)), 429
    except ValueError as exc:
        _delete_job(job_id)
        return _json_legacy_error(str(exc)), 400
    except Exception as exc:
        _delete_job(job_id)
        return _json_legacy_error(str(exc)), 500

    metadata = _read_job_metadata(job_id)
    return _json_accepted(_job_payload(metadata))


@app.route("/synthesise/jobs/<job_id>", methods=["GET"])
def get_synthesise_job(job_id):
    metadata = _read_job_metadata(job_id)
    if metadata is None:
        return jsonify({"job_id": job_id, "status": "expired"}), 410

    payload = _job_payload(metadata)
    if payload["status"] == "expired":
        return jsonify(payload), 410

    return jsonify(payload)


@app.route("/synthesise/jobs/<job_id>", methods=["DELETE"])
def delete_synthesise_job(job_id):
    _delete_job(job_id)
    return "", 204


@app.route("/synthesise/jobs/<job_id>/download", methods=["GET"])
def download_synthesise_job(job_id):
    metadata = _read_job_metadata(job_id)
    if metadata is None or metadata.get("status") == "expired":
        return jsonify({"job_id": job_id, "status": "expired"}), 410
    if metadata.get("status") == "failed":
        return jsonify(_job_payload(metadata)), 400
    if metadata.get("status") != "ready":
        return jsonify(_job_payload(metadata)), 409

    output_path = os.path.join(_job_dir(job_id), "output.wav")
    if not os.path.exists(output_path):
        _delete_job(job_id)
        return jsonify({"job_id": job_id, "status": "expired"}), 410

    return send_file(
        output_path,
        as_attachment=True,
        download_name=metadata.get("download_name", "output.wav"),
    )


if __name__ == "__main__":
    server_host = os.environ.get("HOST", "127.0.0.1")
    server_port = _get_server_port()
    should_open_browser = os.environ.get("WEB_FLASK_OPEN_BROWSER", "1") != "0"

    if should_open_browser and server_host in ("127.0.0.1", "localhost"):
        Timer(1, lambda: open_browser(server_port)).start()

    print(f"DEBUG: SERVER RUNNING ON {server_host}:{server_port}")
    app.run(host=server_host, port=server_port, debug=False)
