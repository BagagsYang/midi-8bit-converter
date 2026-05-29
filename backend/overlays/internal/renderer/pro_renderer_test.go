package renderer

import (
	"bytes"
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
