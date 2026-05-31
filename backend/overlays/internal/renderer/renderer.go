package renderer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	MinCurveFrequencyHz       = 8.175798915643707
	MaxCurveFrequencyHz       = 12543.853951415975
	MinCurveGainDB            = -36.0
	MaxCurveGainDB            = 12.0
	MaxCurvePoints            = 8
	CurveFrequencyToleranceHz = 1e-6
	MaxRenderSeconds          = 30 * 60
	MaxMIDINotes              = 20_000
	MaxRenderLayers           = 4
	MinOctaveShift            = -2
	MaxOctaveShift            = 2
	NormalisedPeak            = 0.89
)

var validWaveTypes = map[string]struct{}{
	"pulse":    {},
	"sine":     {},
	"sawtooth": {},
	"triangle": {},
	"noise":    {},
}

type FrequencyCurvePoint struct {
	FrequencyHz float64 `json:"frequency_hz"`
	GainDB      float64 `json:"gain_db"`
}

type Layer struct {
	Type              string                `json:"type"`
	Duty              float64               `json:"duty"`
	Volume            float64               `json:"volume"`
	FrequencyCurve    []FrequencyCurvePoint `json:"frequency_curve"`
	MIDIChannels      []int                 `json:"midi_channels,omitempty"`
	VibratoDepthCents float64               `json:"vibrato_depth_cents,omitempty"`
	VibratoRateHz     float64               `json:"vibrato_rate_hz,omitempty"`
	OctaveShift       int                   `json:"octave_shift,omitempty"`
}

type RenderLimitError struct {
	Message string
}

func (e RenderLimitError) Error() string {
	return e.Message
}

func ParseLayersJSON(layersJSON string) ([]Layer, error) {
	var rawLayers []map[string]any
	if err := json.Unmarshal([]byte(layersJSON), &rawLayers); err != nil {
		return nil, fmt.Errorf("Invalid layer JSON: %s", jsonErrorMessage(err))
	}

	if err := validateLayerCount(len(rawLayers)); err != nil {
		return nil, err
	}

	layers := make([]Layer, 0, len(rawLayers))
	for index, rawLayer := range rawLayers {
		layer, err := SanitiseLayer(rawLayer, index+1)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

func SanitiseLayer(rawLayer map[string]any, layerIndex int) (Layer, error) {
	waveType, _ := rawLayer["type"].(string)
	if waveType == "" {
		waveType = "pulse"
	}
	if _, ok := validWaveTypes[waveType]; !ok {
		return Layer{}, fmt.Errorf(
			"Layer %d has unsupported waveform '%s'. Expected one of: noise, pulse, sawtooth, sine, triangle.",
			layerIndex,
			waveType,
		)
	}

	duty, err := parseLayerNumber(rawLayer, layerIndex, "duty", 0.5)
	if err != nil {
		return Layer{}, err
	}
	if duty < 0.01 || duty > 0.99 {
		return Layer{}, fmt.Errorf("Layer %d duty must be between 0.01 and 0.99.", layerIndex)
	}

	volume, err := parseLayerNumber(rawLayer, layerIndex, "volume", 1.0)
	if err != nil {
		return Layer{}, err
	}
	if volume < 0 {
		return Layer{}, fmt.Errorf("Layer %d volume must be 0 or greater.", layerIndex)
	}

	curve, err := sanitiseFrequencyCurve(rawLayer["frequency_curve"], layerIndex)
	if err != nil {
		return Layer{}, err
	}

	midiChannels, err := sanitiseChannelList(rawLayer["midi_channels"], layerIndex)
	if err != nil {
		return Layer{}, err
	}
	vibratoDepthCents, err := parseLayerNumber(rawLayer, layerIndex, "vibrato_depth_cents", 0)
	if err != nil {
		return Layer{}, err
	}
	if vibratoDepthCents < 0 || vibratoDepthCents > 200 {
		return Layer{}, fmt.Errorf("Layer %d vibrato_depth_cents must be between 0 and 200.", layerIndex)
	}
	vibratoRateHz, err := parseLayerNumber(rawLayer, layerIndex, "vibrato_rate_hz", 5)
	if err != nil {
		return Layer{}, err
	}
	if vibratoDepthCents == 0 && vibratoRateHz == 0 {
		vibratoRateHz = 5
	}
	if vibratoRateHz < 0.1 || vibratoRateHz > 20 {
		return Layer{}, fmt.Errorf("Layer %d vibrato_rate_hz must be between 0.1 and 20.", layerIndex)
	}
	octaveShift, err := parseLayerInteger(rawLayer, layerIndex, "octave_shift", 0)
	if err != nil {
		return Layer{}, err
	}
	if octaveShift < MinOctaveShift || octaveShift > MaxOctaveShift {
		return Layer{}, fmt.Errorf("Layer %d octave_shift must be between %d and %d.", layerIndex, MinOctaveShift, MaxOctaveShift)
	}

	return Layer{
		Type:              waveType,
		Duty:              duty,
		Volume:            volume,
		FrequencyCurve:    curve,
		MIDIChannels:      midiChannels,
		VibratoDepthCents: vibratoDepthCents,
		VibratoRateHz:     vibratoRateHz,
		OctaveShift:       octaveShift,
	}, nil
}

func NormaliseRuntimeLayers(layers []Layer) ([]Layer, error) {
	if len(layers) == 0 {
		return []Layer{defaultLayer()}, nil
	}
	if err := validateLayerCount(len(layers)); err != nil {
		return nil, err
	}

	audibleLayers := make([]Layer, 0, len(layers))
	for _, layer := range layers {
		rawLayer := map[string]any{
			"type":                layer.Type,
			"duty":                layer.Duty,
			"volume":              layer.Volume,
			"frequency_curve":     curveAsRaw(layer.FrequencyCurve),
			"midi_channels":       layer.MIDIChannels,
			"vibrato_depth_cents": layer.VibratoDepthCents,
			"vibrato_rate_hz":     layer.VibratoRateHz,
			"octave_shift":        layer.OctaveShift,
		}
		sanitisedLayer, err := SanitiseLayer(rawLayer, len(audibleLayers)+1)
		if err != nil {
			return nil, err
		}
		if sanitisedLayer.Volume <= 0 {
			continue
		}
		audibleLayers = append(audibleLayers, sanitisedLayer)
	}
	if len(audibleLayers) == 0 {
		return []Layer{defaultLayer()}, nil
	}
	return audibleLayers, nil
}

func BuildCurvePayloadHash(layers []Layer) (string, error) {
	runtimeLayers, err := NormaliseRuntimeLayers(layers)
	if err != nil {
		return "", err
	}
	payload := canonicalLayersJSON(runtimeLayers)
	sum := sha1.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])[:8], nil
}

func BuildOutputSuffix(layers []Layer) (string, error) {
	runtimeLayers, err := NormaliseRuntimeLayers(layers)
	if err != nil {
		return "", err
	}
	suffix := runtimeLayers[0].Type
	if len(runtimeLayers) > 1 {
		suffix = "mix"
	}
	for _, layer := range runtimeLayers {
		if len(layer.FrequencyCurve) > 0 {
			hashValue, err := BuildCurvePayloadHash(runtimeLayers)
			if err != nil {
				return "", err
			}
			return suffix + "_" + hashValue, nil
		}
	}
	return suffix, nil
}

func BuildOutputFilename(originalFilename string, layers []Layer) (string, error) {
	suffix, err := BuildOutputSuffix(layers)
	if err != nil {
		return "", err
	}
	return originalFilename + "_" + suffix + ".wav", nil
}

func EvaluateFrequencyCurveGainDB(curvePoints []FrequencyCurvePoint, frequencyHz float64) float64 {
	if len(curvePoints) == 0 {
		return 0.0
	}
	if frequencyHz <= curvePoints[0].FrequencyHz {
		return curvePoints[0].GainDB
	}
	if frequencyHz >= curvePoints[len(curvePoints)-1].FrequencyHz {
		return curvePoints[len(curvePoints)-1].GainDB
	}
	if len(curvePoints) == 1 {
		return curvePoints[0].GainDB
	}

	logFrequency := math.Log(frequencyHz)
	for index := 0; index < len(curvePoints)-1; index++ {
		leftPoint := curvePoints[index]
		rightPoint := curvePoints[index+1]
		if leftPoint.FrequencyHz <= frequencyHz && frequencyHz <= rightPoint.FrequencyHz {
			leftLogFrequency := math.Log(leftPoint.FrequencyHz)
			rightLogFrequency := math.Log(rightPoint.FrequencyHz)
			interpolation := (logFrequency - leftLogFrequency) / (rightLogFrequency - leftLogFrequency)
			return leftPoint.GainDB + interpolation*(rightPoint.GainDB-leftPoint.GainDB)
		}
	}

	return curvePoints[len(curvePoints)-1].GainDB
}

func defaultLayer() Layer {
	return Layer{
		Type:           "pulse",
		Duty:           0.5,
		Volume:         1.0,
		FrequencyCurve: []FrequencyCurvePoint{},
	}
}

func validateLayerCount(count int) error {
	if count > MaxRenderLayers {
		return RenderLimitError{
			Message: fmt.Sprintf("Synthesis supports at most %d sound layers.", MaxRenderLayers),
		}
	}
	return nil
}

func parseLayerNumber(rawLayer map[string]any, layerIndex int, fieldName string, defaultValue float64) (float64, error) {
	rawValue, ok := rawLayer[fieldName]
	if !ok || rawValue == nil {
		return defaultValue, nil
	}
	value, err := parseFiniteNumber(rawValue, fmt.Sprintf("Layer %d %s", layerIndex, fieldName))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseLayerInteger(rawLayer map[string]any, layerIndex int, fieldName string, defaultValue int) (int, error) {
	rawValue, ok := rawLayer[fieldName]
	if !ok || rawValue == nil {
		return defaultValue, nil
	}
	value, err := parseFiniteNumber(rawValue, fmt.Sprintf("Layer %d %s", layerIndex, fieldName))
	if err != nil {
		return 0, err
	}
	intValue := int(value)
	if value != float64(intValue) {
		return 0, fmt.Errorf("Layer %d %s must be an integer.", layerIndex, fieldName)
	}
	return intValue, nil
}

func parseFiniteNumber(rawValue any, fieldLabel string) (float64, error) {
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

func sanitiseChannelList(rawValue any, layerIndex int) ([]int, error) {
	if rawValue == nil {
		return nil, nil
	}
	rawItems, ok := rawValue.([]any)
	if !ok {
		if typedChannels, ok := rawValue.([]int); ok {
			return uniqueSortedChannels(typedChannels, layerIndex)
		}
		return nil, fmt.Errorf("Layer %d midi_channels must be an array.", layerIndex)
	}
	channels := make([]int, 0, len(rawItems))
	for _, rawItem := range rawItems {
		channelFloat, err := parseFiniteNumber(rawItem, fmt.Sprintf("Layer %d midi_channels", layerIndex))
		if err != nil {
			return nil, err
		}
		channel := int(channelFloat)
		if channelFloat != float64(channel) {
			return nil, fmt.Errorf("Layer %d midi_channels values must be integers.", layerIndex)
		}
		channels = append(channels, channel)
	}
	return uniqueSortedChannels(channels, layerIndex)
}

func uniqueSortedChannels(channels []int, layerIndex int) ([]int, error) {
	seen := map[int]struct{}{}
	result := make([]int, 0, len(channels))
	for _, channel := range channels {
		if channel < 1 || channel > 16 {
			return nil, fmt.Errorf("Layer %d midi_channels values must be between 1 and 16.", layerIndex)
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		result = append(result, channel)
	}
	sort.Ints(result)
	return result, nil
}

func sanitiseFrequencyCurve(rawCurve any, layerIndex int) ([]FrequencyCurvePoint, error) {
	if rawCurve == nil {
		return []FrequencyCurvePoint{}, nil
	}
	curveItems, ok := rawCurve.([]any)
	if !ok {
		return nil, fmt.Errorf("Layer %d frequency_curve must be an array.", layerIndex)
	}
	if len(curveItems) == 0 {
		return []FrequencyCurvePoint{}, nil
	}
	if len(curveItems) > MaxCurvePoints {
		return nil, fmt.Errorf("Layer %d frequency_curve supports at most %d points.", layerIndex, MaxCurvePoints)
	}

	points := make([]FrequencyCurvePoint, 0, len(curveItems))
	for pointIndex, rawPoint := range curveItems {
		point, ok := rawPoint.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Layer %d frequency_curve point %d must be an object.", layerIndex, pointIndex+1)
		}

		frequencyHz, err := parseFiniteNumber(
			point["frequency_hz"],
			fmt.Sprintf("Layer %d frequency_curve point %d frequency_hz", layerIndex, pointIndex+1),
		)
		if err != nil {
			return nil, err
		}
		frequencyHz = clampFrequencyBoundary(frequencyHz)
		if frequencyHz < MinCurveFrequencyHz || frequencyHz > MaxCurveFrequencyHz {
			return nil, fmt.Errorf(
				"Layer %d frequency_curve point %d frequency_hz must be between %.6f and %.6f.",
				layerIndex,
				pointIndex+1,
				MinCurveFrequencyHz,
				MaxCurveFrequencyHz,
			)
		}

		gainDB, err := parseFiniteNumber(
			point["gain_db"],
			fmt.Sprintf("Layer %d frequency_curve point %d gain_db", layerIndex, pointIndex+1),
		)
		if err != nil {
			return nil, err
		}
		if gainDB < MinCurveGainDB || gainDB > MaxCurveGainDB {
			return nil, fmt.Errorf(
				"Layer %d frequency_curve point %d gain_db must be between %.1f and %.1f.",
				layerIndex,
				pointIndex+1,
				MinCurveGainDB,
				MaxCurveGainDB,
			)
		}

		points = append(points, FrequencyCurvePoint{
			FrequencyHz: frequencyHz,
			GainDB:      gainDB,
		})
	}

	sort.Slice(points, func(left, right int) bool {
		return points[left].FrequencyHz < points[right].FrequencyHz
	})
	for index := 1; index < len(points); index++ {
		if points[index].FrequencyHz <= points[index-1].FrequencyHz {
			return nil, fmt.Errorf("Layer %d frequency_curve frequencies must be strictly increasing.", layerIndex)
		}
	}
	return points, nil
}

func clampFrequencyBoundary(frequencyHz float64) float64 {
	if frequencyHz < MinCurveFrequencyHz && math.Abs(frequencyHz-MinCurveFrequencyHz) <= CurveFrequencyToleranceHz {
		return MinCurveFrequencyHz
	}
	if frequencyHz > MaxCurveFrequencyHz && math.Abs(frequencyHz-MaxCurveFrequencyHz) <= CurveFrequencyToleranceHz {
		return MaxCurveFrequencyHz
	}
	return frequencyHz
}

func curveAsRaw(curve []FrequencyCurvePoint) []any {
	rawCurve := make([]any, 0, len(curve))
	for _, point := range curve {
		rawCurve = append(rawCurve, map[string]any{
			"frequency_hz": point.FrequencyHz,
			"gain_db":      point.GainDB,
		})
	}
	return rawCurve
}

func canonicalLayersJSON(layers []Layer) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for index, layer := range layers {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(`{"duty":`)
		builder.WriteString(formatPythonFloat(layer.Duty))
		builder.WriteString(`,"frequency_curve":[`)
		for pointIndex, point := range layer.FrequencyCurve {
			if pointIndex > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(`{"frequency_hz":`)
			builder.WriteString(formatPythonFloat(point.FrequencyHz))
			builder.WriteString(`,"gain_db":`)
			builder.WriteString(formatPythonFloat(point.GainDB))
			builder.WriteByte('}')
		}
		builder.WriteString(`],"type":`)
		typeJSON, _ := json.Marshal(layer.Type)
		builder.Write(typeJSON)
		builder.WriteString(`,"volume":`)
		builder.WriteString(formatPythonFloat(layer.Volume))
		builder.WriteByte('}')
	}
	builder.WriteByte(']')
	return builder.String()
}

func formatPythonFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.ContainsAny(formatted, ".eE") {
		formatted += ".0"
	}
	return formatted
}

func jsonErrorMessage(err error) string {
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return "Expecting value"
	}
	return err.Error()
}
