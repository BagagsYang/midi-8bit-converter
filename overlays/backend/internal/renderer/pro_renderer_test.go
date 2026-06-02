package renderer

import (
	"bytes"
	"encoding/binary"
	"math"
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

func TestProDetuneMatchesEquivalentPitchOffset(t *testing.T) {
	baseNote := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}
	sampleRate := 48000

	detuned, err := RenderNotesWAV(baseNote, sampleRate, []Layer{{Type: "sine", Duty: 0.5, Volume: 1, DetuneCents: 100}})
	if err != nil {
		t.Fatal(err)
	}
	transposed, err := RenderNotesWAV([]Note{{Pitch: 61, Start: 0, End: 0.25, Velocity: 100, Channel: 1}}, sampleRate, []Layer{{Type: "sine", Duty: 0.5, Volume: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if diff := proMaxPCM16AbsDiff(t, detuned, transposed); diff > 1 {
		t.Fatalf("100 cent detune should match one semitone transposition; max PCM diff = %d", diff)
	}
}

func TestProChannelBusVolumeMuteAndSolo(t *testing.T) {
	notes := []Note{
		{Pitch: 60, Start: 0, End: 0.25, Velocity: 100, Channel: 1},
		{Pitch: 64, Start: 0, End: 0.25, Velocity: 100, Channel: 2},
	}
	layers := []Layer{{Type: "sine", Duty: 0.5, Volume: 1}}
	sampleRate := 48000

	channelOneOnly, err := RenderNotesWAVWithOptions(notes, sampleRate, layers, RenderOptions{
		ChannelBuses: []ChannelBus{{Channel: 1, Volume: 1, Solo: true}, {Channel: 2, Volume: 1}},
		Output:       OutputOptions{LimiterEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	channelOneReference, err := RenderNotesWAVWithOptions([]Note{notes[0]}, sampleRate, layers, RenderOptions{
		Output: OutputOptions{LimiterEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := proMaxPCM16AbsDiff(t, channelOneOnly, channelOneReference); diff > 1 {
		t.Fatalf("soloed channel should match channel one reference; max PCM diff = %d", diff)
	}

	mutedChannelOne, err := RenderNotesWAVWithOptions(notes, sampleRate, layers, RenderOptions{
		ChannelBuses: []ChannelBus{{Channel: 1, Volume: 1, Mute: true}, {Channel: 2, Volume: 1}},
		Output:       OutputOptions{LimiterEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	channelTwoReference, err := RenderNotesWAVWithOptions([]Note{notes[1]}, sampleRate, layers, RenderOptions{
		Output: OutputOptions{LimiterEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := proMaxPCM16AbsDiff(t, mutedChannelOne, channelTwoReference); diff > 1 {
		t.Fatalf("muted channel should leave channel two reference; max PCM diff = %d", diff)
	}
}

func TestProLimiterPreventsClipping(t *testing.T) {
	notes := []Note{{Pitch: 60, Start: 0, End: 0.25, Velocity: 127, Channel: 1}}
	layers := []Layer{
		{Type: "pulse", Duty: 0.5, Volume: 2},
		{Type: "sawtooth", Duty: 0.5, Volume: 2},
	}

	limited, err := RenderNotesWAVWithOptions(notes, 48000, layers, RenderOptions{
		Output: OutputOptions{MasterGainDB: 12, LimiterEnabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proWAVHasSignal(limited) {
		t.Fatal("limited render should retain signal")
	}
	if peak := proMaxPCM16Abs(limited); peak > 32767 {
		t.Fatalf("limited render peak = %d, want <= 32767", peak)
	}
}

func TestProPolyBLEPChangesHighSawAndPulse(t *testing.T) {
	saw := generateStaticWaveform(7902.13, 48000, 512, "sawtooth", 0.5)
	pulse := generateStaticWaveform(7902.13, 48000, 512, "pulse", 0.5)

	if proFloat32Equal(saw, proLegacyStaticWaveform(7902.13, 48000, 512, "sawtooth", 0.5)) {
		t.Fatal("high sawtooth should differ from legacy naive oscillator")
	}
	if proFloat32Equal(pulse, proLegacyStaticWaveform(7902.13, 48000, 512, "pulse", 0.5)) {
		t.Fatal("high pulse should differ from legacy naive oscillator")
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

func proMaxPCM16Abs(wavBytes []byte) int {
	maxValue := 0
	for offset := 44; offset+1 < len(wavBytes); offset += 2 {
		value := int(int16(binary.LittleEndian.Uint16(wavBytes[offset : offset+2])))
		if value < 0 {
			value = -value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func proFloat32Equal(left, right []float32) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func proLegacyStaticWaveform(frequency float64, sampleRate int, sampleCount int, waveType string, dutyCycle float64) []float32 {
	waveform := make([]float32, sampleCount)
	floatSampleRate := float32(sampleRate)
	floatFrequency := float32(frequency)
	for index := range waveform {
		t := float32(index) / floatSampleRate
		cyclePosition := float32(math.Mod(float64(t*floatFrequency), 1.0))
		switch waveType {
		case "sawtooth":
			waveform[index] = 2.0*cyclePosition - 1.0
		case "pulse":
			if cyclePosition < float32(dutyCycle) {
				waveform[index] = 1.0
			} else {
				waveform[index] = -1.0
			}
		}
	}
	return waveform
}
