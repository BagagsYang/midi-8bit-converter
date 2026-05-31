package renderer

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestProNoiseAndVibratoChangeRenderedOutput(t *testing.T) {
	notes := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	sampleRate := 48000

	base, err := RenderNotesWAV(notes, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	noise, err := RenderNotesWAV(notes, sampleRate, []Layer{{Type: "noise", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	vibrato, err := RenderNotesWAV(notes, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1, VibratoDepthCents: 80, VibratoRateHz: 5}})
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(base, noise) {
		t.Fatal("noise layer should change rendered output")
	}
	if bytes.Equal(base, vibrato) {
		t.Fatal("vibrato should change rendered output")
	}
}

func TestProChannelFilteringRoutesLayers(t *testing.T) {
	notes := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	sampleRate := 48000

	allChannels, err := RenderNotesWAV(notes, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	filteredOut, err := RenderNotesWAV(notes, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1, MIDIChannels: []int{2}}})
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(allChannels, filteredOut) {
		t.Fatal("layer routed to another channel should not match the audible render")
	}
}

func TestProOctaveShiftTransposesLayer(t *testing.T) {
	baseNote := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	transposedNote := []Note{{Pitch: 72, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	sampleRate := 48000

	base, err := RenderNotesWAV(baseNote, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	shifted, err := RenderNotesWAV(baseNote, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1, OctaveShift: 1}})
	if err != nil {
		t.Fatal(err)
	}
	transposed, err := RenderNotesWAV(transposedNote, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(base, shifted) {
		t.Fatal("octave shift should change rendered output")
	}
	if diff := proMaxPCM16AbsDiff(t, shifted, transposed); diff > 1 {
		t.Fatalf("octave shift should match one-octave transposition; max PCM diff = %d", diff)
	}
}

func proMaxPCM16AbsDiff(t *testing.T, left, right []byte) int {
	t.Helper()
	if len(left) != len(right) {
		t.Fatalf("WAV length mismatch: %d != %d", len(left), len(right))
	}
	maxDiff := 0
	for offset := 44; offset+1 < len(left); offset += 2 {
		leftSample := int(int16(binary.LittleEndian.Uint16(left[offset : offset+2])))
		rightSample := int(int16(binary.LittleEndian.Uint16(right[offset : offset+2])))
		diff := leftSample - rightSample
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
}
