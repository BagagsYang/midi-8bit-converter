import argparse
import json

import numpy as np
import pretty_midi
from scipy.io import wavfile

VALID_WAVE_TYPES = {"pulse", "sine", "sawtooth", "triangle"}

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
            duration = note.end - note.start
            if duration <= 0: continue
            
            freq = pretty_midi.note_number_to_hz(note.pitch)
            note_volume = note.velocity / 127.0
            
            t = np.linspace(0, duration, int(sample_rate * duration), endpoint=False)
            mixed_note_waveform = np.zeros_like(t)
            
            for layer in layers:
                w_type = layer.get('type', 'pulse')
                duty = layer.get('duty', 0.5)
                vol = layer.get('volume', 1.0)
                
                if w_type == 'sine':
                    layer_wave = np.sin(2 * np.pi * freq * t)
                elif w_type == 'sawtooth':
                    layer_wave = 2.0 * (t * freq % 1.0) - 1.0
                elif w_type == 'triangle':
                    layer_wave = 2.0 * np.abs(2.0 * (t * freq % 1.0) - 1.0) - 1.0
                elif w_type == 'pulse':
                    layer_wave = np.where((t * freq) % 1.0 < duty, 1.0, -1.0)
                else:
                    layer_wave = np.zeros_like(t)
                    
                mixed_note_waveform += layer_wave * vol
                
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
        return [{'type': 'pulse', 'duty': 0.5, 'volume': 1.0}]

    audible_layers = []
    for layer in layers:
        volume = float(layer.get('volume', 1.0))
        if volume <= 0:
            continue

        wave_type = layer.get('type', 'pulse')
        duty = float(layer.get('duty', 0.5))
        audible_layers.append({
            'type': wave_type,
            'duty': duty,
            'volume': volume,
        })

    return audible_layers or [{'type': 'pulse', 'duty': 0.5, 'volume': 1.0}]

def parse_layers_json(layers_json):
    try:
        parsed = json.loads(layers_json)
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid layer JSON: {exc.msg}") from exc

    if not isinstance(parsed, list):
        raise ValueError("Layer JSON must be an array of layer objects.")

    layers = []
    for index, layer in enumerate(parsed, start=1):
        if not isinstance(layer, dict):
            raise ValueError(f"Layer {index} must be an object.")

        wave_type = layer.get('type', 'pulse')
        if wave_type not in VALID_WAVE_TYPES:
            raise ValueError(
                f"Layer {index} has unsupported waveform '{wave_type}'. "
                f"Expected one of: {', '.join(sorted(VALID_WAVE_TYPES))}."
            )

        duty = float(layer.get('duty', 0.5))
        if not 0.01 <= duty <= 0.99:
            raise ValueError(f"Layer {index} duty must be between 0.01 and 0.99.")

        volume = float(layer.get('volume', 1.0))
        if volume < 0:
            raise ValueError(f"Layer {index} volume must be 0 or greater.")

        layers.append({
            'type': wave_type,
            'duty': duty,
            'volume': volume,
        })

    return layers

def build_layers_from_args(args):
    if args.layers_json:
        return parse_layers_json(args.layers_json)

    return [{
        'type': args.type,
        'duty': args.duty,
        'volume': 1.0,
    }]

def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("input", help="Input MIDI file")
    parser.add_argument("output", help="Output WAV file")
    parser.add_argument("--type", default="pulse", choices=sorted(VALID_WAVE_TYPES))
    parser.add_argument("--duty", type=float, default=0.5)
    parser.add_argument("--rate", type=int, default=48000)
    parser.add_argument(
        "--layers-json",
        help="JSON array of waveform layers, each containing type, duty, and volume."
    )
    args = parser.parse_args(argv)

    layers = build_layers_from_args(args)
    midi_to_audio(args.input, args.output, args.rate, layers)
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
