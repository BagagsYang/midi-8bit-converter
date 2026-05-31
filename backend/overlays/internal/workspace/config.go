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
	Schema     string        `json:"schema"`
	SampleRate int           `json:"sample_rate"`
	Layers     []LayerConfig `json:"layers"`
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
	VibratoDepthCents float64 `json:"vibrato_depth_cents"`
	VibratoRateHz     float64 `json:"vibrato_rate_hz"`
	OctaveShift       int     `json:"octave_shift"`
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
			VibratoDepthCents: layerPro.VibratoDepthCents,
			VibratoRateHz:     layerPro.VibratoRateHz,
			OctaveShift:       layerPro.OctaveShift,
		})
	}
	if _, err := renderer.NormaliseRuntimeLayers(rendererValidationLayers); err != nil {
		return Config{}, err
	}

	return Config{
		Schema:     ConfigSchema,
		SampleRate: sampleRate,
		Layers:     normalisedLayers,
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
	return len(layer.MIDIChannels) > 0 || layer.VibratoDepthCents > 0 || layer.OctaveShift != 0
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
