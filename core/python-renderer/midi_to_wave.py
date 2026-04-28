# Copyright (C) 2026 Jeremy Yang
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.

import argparse
import hashlib
import json
import math

import numpy as np
import pretty_midi
from scipy.io import wavfile

VALID_WAVE_TYPES = {"pulse", "sine", "sawtooth", "triangle"}
MIN_CURVE_FREQUENCY_HZ = float(pretty_midi.note_number_to_hz(0))
MAX_CURVE_FREQUENCY_HZ = float(pretty_midi.note_number_to_hz(127))
MIN_CURVE_GAIN_DB = -36.0
MAX_CURVE_GAIN_DB = 12.0
MAX_CURVE_POINTS = 8
CURVE_FREQUENCY_TOLERANCE_HZ = 1e-6


def _parse_finite_number(raw_value, field_label):
    try:
        value = float(raw_value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{field_label} must be a number.") from exc

    if not math.isfinite(value):
        raise ValueError(f"{field_label} must be finite.")

    return value


def _parse_layer_number(layer, layer_index, field_name, default_value):
    return _parse_finite_number(
        layer.get(field_name, default_value),
        f"Layer {layer_index} {field_name}",
    )


def _default_layer():
    return {
        "type": "pulse",
        "duty": 0.5,
        "volume": 1.0,
        "frequency_curve": [],
    }


def _sanitise_frequency_curve(curve, layer_index):
    if curve is None:
        return []

    if not isinstance(curve, list):
        raise ValueError(f"Layer {layer_index} frequency_curve must be an array.")

    if not curve:
        return []

    if len(curve) > MAX_CURVE_POINTS:
        raise ValueError(
            f"Layer {layer_index} frequency_curve supports at most "
            f"{MAX_CURVE_POINTS} points."
        )

    points = []
    for point_index, point in enumerate(curve, start=1):
        if not isinstance(point, dict):
            raise ValueError(
                f"Layer {layer_index} frequency_curve point {point_index} must be an object."
            )

        frequency_hz = _parse_finite_number(
            point.get("frequency_hz"),
            f"Layer {layer_index} frequency_curve point {point_index} frequency_hz",
        )
        if (
            frequency_hz < MIN_CURVE_FREQUENCY_HZ
            and math.isclose(
                frequency_hz,
                MIN_CURVE_FREQUENCY_HZ,
                rel_tol=0.0,
                abs_tol=CURVE_FREQUENCY_TOLERANCE_HZ,
            )
        ):
            frequency_hz = MIN_CURVE_FREQUENCY_HZ
        elif (
            frequency_hz > MAX_CURVE_FREQUENCY_HZ
            and math.isclose(
                frequency_hz,
                MAX_CURVE_FREQUENCY_HZ,
                rel_tol=0.0,
                abs_tol=CURVE_FREQUENCY_TOLERANCE_HZ,
            )
        ):
            frequency_hz = MAX_CURVE_FREQUENCY_HZ

        if not MIN_CURVE_FREQUENCY_HZ <= frequency_hz <= MAX_CURVE_FREQUENCY_HZ:
            raise ValueError(
                f"Layer {layer_index} frequency_curve point {point_index} frequency_hz "
                f"must be between {MIN_CURVE_FREQUENCY_HZ:.6f} and "
                f"{MAX_CURVE_FREQUENCY_HZ:.6f}."
            )

        gain_db = _parse_finite_number(
            point.get("gain_db"),
            f"Layer {layer_index} frequency_curve point {point_index} gain_db",
        )
        if not MIN_CURVE_GAIN_DB <= gain_db <= MAX_CURVE_GAIN_DB:
            raise ValueError(
                f"Layer {layer_index} frequency_curve point {point_index} gain_db "
                f"must be between {MIN_CURVE_GAIN_DB:.1f} and {MAX_CURVE_GAIN_DB:.1f}."
            )

        points.append({
            "frequency_hz": frequency_hz,
            "gain_db": gain_db,
        })

    points.sort(key=lambda point: point["frequency_hz"])
    for point_index in range(1, len(points)):
        previous_frequency = points[point_index - 1]["frequency_hz"]
        current_frequency = points[point_index]["frequency_hz"]
        if current_frequency <= previous_frequency:
            raise ValueError(
                f"Layer {layer_index} frequency_curve frequencies must be strictly increasing."
            )

    return points


def sanitise_layer(layer, layer_index):
    if not isinstance(layer, dict):
        raise ValueError(f"Layer {layer_index} must be an object.")

    wave_type = layer.get("type", "pulse")
    if not isinstance(wave_type, str):
        raise ValueError(f"Layer {layer_index} waveform type must be a string.")
    if wave_type not in VALID_WAVE_TYPES:
        raise ValueError(
            f"Layer {layer_index} has unsupported waveform '{wave_type}'. "
            f"Expected one of: {', '.join(sorted(VALID_WAVE_TYPES))}."
        )

    duty = _parse_layer_number(layer, layer_index, "duty", 0.5)
    if not 0.01 <= duty <= 0.99:
        raise ValueError(f"Layer {layer_index} duty must be between 0.01 and 0.99.")

    volume = _parse_layer_number(layer, layer_index, "volume", 1.0)
    if volume < 0:
        raise ValueError(f"Layer {layer_index} volume must be 0 or greater.")

    frequency_curve = _sanitise_frequency_curve(
        layer.get("frequency_curve"),
        layer_index,
    )

    return {
        "type": wave_type,
        "duty": duty,
        "volume": volume,
        "frequency_curve": frequency_curve,
    }


def db_to_linear_gain(gain_db):
    return 10.0 ** (gain_db / 20.0)


def build_curve_payload_hash(layers):
    canonical_payload = json.dumps(layers, sort_keys=True, separators=(",", ":"))
    return hashlib.sha1(canonical_payload.encode("utf-8")).hexdigest()[:8]


def build_output_suffix(layers):
    runtime_layers = normalise_runtime_layers(layers)
    suffix = "mix" if len(runtime_layers) > 1 else runtime_layers[0]["type"]
    if any(layer["frequency_curve"] for layer in runtime_layers):
        suffix = f"{suffix}_{build_curve_payload_hash(runtime_layers)}"

    return suffix


def build_output_filename(original_filename, layers):
    return f"{original_filename}_{build_output_suffix(layers)}.wav"


def evaluate_frequency_curve_gain_db(curve_points, frequency_hz):
    if not curve_points:
        return 0.0

    if frequency_hz <= curve_points[0]["frequency_hz"]:
        return curve_points[0]["gain_db"]
    if frequency_hz >= curve_points[-1]["frequency_hz"]:
        return curve_points[-1]["gain_db"]
    if len(curve_points) == 1:
        return curve_points[0]["gain_db"]

    log_frequency = math.log(frequency_hz)
    for left_point, right_point in zip(curve_points, curve_points[1:]):
        left_frequency = left_point["frequency_hz"]
        right_frequency = right_point["frequency_hz"]
        if left_frequency <= frequency_hz <= right_frequency:
            left_log_frequency = math.log(left_frequency)
            right_log_frequency = math.log(right_frequency)
            interpolation = (
                (log_frequency - left_log_frequency)
                / (right_log_frequency - left_log_frequency)
            )
            return left_point["gain_db"] + (
                interpolation * (right_point["gain_db"] - left_point["gain_db"])
            )

    return curve_points[-1]["gain_db"]


def generate_waveform(freq, duration, sample_rate, wave_type='pulse', duty_cycle=0.5):
    """Generates various audio waveforms."""
    t = np.linspace(0, duration, int(sample_rate * duration), endpoint=False)
    
    if wave_type == 'sine':
        return np.sin(2 * np.pi * freq * t)
    
    elif wave_type == 'sawtooth':
        # Linear ramp from -1 to 1
        return 2.0 * (t * freq % 1.0) - 1.0
    
    elif wave_type == 'triangle':
        # Absolute value of a sawtooth
        return 2.0 * np.abs(2.0 * (t * freq % 1.0) - 1.0) - 1.0
    
    elif wave_type == 'pulse':
        # Default square wave if duty is 0.5
        return np.where((t * freq) % 1.0 < duty_cycle, 1.0, -1.0)
    
    return np.zeros_like(t)


def apply_envelope(waveform, sample_rate, attack=0.005, release=0.005):
    """Applies a simple linear attack/release envelope to prevent clicks."""
    if len(waveform) == 0: return waveform
    attack_samples = min(int(attack * sample_rate), len(waveform) // 2)
    release_samples = min(int(release * sample_rate), len(waveform) - attack_samples)
    envelope = np.ones(len(waveform))
    if attack_samples > 0: envelope[:attack_samples] = np.linspace(0, 1, attack_samples)
    if release_samples > 0: envelope[-release_samples:] = np.linspace(1, 0, release_samples)
    return waveform * envelope


def midi_to_audio(midi_path, output_path, sample_rate=48000, layers=None):
    layers = normalise_runtime_layers(layers)

    midi_data = pretty_midi.PrettyMIDI(midi_path)
    total_time = midi_data.get_end_time()
    total_samples = int(np.ceil(total_time * sample_rate))
    audio_buffer = np.zeros(total_samples, dtype=np.float64)

    if total_samples == 0:
        wavfile.write(output_path, sample_rate, np.zeros(0, dtype=np.int16))
        return

    for instrument in midi_data.instruments:
        if instrument.is_drum: continue
        for note in instrument.notes:
            start_sample = int(note.start * sample_rate)
            end_sample = int(math.ceil(note.end * sample_rate))
            note_sample_length = end_sample - start_sample
            if note_sample_length <= 0:
                continue
            duration = note_sample_length / sample_rate
            
            freq = pretty_midi.note_number_to_hz(note.pitch)
            note_volume = note.velocity / 127.0
            
            mixed_note_waveform = np.zeros(
                int(sample_rate * duration),
                dtype=np.float64,
            )
            
            for layer in layers:
                curve_gain_db = evaluate_frequency_curve_gain_db(
                    layer["frequency_curve"],
                    freq,
                )
                effective_volume = layer["volume"] * db_to_linear_gain(curve_gain_db)
                if effective_volume <= 0:
                    continue

                layer_wave = generate_waveform(
                    freq,
                    duration,
                    sample_rate,
                    layer["type"],
                    layer["duty"],
                )
                mixed_note_waveform += layer_wave * effective_volume
                
            mixed_note_waveform = apply_envelope(mixed_note_waveform, sample_rate)
            end_sample = start_sample + len(mixed_note_waveform)
            
            if end_sample > len(audio_buffer):
                actual_len = len(audio_buffer) - start_sample
                audio_buffer[start_sample:] += mixed_note_waveform[:actual_len] * note_volume
            else:
                audio_buffer[start_sample:end_sample] += mixed_note_waveform * note_volume

    max_val = np.max(np.abs(audio_buffer))
    if max_val > 0:
        audio_buffer = (audio_buffer / max_val) * 0.89

    wavfile.write(output_path, sample_rate, (audio_buffer * 32767).astype(np.int16))

def normalise_runtime_layers(layers):
    if not layers:
        return [_default_layer()]

    audible_layers = []
    for layer_index, layer in enumerate(layers, start=1):
        sanitised_layer = sanitise_layer(layer, layer_index)
        if sanitised_layer["volume"] <= 0:
            continue

        audible_layers.append(sanitised_layer)

    return audible_layers or [_default_layer()]

def parse_layers_json(layers_json):
    try:
        parsed = json.loads(layers_json)
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid layer JSON: {exc.msg}") from exc

    if not isinstance(parsed, list):
        raise ValueError("Layer JSON must be an array of layer objects.")

    return [
        sanitise_layer(layer, index)
        for index, layer in enumerate(parsed, start=1)
    ]

def build_layers_from_args(args):
    if args.layers_json:
        return parse_layers_json(args.layers_json)

    return [sanitise_layer({
        "type": args.type,
        "duty": args.duty,
        "volume": 1.0,
        "frequency_curve": [],
    }, 1)]

def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("input", help="Input MIDI file")
    parser.add_argument("output", help="Output WAV file")
    parser.add_argument("--type", default="pulse", choices=sorted(VALID_WAVE_TYPES))
    parser.add_argument("--duty", type=float, default=0.5)
    parser.add_argument("--rate", type=int, default=48000)
    parser.add_argument(
        "--layers-json",
        help=(
            "JSON array of waveform layers, each containing type, duty, volume, "
            "and optional frequency_curve points."
        )
    )
    args = parser.parse_args(argv)

    layers = build_layers_from_args(args)
    midi_to_audio(args.input, args.output, args.rate, layers)
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
