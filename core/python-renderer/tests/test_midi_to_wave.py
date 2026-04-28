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

import json
import math
import sys
import tempfile
import unittest
from pathlib import Path

import numpy as np
import pretty_midi
from scipy.io import wavfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import midi_to_wave


class ParseLayersJsonTests(unittest.TestCase):
    def test_accepts_layer_without_frequency_curve(self):
        layers = midi_to_wave.parse_layers_json(json.dumps([{
            "type": "pulse",
            "duty": 0.5,
            "volume": 1.0,
        }]))

        self.assertEqual(
            [{
                "type": "pulse",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [],
            }],
            layers,
        )

    def test_accepts_empty_frequency_curve(self):
        layers = midi_to_wave.parse_layers_json(json.dumps([{
            "type": "sine",
            "duty": 0.5,
            "volume": 0.8,
            "frequency_curve": [],
        }]))

        self.assertEqual([], layers[0]["frequency_curve"])

    def test_sorts_frequency_curve_points(self):
        layers = midi_to_wave.parse_layers_json(json.dumps([{
            "type": "triangle",
            "duty": 0.5,
            "volume": 0.5,
            "frequency_curve": [
                {"frequency_hz": 880.0, "gain_db": 6.0},
                {"frequency_hz": 220.0, "gain_db": -6.0},
            ],
        }]))

        self.assertEqual(
            [220.0, 880.0],
            [point["frequency_hz"] for point in layers[0]["frequency_curve"]],
        )

    def test_accepts_boundary_frequency_with_small_rounding_drift(self):
        layers = midi_to_wave.parse_layers_json(json.dumps([{
            "type": "sine",
            "duty": 0.5,
            "volume": 1.0,
            "frequency_curve": [
                {"frequency_hz": 8.1757989156, "gain_db": 0.0},
                {"frequency_hz": 12543.8539514, "gain_db": 0.0},
            ],
        }]))

        self.assertEqual(
            midi_to_wave.MIN_CURVE_FREQUENCY_HZ,
            layers[0]["frequency_curve"][0]["frequency_hz"],
        )
        self.assertAlmostEqual(
            12543.8539514,
            layers[0]["frequency_curve"][1]["frequency_hz"],
            places=10,
        )
        self.assertLessEqual(
            layers[0]["frequency_curve"][1]["frequency_hz"],
            midi_to_wave.MAX_CURVE_FREQUENCY_HZ,
        )

    def test_rejects_duplicate_curve_frequencies(self):
        with self.assertRaisesRegex(ValueError, "strictly increasing"):
            midi_to_wave.parse_layers_json(json.dumps([{
                "type": "sine",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [
                    {"frequency_hz": 440.0, "gain_db": 0.0},
                    {"frequency_hz": 440.0, "gain_db": -3.0},
                ],
            }]))

    def test_rejects_out_of_range_curve_frequency(self):
        with self.assertRaisesRegex(ValueError, "frequency_hz"):
            midi_to_wave.parse_layers_json(json.dumps([{
                "type": "sine",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [
                    {"frequency_hz": 1.0, "gain_db": 0.0},
                ],
            }]))

    def test_rejects_out_of_range_curve_gain(self):
        with self.assertRaisesRegex(ValueError, "gain_db"):
            midi_to_wave.parse_layers_json(json.dumps([{
                "type": "sine",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [
                    {"frequency_hz": 440.0, "gain_db": 18.0},
                ],
            }]))

    def test_rejects_non_numeric_curve_field(self):
        with self.assertRaisesRegex(ValueError, "gain_db"):
            midi_to_wave.parse_layers_json(json.dumps([{
                "type": "sine",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [
                    {"frequency_hz": 440.0, "gain_db": "loud"},
                ],
            }]))


class FrequencyCurveEvaluationTests(unittest.TestCase):
    def test_returns_exact_gain_at_curve_point(self):
        gain_db = midi_to_wave.evaluate_frequency_curve_gain_db(
            [{"frequency_hz": 440.0, "gain_db": -9.0}],
            440.0,
        )

        self.assertEqual(-9.0, gain_db)

    def test_interpolates_midpoint_on_log_frequency_axis(self):
        gain_db = midi_to_wave.evaluate_frequency_curve_gain_db(
            [
                {"frequency_hz": 100.0, "gain_db": 0.0},
                {"frequency_hz": 400.0, "gain_db": -12.0},
            ],
            200.0,
        )

        self.assertAlmostEqual(-6.0, gain_db, places=6)

    def test_clamps_below_first_point(self):
        gain_db = midi_to_wave.evaluate_frequency_curve_gain_db(
            [
                {"frequency_hz": 220.0, "gain_db": -8.0},
                {"frequency_hz": 880.0, "gain_db": 3.0},
            ],
            110.0,
        )

        self.assertEqual(-8.0, gain_db)

    def test_clamps_above_last_point(self):
        gain_db = midi_to_wave.evaluate_frequency_curve_gain_db(
            [
                {"frequency_hz": 220.0, "gain_db": -8.0},
                {"frequency_hz": 880.0, "gain_db": 3.0},
            ],
            1760.0,
        )

        self.assertEqual(3.0, gain_db)

    def test_single_point_curve_is_constant(self):
        gain_db = midi_to_wave.evaluate_frequency_curve_gain_db(
            [{"frequency_hz": 440.0, "gain_db": -12.0}],
            1760.0,
        )

        self.assertEqual(-12.0, gain_db)


class OutputNamingTests(unittest.TestCase):
    def test_build_output_suffix_uses_wave_type_without_curve(self):
        suffix = midi_to_wave.build_output_suffix([{
            "type": "pulse",
            "duty": 0.5,
            "volume": 1.0,
            "frequency_curve": [],
        }])

        self.assertEqual("pulse", suffix)

    def test_build_output_suffix_uses_mix_for_multiple_layers_without_curve(self):
        suffix = midi_to_wave.build_output_suffix([
            {
                "type": "pulse",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [],
            },
            {
                "type": "triangle",
                "duty": 0.5,
                "volume": 0.4,
                "frequency_curve": [],
            },
        ])

        self.assertEqual("mix", suffix)

    def test_build_output_suffix_appends_curve_hash(self):
        suffix = midi_to_wave.build_output_suffix([{
            "type": "pulse",
            "duty": 0.5,
            "volume": 1.0,
            "frequency_curve": [
                {"frequency_hz": midi_to_wave.MIN_CURVE_FREQUENCY_HZ, "gain_db": 0.0},
                {"frequency_hz": midi_to_wave.MAX_CURVE_FREQUENCY_HZ, "gain_db": 0.0},
            ],
        }])

        self.assertEqual("pulse_dee027b8", suffix)
        self.assertEqual(
            "lead_pulse_dee027b8.wav",
            midi_to_wave.build_output_filename("lead", [{
                "type": "pulse",
                "duty": 0.5,
                "volume": 1.0,
                "frequency_curve": [
                    {"frequency_hz": midi_to_wave.MIN_CURVE_FREQUENCY_HZ, "gain_db": 0.0},
                    {"frequency_hz": midi_to_wave.MAX_CURVE_FREQUENCY_HZ, "gain_db": 0.0},
                ],
            }]),
        )


class MidiToAudioTests(unittest.TestCase):
    def test_very_short_note_still_renders_audio_sample(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            midi_path = Path(temp_dir) / "short.mid"
            wav_path = Path(temp_dir) / "short.wav"
            self._write_test_midi(
                midi_path,
                [
                    (69, 0.0, 0.00001),
                ],
            )

            midi_to_wave.midi_to_audio(
                str(midi_path),
                str(wav_path),
                sample_rate=16000,
                layers=[{
                    "type": "pulse",
                    "duty": 0.5,
                    "volume": 1.0,
                    "frequency_curve": [],
                }],
            )

            _, data = wavfile.read(wav_path)
            self.assertGreater(len(data), 0)
            self.assertNotEqual(0, int(np.max(np.abs(data))))

    def test_frequency_curve_changes_note_level_gain_by_pitch(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            midi_path = Path(temp_dir) / "notes.mid"
            wav_path = Path(temp_dir) / "notes.wav"
            self._write_test_midi(
                midi_path,
                [
                    (60, 0.0, 0.4),
                    (84, 0.5, 0.9),
                ],
            )

            low_frequency = pretty_midi.note_number_to_hz(60)
            high_frequency = pretty_midi.note_number_to_hz(84)
            midi_to_wave.midi_to_audio(
                str(midi_path),
                str(wav_path),
                sample_rate=16000,
                layers=[{
                    "type": "sine",
                    "duty": 0.5,
                    "volume": 1.0,
                    "frequency_curve": [
                        {"frequency_hz": low_frequency, "gain_db": -24.0},
                        {"frequency_hz": high_frequency, "gain_db": 0.0},
                    ],
                }],
            )

            _, data = wavfile.read(wav_path)
            low_rms = self._segment_rms(data, 0.05, 0.35, 16000)
            high_rms = self._segment_rms(data, 0.55, 0.85, 16000)

            self.assertGreater(high_rms / low_rms, 8.0)

    def test_missing_curve_matches_empty_curve_output(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            midi_path = Path(temp_dir) / "compat.mid"
            wav_without_curve = Path(temp_dir) / "without_curve.wav"
            wav_with_empty_curve = Path(temp_dir) / "with_empty_curve.wav"
            self._write_test_midi(midi_path, [(69, 0.0, 0.5)])

            base_layer = {
                "type": "pulse",
                "duty": 0.25,
                "volume": 0.7,
            }
            midi_to_wave.midi_to_audio(
                str(midi_path),
                str(wav_without_curve),
                sample_rate=16000,
                layers=[base_layer],
            )
            midi_to_wave.midi_to_audio(
                str(midi_path),
                str(wav_with_empty_curve),
                sample_rate=16000,
                layers=[dict(base_layer, frequency_curve=[])],
            )

            _, data_without_curve = wavfile.read(wav_without_curve)
            _, data_with_empty_curve = wavfile.read(wav_with_empty_curve)
            np.testing.assert_array_equal(data_without_curve, data_with_empty_curve)

    def _write_test_midi(self, midi_path, note_specs):
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
        midi.write(str(midi_path))

    def _segment_rms(self, samples, start_time, end_time, sample_rate):
        start_index = int(start_time * sample_rate)
        end_index = int(end_time * sample_rate)
        segment = samples[start_index:end_index].astype(np.float64)
        return math.sqrt(np.mean(np.square(segment)))


if __name__ == "__main__":
    unittest.main()
