package renderer

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type rendererExpectations struct {
	Cases              []rendererCase    `json:"cases"`
	InvalidLayerErrors map[string]string `json:"invalid_layer_errors"`
	Limits             struct {
		MaxRenderSeconds int     `json:"max_render_seconds"`
		MaxMIDINotes     int     `json:"max_midi_notes"`
		MaxRenderLayers  int     `json:"max_render_layers"`
		NormalisedPeak   float64 `json:"normalised_peak"`
	} `json:"limits"`
}

type rendererCase struct {
	Name             string             `json:"name"`
	MIDI             string             `json:"midi"`
	Layers           []Layer            `json:"layers"`
	NoteSpecs        []Note             `json:"note_specs"`
	OutputName       string             `json:"output_name"`
	CurveHash        string             `json:"curve_hash"`
	CurveGainSamples map[string]float64 `json:"curve_gain_samples"`
	SampleRate       int                `json:"sample_rate"`
	WAV              struct {
		SHA256 string `json:"sha256"`
		Size   int    `json:"size"`
	} `json:"wav"`
}

func TestRendererConstantsMatchPythonBaseline(t *testing.T) {
	expectations := loadRendererExpectations(t)

	if expectations.Limits.MaxRenderSeconds != MaxRenderSeconds {
		t.Fatalf("MaxRenderSeconds = %d", MaxRenderSeconds)
	}
	if expectations.Limits.MaxMIDINotes != MaxMIDINotes {
		t.Fatalf("MaxMIDINotes = %d", MaxMIDINotes)
	}
	if expectations.Limits.MaxRenderLayers != MaxRenderLayers {
		t.Fatalf("MaxRenderLayers = %d", MaxRenderLayers)
	}
	if expectations.Limits.NormalisedPeak != NormalisedPeak {
		t.Fatalf("NormalisedPeak = %f", NormalisedPeak)
	}
}

func TestOutputNamesAndCurveHashesMatchPythonBaseline(t *testing.T) {
	expectations := loadRendererExpectations(t)

	for _, testCase := range expectations.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			filename, err := BuildOutputFilename(trimMIDIExtension(testCase.MIDIName()), testCase.Layers)
			if err != nil {
				t.Fatal(err)
			}
			if filename != testCase.OutputName {
				t.Fatalf("BuildOutputFilename() = %q, want %q", filename, testCase.OutputName)
			}
			if testCase.CurveHash != "" {
				hashValue, err := BuildCurvePayloadHash(testCase.Layers)
				if err != nil {
					t.Fatal(err)
				}
				if hashValue != testCase.CurveHash {
					t.Fatalf("BuildCurvePayloadHash() = %q, want %q", hashValue, testCase.CurveHash)
				}
			}
		})
	}
}

func TestCurveGainEvaluationMatchesPythonBaseline(t *testing.T) {
	expectations := loadRendererExpectations(t)

	for _, testCase := range expectations.Cases {
		if len(testCase.CurveGainSamples) == 0 {
			continue
		}
		curve := testCase.Layers[0].FrequencyCurve
		for frequency, expectedGain := range testCase.CurveGainSamples {
			t.Run(testCase.Name+"/"+frequency, func(t *testing.T) {
				frequencyHz, err := parseFrequencyFixtureKey(frequency)
				if err != nil {
					t.Fatal(err)
				}
				gain := EvaluateFrequencyCurveGainDB(curve, frequencyHz)
				if math.Abs(gain-expectedGain) > 1e-12 {
					t.Fatalf("gain = %.16f, want %.16f", gain, expectedGain)
				}
			})
		}
	}
}

func TestRenderNotesWAVMatchesPythonBaselineBytes(t *testing.T) {
	expectations := loadRendererExpectations(t)

	for _, testCase := range expectations.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			wavBytes, err := RenderNotesWAV(testCase.NoteSpecs, testCase.SampleRate, testCase.Layers)
			if err != nil {
				t.Fatal(err)
			}
			if len(wavBytes) != testCase.WAV.Size {
				t.Fatalf("WAV size = %d, want %d", len(wavBytes), testCase.WAV.Size)
			}
			hash := sha256.Sum256(wavBytes)
			actualSHA256 := fmt.Sprintf("%x", hash[:])
			if actualSHA256 == testCase.WAV.SHA256 {
				return
			}

			expectedWAV := readBaselineWAV(t, testCase.Name)
			maxDiff := maxPCM16AbsDiff(t, wavBytes, expectedWAV)
			if maxDiff > 1 {
				t.Fatalf("WAV sha256 = %s, want %s; max PCM diff = %d, want <= 1", actualSHA256, testCase.WAV.SHA256, maxDiff)
			}
		})
	}
}

func TestRenderNotesWAVPeakNormalisationKeepsGainOnlyChangesByteIdentical(t *testing.T) {
	notes := []Note{{Pitch: 69, Start: 0, End: 0.5, Velocity: 100}}
	sampleRate := 48000
	base, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.5,
		Volume: 1.0,
	}})
	if err != nil {
		t.Fatal(err)
	}
	quiet, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.5,
		Volume: 0.2,
	}})
	if err != nil {
		t.Fatal(err)
	}
	flatCurve, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.5,
		Volume: 1.0,
		FrequencyCurve: []FrequencyCurvePoint{
			{FrequencyHz: MinCurveFrequencyHz, GainDB: -12.0},
			{FrequencyHz: MaxCurveFrequencyHz, GainDB: -12.0},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(base, quiet) {
		t.Fatal("volume-only gain change should remain byte-identical after peak normalisation")
	}
	if !bytes.Equal(base, flatCurve) {
		t.Fatal("flat frequency-curve gain should remain byte-identical after peak normalisation")
	}
}

func TestRenderNotesWAVAppliesShapeAndRelativeLayerChanges(t *testing.T) {
	notes := []Note{
		{Pitch: 60, Start: 0, End: 0.25, Velocity: 96},
		{Pitch: 64, Start: 0.3, End: 0.55, Velocity: 104},
		{Pitch: 67, Start: 0.6, End: 0.9, Velocity: 112},
	}
	sampleRate := 44100
	base, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.5,
		Volume: 1.0,
	}})
	if err != nil {
		t.Fatal(err)
	}
	narrowPulse, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.25,
		Volume: 1.0,
	}})
	if err != nil {
		t.Fatal(err)
	}
	tiltedCurve, err := RenderNotesWAV(notes, sampleRate, []Layer{{
		Type:   "pulse",
		Duty:   0.5,
		Volume: 1.0,
		FrequencyCurve: []FrequencyCurvePoint{
			{FrequencyHz: MinCurveFrequencyHz, GainDB: -12.0},
			{FrequencyHz: 440.0, GainDB: 0.0},
			{FrequencyHz: MaxCurveFrequencyHz, GainDB: 12.0},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	pulseLead, err := RenderNotesWAV(notes, sampleRate, []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 1.0},
		{Type: "sine", Duty: 0.5, Volume: 0.2},
	})
	if err != nil {
		t.Fatal(err)
	}
	sineLead, err := RenderNotesWAV(notes, sampleRate, []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 0.2},
		{Type: "sine", Duty: 0.5, Volume: 1.0},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertDifferentWAVBytes(t, base, narrowPulse, "pulse duty change")
	assertDifferentWAVBytes(t, base, tiltedCurve, "non-flat frequency curve")
	assertDifferentWAVBytes(t, pulseLead, sineLead, "relative layer balance")
}

func TestInvalidLayerErrorsMatchPythonBaseline(t *testing.T) {
	expectations := loadRendererExpectations(t)

	_, err := ParseLayersJSON(`[
		{
			"type": "sine",
			"duty": 0.5,
			"volume": 1.0,
			"frequency_curve": [
				{"frequency_hz": 440.0, "gain_db": 0.0},
				{"frequency_hz": 440.0, "gain_db": -6.0}
			]
		}
	]`)
	if err == nil || err.Error() != expectations.InvalidLayerErrors["duplicate_curve_frequency"] {
		t.Fatalf("duplicate frequency error = %v", err)
	}

	_, err = ParseLayersJSON(`[
		{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]},
		{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]},
		{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]},
		{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]},
		{"type":"pulse","duty":0.5,"volume":1.0,"frequency_curve":[]}
	]`)
	var limitErr RenderLimitError
	if !errors.As(err, &limitErr) || err.Error() != expectations.InvalidLayerErrors["too_many_layers"] {
		t.Fatalf("too many layers error = %v", err)
	}
}

func TestNormaliseRuntimeLayersDropsSilentLayers(t *testing.T) {
	layers, err := NormaliseRuntimeLayers([]Layer{
		{Type: "sine", Duty: 0.5, Volume: 0, FrequencyCurve: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 1 || layers[0].Type != "pulse" || layers[0].Volume != 1.0 {
		t.Fatalf("silent layers did not fall back to default pulse layer: %#v", layers)
	}
}

func assertDifferentWAVBytes(t *testing.T, left, right []byte, label string) {
	t.Helper()
	if bytes.Equal(left, right) {
		t.Fatalf("%s should change rendered WAV bytes", label)
	}
}

func loadRendererExpectations(t *testing.T) rendererExpectations {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "python-baseline", "renderer", "expectations.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var expectations rendererExpectations
	if err := json.Unmarshal(data, &expectations); err != nil {
		t.Fatal(err)
	}
	return expectations
}

func (c rendererCase) MIDIName() string {
	if c.MIDI != "" {
		return c.MIDI
	}
	return c.Name + ".mid"
}

func trimMIDIExtension(name string) string {
	extension := filepath.Ext(name)
	if extension == "" {
		return name
	}
	return name[:len(name)-len(extension)]
}

func parseFrequencyFixtureKey(value string) (float64, error) {
	var parsed float64
	err := json.Unmarshal([]byte(value), &parsed)
	return parsed, err
}

func readBaselineWAV(t *testing.T, caseName string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "python-baseline", "renderer", caseName+".wav"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func maxPCM16AbsDiff(t *testing.T, actualWAV, expectedWAV []byte) int {
	t.Helper()
	if len(actualWAV) != len(expectedWAV) {
		t.Fatalf("WAV byte lengths differ: actual %d, expected %d", len(actualWAV), len(expectedWAV))
	}
	if len(actualWAV) < 44 || string(actualWAV[:4]) != "RIFF" || string(expectedWAV[:4]) != "RIFF" {
		t.Fatalf("invalid WAV header")
	}
	if string(actualWAV[:44]) != string(expectedWAV[:44]) {
		t.Fatalf("WAV headers differ")
	}

	maxDiff := 0
	for offset := 44; offset < len(actualWAV); offset += 2 {
		actual := int(int16(binary.LittleEndian.Uint16(actualWAV[offset : offset+2])))
		expected := int(int16(binary.LittleEndian.Uint16(expectedWAV[offset : offset+2])))
		diff := actual - expected
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}
