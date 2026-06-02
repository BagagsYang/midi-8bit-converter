package workspace

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"octabit/backend/internal/renderer"
)

const ConfigSchema = "octabit.workspace_config.v1"

var allowedSampleRates = map[int]struct{}{
	44100: {},
	48000: {},
	96000: {},
}

type Config struct {
	Schema           string             `json:"schema"`
	SampleRate       int                `json:"sample_rate"`
	Layers           []LayerConfig      `json:"layers"`
	ChannelBuses     []ChannelBusConfig `json:"channel_buses,omitempty"`
	MasterGainDB     float64            `json:"master_gain_db"`
	LimiterEnabled   bool               `json:"limiter_enabled"`
	NormaliseEnabled bool               `json:"normalise_enabled"`
}

type LayerConfig struct {
	Type           string                         `json:"type"`
	Duty           float64                        `json:"duty"`
	Volume         float64                        `json:"volume"`
	CurveEnabled   bool                           `json:"curve_enabled"`
	FrequencyCurve []renderer.FrequencyCurvePoint `json:"frequency_curve"`
	Pro            *LayerProConfig                `json:"pro,omitempty"`
}

type LayerProConfig struct {
	MIDIChannels      []int   `json:"midi_channels"`
	DetuneCents       float64 `json:"detune_cents"`
	VibratoDepthCents float64 `json:"vibrato_depth_cents"`
	VibratoRateHz     float64 `json:"vibrato_rate_hz"`
	OctaveShift       int     `json:"octave_shift"`
}

type ChannelBusConfig struct {
	Channel int     `json:"channel"`
	Volume  float64 `json:"volume"`
	Mute    bool    `json:"mute"`
	Solo    bool    `json:"solo"`
}

type FormPayload struct {
	Rate       string `json:"rate"`
	LayersJSON string `json:"layers_json"`
}

func ConfigFromFormValues(rawRate, layersJSON string) (Config, error) {
	sampleRate, err := parseSampleRate(formValueOrDefault(rawRate, "48000"))
	if err != nil {
		return Config{}, err
	}
	trimmedLayersJSON := strings.TrimSpace(layersJSON)
	if trimmedLayersJSON == "" {
		trimmedLayersJSON = "[]"
	}
	parsedLayers, err := renderer.ParseLayersJSON(trimmedLayersJSON)
	if err != nil {
		return Config{}, err
	}
	if len(parsedLayers) == 0 {
		defaultConfig := DefaultConfig()
		defaultConfig.SampleRate = sampleRate
		return defaultConfig, nil
	}

	rawLayers := make([]any, 0, len(parsedLayers))
	for _, layer := range parsedLayers {
		rawLayers = append(rawLayers, map[string]any{
			"type":            layer.Type,
			"duty":            layer.Duty,
			"volume":          layer.Volume,
			"curve_enabled":   len(layer.FrequencyCurve) > 0,
			"frequency_curve": frequencyCurveAsRaw(layer.FrequencyCurve),
		})
	}
	return NormaliseConfig(map[string]any{
		"schema":      ConfigSchema,
		"sample_rate": sampleRate,
		"layers":      rawLayers,
	})
}

func frequencyCurveAsRaw(curve []renderer.FrequencyCurvePoint) []any {
	rawCurve := make([]any, 0, len(curve))
	for _, point := range curve {
		rawCurve = append(rawCurve, map[string]any{
			"frequency_hz": point.FrequencyHz,
			"gain_db":      point.GainDB,
		})
	}
	return rawCurve
}

func NormaliseConfig(raw map[string]any) (Config, error) {
	if raw == nil {
		return Config{}, fmt.Errorf("Workspace config must be a JSON object.")
	}
	if schema, _ := raw["schema"].(string); schema != ConfigSchema {
		return Config{}, fmt.Errorf("Workspace config schema must be %s.", ConfigSchema)
	}

	sampleRate, err := parseSampleRate(raw["sample_rate"])
	if err != nil {
		return Config{}, err
	}

	rawLayers, ok := raw["layers"].([]any)
	if !ok {
		return Config{}, fmt.Errorf("Workspace config layers must be an array.")
	}
	if len(rawLayers) < 1 || len(rawLayers) > renderer.MaxProRenderLayers {
		return Config{}, fmt.Errorf("Workspace config supports between 1 and %d layers.", renderer.MaxProRenderLayers)
	}
	channelBuses, err := normaliseChannelBuses(raw["channel_buses"])
	if err != nil {
		return Config{}, err
	}
	masterGainDB, err := parseOptionalConfigNumber(raw, "master_gain_db", 0)
	if err != nil {
		return Config{}, err
	}
	if masterGainDB < renderer.MinMasterGainDB || masterGainDB > renderer.MaxMasterGainDB {
		return Config{}, fmt.Errorf("Workspace config master_gain_db must be between %.1f and %.1f.", renderer.MinMasterGainDB, renderer.MaxMasterGainDB)
	}
	limiterEnabled, err := parseOptionalConfigBool(raw, "limiter_enabled", true)
	if err != nil {
		return Config{}, err
	}
	normaliseEnabled, err := parseOptionalConfigBool(raw, "normalise_enabled", false)
	if err != nil {
		return Config{}, err
	}

	normalisedLayers := make([]LayerConfig, 0, len(rawLayers))
	rendererValidationLayers := make([]renderer.Layer, 0, len(rawLayers))
	for index, rawLayerValue := range rawLayers {
		rawLayer, ok := rawLayerValue.(map[string]any)
		if !ok {
			return Config{}, fmt.Errorf("Layer %d must be an object.", index+1)
		}

		curveEnabled, ok := rawLayer["curve_enabled"].(bool)
		if !ok {
			if _, exists := rawLayer["curve_enabled"]; exists {
				return Config{}, fmt.Errorf("Layer %d curve_enabled must be a boolean.", index+1)
			}
			curveEnabled = false
		}

		rendererLayer, err := renderer.SanitiseLayer(map[string]any{
			"type":                rawLayer["type"],
			"duty":                rawLayer["duty"],
			"volume":              rawLayer["volume"],
			"frequency_curve":     rawLayer["frequency_curve"],
			"midi_channels":       rawProValue(rawLayer["pro"], "midi_channels"),
			"detune_cents":        rawProValue(rawLayer["pro"], "detune_cents"),
			"vibrato_depth_cents": rawProValue(rawLayer["pro"], "vibrato_depth_cents"),
			"vibrato_rate_hz":     rawProValue(rawLayer["pro"], "vibrato_rate_hz"),
			"octave_shift":        rawProValue(rawLayer["pro"], "octave_shift"),
		}, index+1)
		if err != nil {
			return Config{}, err
		}
		if rendererLayer.Volume > 2.0 {
			return Config{}, fmt.Errorf("Layer %d volume must be between 0.0 and 2.0.", index+1)
		}

		normalisedPro := LayerProConfig{
			MIDIChannels:      rendererLayer.MIDIChannels,
			DetuneCents:       roundConfigNumber(rendererLayer.DetuneCents, 2),
			VibratoDepthCents: roundConfigNumber(rendererLayer.VibratoDepthCents, 2),
			VibratoRateHz:     roundConfigNumber(rendererLayer.VibratoRateHz, 2),
			OctaveShift:       rendererLayer.OctaveShift,
		}
		var proConfig *LayerProConfig
		if shouldIncludeProConfig(rawLayer["pro"], rendererLayer) {
			proConfig = &normalisedPro
		}

		normalisedLayer := LayerConfig{
			Type:           rendererLayer.Type,
			Duty:           roundConfigNumber(rendererLayer.Duty, 4),
			Volume:         roundConfigNumber(rendererLayer.Volume, 4),
			CurveEnabled:   curveEnabled,
			FrequencyCurve: roundFrequencyCurve(rendererLayer.FrequencyCurve),
			Pro:            proConfig,
		}
		layerPro := layerProConfig(normalisedLayer)
		normalisedLayers = append(normalisedLayers, normalisedLayer)
		rendererValidationLayers = append(rendererValidationLayers, renderer.Layer{
			Type:              normalisedLayer.Type,
			Duty:              normalisedLayer.Duty,
			Volume:            normalisedLayer.Volume,
			FrequencyCurve:    normalisedLayer.FrequencyCurve,
			MIDIChannels:      layerPro.MIDIChannels,
			DetuneCents:       layerPro.DetuneCents,
			VibratoDepthCents: layerPro.VibratoDepthCents,
			VibratoRateHz:     layerPro.VibratoRateHz,
			OctaveShift:       layerPro.OctaveShift,
		})
	}
	if _, err := renderer.NormaliseRuntimeLayers(rendererValidationLayers); err != nil {
		return Config{}, err
	}

	return Config{
		Schema:           ConfigSchema,
		SampleRate:       sampleRate,
		Layers:           normalisedLayers,
		ChannelBuses:     channelBuses,
		MasterGainDB:     roundConfigNumber(masterGainDB, 2),
		LimiterEnabled:   limiterEnabled,
		NormaliseEnabled: normaliseEnabled,
	}, nil
}

func rawProValue(rawPro any, key string) any {
	if rawPro == nil {
		return nil
	}
	proObject, ok := rawPro.(map[string]any)
	if !ok {
		return nil
	}
	return proObject[key]
}

func shouldIncludeProConfig(rawPro any, layer renderer.Layer) bool {
	if _, ok := rawPro.(map[string]any); ok {
		return true
	}
	return len(layer.MIDIChannels) > 0 || layer.DetuneCents != 0 || layer.VibratoDepthCents > 0 || layer.OctaveShift != 0
}

func layerProConfig(layer LayerConfig) LayerProConfig {
	if layer.Pro == nil {
		return LayerProConfig{}
	}
	return *layer.Pro
}

func formValueOrDefault(value, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}
	return trimmed
}

func ConfigToFormPayload(config Config) (FormPayload, error) {
	rendererLayers := make([]renderer.Layer, 0, len(config.Layers))
	for _, layer := range config.Layers {
		frequencyCurve := []renderer.FrequencyCurvePoint{}
		if layer.CurveEnabled {
			frequencyCurve = layer.FrequencyCurve
		}
		rendererLayers = append(rendererLayers, renderer.Layer{
			Type:           layer.Type,
			Duty:           layer.Duty,
			Volume:         layer.Volume,
			FrequencyCurve: frequencyCurve,
		})
	}
	layersJSON, err := marshalPythonLikeRendererLayers(rendererLayers)
	if err != nil {
		return FormPayload{}, err
	}
	return FormPayload{
		Rate:       strconv.Itoa(config.SampleRate),
		LayersJSON: layersJSON,
	}, nil
}

func parseSampleRate(value any) (int, error) {
	var sampleRate int
	switch typedValue := value.(type) {
	case float64:
		if typedValue != math.Trunc(typedValue) {
			return 0, unsupportedSampleRateError()
		}
		sampleRate = int(typedValue)
	case int:
		sampleRate = typedValue
	case string:
		parsedValue, err := strconv.Atoi(typedValue)
		if err != nil {
			return 0, unsupportedSampleRateError()
		}
		sampleRate = parsedValue
	default:
		return 0, unsupportedSampleRateError()
	}
	if _, ok := allowedSampleRates[sampleRate]; !ok {
		return 0, unsupportedSampleRateError()
	}
	return sampleRate, nil
}

func normaliseChannelBuses(rawValue any) ([]ChannelBusConfig, error) {
	if rawValue == nil {
		return []ChannelBusConfig{}, nil
	}
	rawItems, ok := rawValue.([]any)
	if !ok {
		return nil, fmt.Errorf("Workspace config channel_buses must be an array.")
	}
	buses := make([]ChannelBusConfig, 0, len(rawItems))
	seen := map[int]struct{}{}
	for index, rawItem := range rawItems {
		rawBus, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Channel bus %d must be an object.", index+1)
		}
		channelFloat, err := parseConfigFiniteNumber(rawBus["channel"], fmt.Sprintf("Channel bus %d channel", index+1))
		if err != nil {
			return nil, err
		}
		channel := int(channelFloat)
		if channelFloat != float64(channel) {
			return nil, fmt.Errorf("Channel bus %d channel must be an integer.", index+1)
		}
		if channel < 1 || channel > 16 {
			return nil, fmt.Errorf("Channel bus %d channel must be between 1 and 16.", index+1)
		}
		volume, err := parseOptionalConfigNumber(rawBus, "volume", 1)
		if err != nil {
			return nil, fmt.Errorf("Channel bus %d %s", index+1, err.Error())
		}
		if volume < renderer.MinBusVolume || volume > renderer.MaxBusVolume {
			return nil, fmt.Errorf("Channel bus %d volume must be between %.1f and %.1f.", index+1, renderer.MinBusVolume, renderer.MaxBusVolume)
		}
		mute, err := parseOptionalConfigBool(rawBus, "mute", false)
		if err != nil {
			return nil, fmt.Errorf("Channel bus %d %s", index+1, err.Error())
		}
		solo, err := parseOptionalConfigBool(rawBus, "solo", false)
		if err != nil {
			return nil, fmt.Errorf("Channel bus %d %s", index+1, err.Error())
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		buses = append(buses, ChannelBusConfig{
			Channel: channel,
			Volume:  roundConfigNumber(volume, 4),
			Mute:    mute,
			Solo:    solo,
		})
	}
	return buses, nil
}

func parseOptionalConfigNumber(raw map[string]any, key string, defaultValue float64) (float64, error) {
	rawValue, exists := raw[key]
	if !exists || rawValue == nil {
		return defaultValue, nil
	}
	value, err := parseConfigFiniteNumber(rawValue, key)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseConfigFiniteNumber(rawValue any, fieldLabel string) (float64, error) {
	var value float64
	switch typedValue := rawValue.(type) {
	case float64:
		value = typedValue
	case float32:
		value = float64(typedValue)
	case int:
		value = float64(typedValue)
	case int64:
		value = float64(typedValue)
	case json.Number:
		parsedValue, err := typedValue.Float64()
		if err != nil {
			return 0, fmt.Errorf("%s must be a number.", fieldLabel)
		}
		value = parsedValue
	default:
		return 0, fmt.Errorf("%s must be a number.", fieldLabel)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s must be finite.", fieldLabel)
	}
	return value, nil
}

func parseOptionalConfigBool(raw map[string]any, key string, defaultValue bool) (bool, error) {
	rawValue, exists := raw[key]
	if !exists || rawValue == nil {
		return defaultValue, nil
	}
	value, ok := rawValue.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean.", key)
	}
	return value, nil
}

func unsupportedSampleRateError() error {
	return fmt.Errorf("Unsupported sample rate. Choose 44100, 48000, or 96000.")
}

func roundFrequencyCurve(curve []renderer.FrequencyCurvePoint) []renderer.FrequencyCurvePoint {
	rounded := make([]renderer.FrequencyCurvePoint, 0, len(curve))
	for _, point := range curve {
		rounded = append(rounded, renderer.FrequencyCurvePoint{
			FrequencyHz: point.FrequencyHz,
			GainDB:      roundConfigNumber(point.GainDB, 4),
		})
	}
	return rounded
}

func roundConfigNumber(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	rounded := math.Round(value*factor) / factor
	if rounded == 0 {
		return 0
	}
	return rounded
}

func marshalPythonLikeRendererLayers(layers []renderer.Layer) (string, error) {
	payload := make([]map[string]any, 0, len(layers))
	for _, layer := range layers {
		payload = append(payload, map[string]any{
			"type":            layer.Type,
			"duty":            layer.Duty,
			"volume":          layer.Volume,
			"frequency_curve": layer.FrequencyCurve,
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
