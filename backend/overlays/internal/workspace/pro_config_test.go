package workspace

import (
	"strings"
	"testing"

	"octabit/backend/internal/renderer"
)

func TestProOctaveShiftNormalisesAndReachesRuntimeLayer(t *testing.T) {
	config, err := NormaliseConfig(proOctaveRawConfig(float64(2)))
	if err != nil {
		t.Fatal(err)
	}
	if config.Layers[0].Pro == nil {
		t.Fatal("Pro config should be present")
	}
	if config.Layers[0].Pro.OctaveShift != 2 {
		t.Fatalf("OctaveShift = %d, want 2", config.Layers[0].Pro.OctaveShift)
	}

	runtimeLayers := RuntimeLayers(config)
	if runtimeLayers[0].OctaveShift != 2 {
		t.Fatalf("runtime OctaveShift = %d, want 2", runtimeLayers[0].OctaveShift)
	}
	if !requiresMultiChannel(runtimeLayers) {
		t.Fatal("non-zero octave shift should require multi-channel MIDI")
	}
}

func TestProDetuneAndOutputConfigNormalise(t *testing.T) {
	rawConfig := proOctaveRawConfig(float64(0))
	rawConfig["master_gain_db"] = float64(3.5)
	rawConfig["limiter_enabled"] = true
	rawConfig["normalise_enabled"] = false
	rawConfig["channel_buses"] = []any{
		map[string]any{"channel": float64(2), "volume": float64(0.75), "mute": true, "solo": false},
	}
	rawConfig["layers"].([]any)[0].(map[string]any)["pro"].(map[string]any)["detune_cents"] = float64(12.5)

	config, err := NormaliseConfig(rawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Layers[0].Pro == nil || config.Layers[0].Pro.DetuneCents != 12.5 {
		t.Fatalf("detune = %#v, want 12.5", config.Layers[0].Pro)
	}
	if len(config.ChannelBuses) != 1 || config.ChannelBuses[0].Channel != 2 || config.ChannelBuses[0].Volume != 0.75 || !config.ChannelBuses[0].Mute {
		t.Fatalf("channel buses = %#v", config.ChannelBuses)
	}
	if config.MasterGainDB != 3.5 || !config.LimiterEnabled || config.NormaliseEnabled {
		t.Fatalf("output config = gain %.1f limiter %v normalise %v", config.MasterGainDB, config.LimiterEnabled, config.NormaliseEnabled)
	}

	runtimeLayers := RuntimeLayers(config)
	if runtimeLayers[0].DetuneCents != 12.5 {
		t.Fatalf("runtime detune = %.1f, want 12.5", runtimeLayers[0].DetuneCents)
	}
	renderOptions := RenderOptions(config)
	if len(renderOptions.ChannelBuses) != 1 || renderOptions.Output.MasterGainDB != 3.5 || !renderOptions.Output.LimiterEnabled {
		t.Fatalf("render options = %#v", renderOptions)
	}
}

func TestProOctaveShiftDefaultsToZero(t *testing.T) {
	rawConfig := proOctaveRawConfig(nil)
	delete(rawConfig["layers"].([]any)[0].(map[string]any), "pro")

	config, err := NormaliseConfig(rawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Layers[0].Pro != nil {
		t.Fatalf("Pro config = %#v, want nil", config.Layers[0].Pro)
	}
	runtimeLayers := RuntimeLayers(config)
	if runtimeLayers[0].OctaveShift != 0 {
		t.Fatalf("runtime OctaveShift = %d, want 0", runtimeLayers[0].OctaveShift)
	}
	if requiresMultiChannel([]renderer.Layer{{Type: "pulse", Duty: 0.5, Volume: 1, OctaveShift: 0}}) {
		t.Fatal("zero octave shift should not require multi-channel MIDI")
	}
}

func TestProOctaveShiftRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "too low", value: float64(-3), want: "Layer 1 octave_shift must be between -2 and 2."},
		{name: "too high", value: float64(3), want: "Layer 1 octave_shift must be between -2 and 2."},
		{name: "fractional", value: float64(1.5), want: "Layer 1 octave_shift must be an integer."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormaliseConfig(proOctaveRawConfig(test.value))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestProDetuneRejectsInvalidValues(t *testing.T) {
	rawConfig := proOctaveRawConfig(float64(0))
	rawConfig["layers"].([]any)[0].(map[string]any)["pro"].(map[string]any)["detune_cents"] = float64(101)

	_, err := NormaliseConfig(rawConfig)
	if err == nil || !strings.Contains(err.Error(), "Layer 1 detune_cents must be between -100 and 100.") {
		t.Fatalf("error = %v", err)
	}
}

func TestProWorkspaceConfigAcceptsTenLayers(t *testing.T) {
	rawConfig := proOctaveRawConfig(nil)
	layer := rawConfig["layers"].([]any)[0]
	layers := make([]any, 10)
	for index := range layers {
		layers[index] = layer
	}
	rawConfig["layers"] = layers

	config, err := NormaliseConfig(rawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Layers) != 10 {
		t.Fatalf("layer count = %d, want 10", len(config.Layers))
	}

	rawConfig["layers"] = append(layers, layer)
	_, err = NormaliseConfig(rawConfig)
	if err == nil || !strings.Contains(err.Error(), "between 1 and 10 layers") {
		t.Fatalf("11 layers error = %v", err)
	}
}

func proOctaveRawConfig(octaveShift any) map[string]any {
	pro := map[string]any{
		"midi_channels":       []any{},
		"detune_cents":        float64(0),
		"vibrato_depth_cents": float64(0),
		"vibrato_rate_hz":     float64(5),
		"octave_shift":        octaveShift,
	}
	if octaveShift == nil {
		delete(pro, "octave_shift")
	}
	return map[string]any{
		"schema":      ConfigSchema,
		"sample_rate": float64(48000),
		"layers": []any{
			map[string]any{
				"type":            "pulse",
				"duty":            float64(0.5),
				"volume":          float64(1),
				"curve_enabled":   false,
				"frequency_curve": []any{},
				"pro":             pro,
			},
		},
	}
}
