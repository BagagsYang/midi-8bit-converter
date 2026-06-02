package midi

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"octabit/backend/internal/renderer"
)

type baselineManifest struct {
	MIDI map[string]struct {
		Path      string          `json:"path"`
		NoteSpecs []renderer.Note `json:"note_specs"`
	} `json:"midi"`
}

type rendererExpectations struct {
	Cases []struct {
		Name       string           `json:"name"`
		MIDI       string           `json:"midi"`
		Layers     []renderer.Layer `json:"layers"`
		SampleRate int              `json:"sample_rate"`
		WAV        struct {
			SHA256 string `json:"sha256"`
			Size   int    `json:"size"`
		} `json:"wav"`
	} `json:"cases"`
}

func TestReadNotesMatchesPythonBaselineFixtures(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata", "python-baseline")
	manifest := loadBaselineManifest(t, filepath.Join(fixtureRoot, "manifest.json"))

	for name, metadata := range manifest.MIDI {
		t.Run(name, func(t *testing.T) {
			notes, err := ReadNotes(filepath.Join(fixtureRoot, metadata.Path))
			if err != nil {
				t.Fatal(err)
			}
			assertNotesAlmostEqual(t, notes, metadata.NoteSpecs)
		})
	}
}

func TestReadNotesThenRenderWAVMatchesPythonBaseline(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata", "python-baseline")
	expectations := loadRendererExpectations(t, filepath.Join(fixtureRoot, "renderer", "expectations.json"))

	for _, testCase := range expectations.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			notes, err := ReadNotes(filepath.Join(fixtureRoot, "midi", testCase.MIDI))
			if err != nil {
				t.Fatal(err)
			}
			wavBytes, err := renderer.RenderNotesWAV(notes, testCase.SampleRate, testCase.Layers)
			if err != nil {
				t.Fatal(err)
			}
			if len(wavBytes) != testCase.WAV.Size {
				t.Fatalf("WAV size = %d, want %d", len(wavBytes), testCase.WAV.Size)
			}
			hash := sha256.Sum256(wavBytes)
			if actualSHA256 := fmt.Sprintf("%x", hash[:]); actualSHA256 != testCase.WAV.SHA256 {
				expectedWAV := readBaselineWAV(t, fixtureRoot, testCase.Name)
				maxDiff := maxPCM16AbsDiff(t, wavBytes, expectedWAV)
				if maxDiff > 1 {
					t.Fatalf("WAV sha256 = %s, want %s; max PCM diff = %d, want <= 1", actualSHA256, testCase.WAV.SHA256, maxDiff)
				}
			}
		})
	}
}

func loadBaselineManifest(t *testing.T, path string) baselineManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest baselineManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func loadRendererExpectations(t *testing.T, path string) rendererExpectations {
	t.Helper()
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

func assertNotesAlmostEqual(t *testing.T, actual, expected []renderer.Note) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("note count = %d, want %d; notes = %#v", len(actual), len(expected), actual)
	}
	for index := range actual {
		if actual[index].Pitch != expected[index].Pitch {
			t.Fatalf("note %d pitch = %d, want %d", index, actual[index].Pitch, expected[index].Pitch)
		}
		if actual[index].Velocity != expected[index].Velocity {
			t.Fatalf("note %d velocity = %d, want %d", index, actual[index].Velocity, expected[index].Velocity)
		}
		if math.Abs(actual[index].Start-expected[index].Start) > 1e-15 {
			t.Fatalf("note %d start = %.17f, want %.17f", index, actual[index].Start, expected[index].Start)
		}
		if math.Abs(actual[index].End-expected[index].End) > 1e-15 {
			t.Fatalf("note %d end = %.17f, want %.17f", index, actual[index].End, expected[index].End)
		}
	}
}

func readBaselineWAV(t *testing.T, fixtureRoot, caseName string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "renderer", caseName+".wav"))
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
