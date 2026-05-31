package renderer

import (
	"bytes"
	"encoding/binary"
	"strings"
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

func TestProChannelRoutingOnlyPlaysSelectedLayerForChannel(t *testing.T) {
	sampleRate := 48000
	channelOneNote := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	channelTwoNote := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 2}}
	channelThreeNote := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 3}}
	routedLayers := []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 1, MIDIChannels: []int{1}},
		{Type: "sine", Duty: 0.5, Volume: 1, MIDIChannels: []int{2}},
	}

	channelOneRouted, err := RenderNotesWAV(channelOneNote, sampleRate, routedLayers)
	if err != nil {
		t.Fatal(err)
	}
	channelOnePulse, err := RenderNotesWAV(channelOneNote, sampleRate, []Layer{{Type: "pulse", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if diff := proMaxPCM16AbsDiff(t, channelOneRouted, channelOnePulse); diff > 1 {
		t.Fatalf("channel 1 should use only the pulse layer; max PCM diff = %d", diff)
	}

	channelTwoRouted, err := RenderNotesWAV(channelTwoNote, sampleRate, routedLayers)
	if err != nil {
		t.Fatal(err)
	}
	channelTwoSine, err := RenderNotesWAV(channelTwoNote, sampleRate, []Layer{{Type: "sine", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if diff := proMaxPCM16AbsDiff(t, channelTwoRouted, channelTwoSine); diff > 1 {
		t.Fatalf("channel 2 should use only the sine layer; max PCM diff = %d", diff)
	}

	channelThreeRouted, err := RenderNotesWAV(channelThreeNote, sampleRate, routedLayers)
	if err != nil {
		t.Fatal(err)
	}
	if proWAVHasSignal(channelThreeRouted) {
		t.Fatal("unselected channel should not be audible when channel routing is active")
	}
}

func TestProOutputFilenameHashesChannelRouting(t *testing.T) {
	channelOneName, err := BuildOutputFilename("song", []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 1, MIDIChannels: []int{1}},
		{Type: "sine", Duty: 0.5, Volume: 1, MIDIChannels: []int{2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	channelTwoName, err := BuildOutputFilename("song", []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 1, MIDIChannels: []int{2}},
		{Type: "sine", Duty: 0.5, Volume: 1, MIDIChannels: []int{1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if channelOneName == channelTwoName {
		t.Fatalf("channel routing should affect output filename hash: %q", channelOneName)
	}
	if !strings.HasPrefix(channelOneName, "song_mix_") || !strings.HasPrefix(channelTwoName, "song_mix_") {
		t.Fatalf("routed filenames should include a mix hash: %q, %q", channelOneName, channelTwoName)
	}
}

func TestProRuntimeAllowsTenLayers(t *testing.T) {
	layers := make([]Layer, 10)
	for index := range layers {
		layers[index] = Layer{Type: "pulse", Duty: 0.5, Volume: 1}
	}
	if _, err := RenderNotesWAV([]Note{{Pitch: 60, Start: 0, End: 0.1, Velocity: 100, Channel: 1}}, 48000, layers); err != nil {
		t.Fatal(err)
	}

	_, err := RenderNotesWAV([]Note{{Pitch: 60, Start: 0, End: 0.1, Velocity: 100, Channel: 1}}, 48000, append(layers, Layer{Type: "pulse", Duty: 0.5, Volume: 1}))
	if err == nil || !strings.Contains(err.Error(), "at most 10 sound layers") {
		t.Fatalf("11 layers error = %v", err)
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

func proWAVHasSignal(wavBytes []byte) bool {
	for offset := 44; offset+1 < len(wavBytes); offset += 2 {
		if int16(binary.LittleEndian.Uint16(wavBytes[offset:offset+2])) != 0 {
			return true
		}
	}
	return false
}
