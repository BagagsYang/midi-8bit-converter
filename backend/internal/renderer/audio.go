package renderer

import (
	"bytes"
	"encoding/binary"
	"math"
)

const (
	audioWorkingScale = 32767.0
	attackSeconds     = 0.005
	releaseSeconds    = 0.005
)

type Note struct {
	Pitch    int     `json:"pitch"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Velocity int     `json:"velocity"`
	Channel  int     `json:"channel,omitempty"`
}

func RenderNotesWAV(notes []Note, sampleRate int, layers []Layer) ([]byte, error) {
	runtimeLayers, err := NormaliseRuntimeLayers(layers)
	if err != nil {
		return nil, err
	}
	totalSamples, err := validateNoteRenderLimits(notes, sampleRate, runtimeLayers)
	if err != nil {
		return nil, err
	}

	audioBuffer := make([]float32, totalSamples)
	for _, note := range notes {
		startSample := int(note.Start * float64(sampleRate))
		endSample := int(math.Ceil(note.End * float64(sampleRate)))
		noteSampleLength := endSample - startSample
		if noteSampleLength <= 0 {
			continue
		}

		frequency := noteNumberToHz(note.Pitch)
		noteVolume := float32(float64(note.Velocity) / 127.0)
		mixedNoteWaveform := make([]float32, noteSampleLength)

		for _, layer := range runtimeLayers {
			curveGainDB := EvaluateFrequencyCurveGainDB(layer.FrequencyCurve, frequency)
			effectiveVolume := float32(layer.Volume * dbToLinearGain(curveGainDB))
			if effectiveVolume <= 0 {
				continue
			}
			layerWave := generateWaveform(frequency, sampleRate, noteSampleLength, layer.Type, layer.Duty)
			for index := range mixedNoteWaveform {
				mixedNoteWaveform[index] += layerWave[index] * effectiveVolume
			}
		}

		applyEnvelope(mixedNoteWaveform, sampleRate)
		endSample = startSample + len(mixedNoteWaveform)
		if endSample > len(audioBuffer) {
			actualLength := len(audioBuffer) - startSample
			if actualLength <= 0 {
				continue
			}
			for index := 0; index < actualLength; index++ {
				audioBuffer[startSample+index] += mixedNoteWaveform[index] * noteVolume
			}
			continue
		}
		for index := range mixedNoteWaveform {
			audioBuffer[startSample+index] += mixedNoteWaveform[index] * noteVolume
		}
	}

	normaliseAndScale(audioBuffer)
	samples := make([]int16, len(audioBuffer))
	for index, sample := range audioBuffer {
		samples[index] = int16(sample)
	}
	return encodePCM16WAV(sampleRate, samples), nil
}

func validateNoteRenderLimits(notes []Note, sampleRate int, layers []Layer) (int, error) {
	if err := validateLayerCount(len(layers)); err != nil {
		return 0, err
	}
	if len(notes) > MaxMIDINotes {
		return 0, RenderLimitError{
			Message: "MIDI contains too many notes; the limit is 20000 notes.",
		}
	}

	totalTime := 0.0
	for _, note := range notes {
		if note.End > totalTime {
			totalTime = note.End
		}
	}
	if totalTime > MaxRenderSeconds {
		return 0, RenderLimitError{
			Message: "MIDI duration must be 1800 seconds or less.",
		}
	}

	totalSamples := int(math.Ceil(totalTime * float64(sampleRate)))
	if totalSamples > MaxRenderSeconds*96000 {
		return 0, RenderLimitError{
			Message: "Rendered audio is too large; use a shorter MIDI file or lower sample rate.",
		}
	}
	outputBytes := totalSamples * 2
	if outputBytes > MaxRenderSeconds*96000*2 {
		return 0, RenderLimitError{
			Message: "Rendered WAV output is too large; use a shorter MIDI file or lower sample rate.",
		}
	}
	for _, note := range notes {
		noteSamples := int(math.Ceil(note.End*float64(sampleRate))) - int(note.Start*float64(sampleRate))
		if noteSamples > MaxRenderSeconds*96000 {
			return 0, RenderLimitError{
				Message: "A MIDI note is too long to render safely.",
			}
		}
	}
	return totalSamples, nil
}

func generateWaveform(frequency float64, sampleRate int, sampleCount int, waveType string, dutyCycle float64) []float32 {
	waveform := make([]float32, sampleCount)
	floatSampleRate := float32(sampleRate)
	floatFrequency := float32(frequency)
	for index := range waveform {
		t := float32(index) / floatSampleRate
		cyclePosition := float32(math.Mod(float64(t*floatFrequency), 1.0))
		switch waveType {
		case "sine":
			waveform[index] = float32(math.Sin(float64(2 * math.Pi * floatFrequency * t)))
		case "sawtooth":
			waveform[index] = 2.0*cyclePosition - 1.0
		case "triangle":
			waveform[index] = 2.0*float32(math.Abs(float64(2.0*cyclePosition-1.0))) - 1.0
		case "pulse":
			if cyclePosition < float32(dutyCycle) {
				waveform[index] = 1.0
			} else {
				waveform[index] = -1.0
			}
		default:
			waveform[index] = 0
		}
	}
	return waveform
}

func applyEnvelope(waveform []float32, sampleRate int) {
	if len(waveform) == 0 {
		return
	}
	attackSamples := min(int(attackSeconds*float64(sampleRate)), len(waveform)/2)
	releaseSamples := min(int(releaseSeconds*float64(sampleRate)), len(waveform)-attackSamples)
	if attackSamples > 0 {
		for index, value := range linspace(0, 1, attackSamples) {
			waveform[index] *= value
		}
	}
	if releaseSamples > 0 {
		start := len(waveform) - releaseSamples
		for index, value := range linspace(1, 0, releaseSamples) {
			waveform[start+index] *= value
		}
	}
}

func linspace(start, stop float32, count int) []float32 {
	values := make([]float32, count)
	if count == 0 {
		return values
	}
	if count == 1 {
		values[0] = start
		return values
	}
	step := (stop - start) / float32(count-1)
	for index := range values {
		values[index] = start + float32(index)*step
	}
	return values
}

func normaliseAndScale(audioBuffer []float32) {
	if len(audioBuffer) == 0 {
		return
	}
	minValue := audioBuffer[0]
	maxValue := audioBuffer[0]
	for _, value := range audioBuffer[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	maxAbs := math.Max(math.Abs(float64(minValue)), math.Abs(float64(maxValue)))
	if maxAbs > 0 {
		gain := float32(NormalisedPeak / maxAbs)
		for index := range audioBuffer {
			audioBuffer[index] *= gain
		}
	}
	for index := range audioBuffer {
		audioBuffer[index] *= audioWorkingScale
	}
}

func noteNumberToHz(noteNumber int) float64 {
	return 440.0 * math.Pow(2.0, float64(noteNumber-69)/12.0)
}

func dbToLinearGain(gainDB float64) float64 {
	return math.Pow(10.0, gainDB/20.0)
}

func encodePCM16WAV(sampleRate int, samples []int16) []byte {
	dataBytes := uint32(len(samples) * 2)
	riffSize := uint32(36) + dataBytes
	var buffer bytes.Buffer
	buffer.Grow(44 + len(samples)*2)
	buffer.WriteString("RIFF")
	_ = binary.Write(&buffer, binary.LittleEndian, riffSize)
	buffer.WriteString("WAVE")
	buffer.WriteString("fmt ")
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(16))
	buffer.WriteString("data")
	_ = binary.Write(&buffer, binary.LittleEndian, dataBytes)
	for _, sample := range samples {
		_ = binary.Write(&buffer, binary.LittleEndian, sample)
	}
	return buffer.Bytes()
}
