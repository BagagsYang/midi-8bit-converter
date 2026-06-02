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
	return renderNotesWAV(notes, sampleRate, layers, RenderOptions{
		Output: OutputOptions{NormaliseEnabled: true},
	}, false)
}

func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		Output: OutputOptions{
			MasterGainDB:     0,
			LimiterEnabled:   true,
			NormaliseEnabled: false,
		},
	}
}

func RenderNotesWAVWithOptions(notes []Note, sampleRate int, layers []Layer, options RenderOptions) ([]byte, error) {
	return renderNotesWAV(notes, sampleRate, layers, options, true)
}

func renderNotesWAV(notes []Note, sampleRate int, layers []Layer, options RenderOptions, usePolyBLEP bool) ([]byte, error) {
	runtimeLayers, err := NormaliseRuntimeLayers(layers)
	if err != nil {
		return nil, err
	}
	renderOptions, err := normaliseRenderOptions(options)
	if err != nil {
		return nil, err
	}
	channelRoutingActive := hasChannelRouting(runtimeLayers)
	totalSamples, err := validateNoteRenderLimits(notes, sampleRate, runtimeLayers)
	if err != nil {
		return nil, err
	}

	channelBuffers := map[int][]float32{}
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
			if !layerAllowsChannel(layer, note.Channel, channelRoutingActive) {
				continue
			}
			layerFrequency := frequency * math.Pow(2, float64(layer.OctaveShift)) * math.Pow(2, layer.DetuneCents/1200.0)
			curveGainDB := EvaluateFrequencyCurveGainDB(layer.FrequencyCurve, layerFrequency)
			effectiveVolume := float32(layer.Volume * dbToLinearGain(curveGainDB))
			if effectiveVolume <= 0 {
				continue
			}
			layerWave := generateWaveformWithQuality(layerFrequency, sampleRate, noteSampleLength, layer, usePolyBLEP)
			for index := range mixedNoteWaveform {
				mixedNoteWaveform[index] += layerWave[index] * effectiveVolume
			}
		}

		applyEnvelope(mixedNoteWaveform, sampleRate)
		channelBuffer := channelBuffers[note.Channel]
		if channelBuffer == nil {
			channelBuffer = make([]float32, totalSamples)
			channelBuffers[note.Channel] = channelBuffer
		}
		endSample = startSample + len(mixedNoteWaveform)
		if endSample > len(channelBuffer) {
			actualLength := len(channelBuffer) - startSample
			if actualLength <= 0 {
				continue
			}
			for index := 0; index < actualLength; index++ {
				channelBuffer[startSample+index] += mixedNoteWaveform[index] * noteVolume
			}
			continue
		}
		for index := range mixedNoteWaveform {
			channelBuffer[startSample+index] += mixedNoteWaveform[index] * noteVolume
		}
	}

	audioBuffer := mixChannelBuffers(channelBuffers, totalSamples, renderOptions.ChannelBuses)
	applyOutputProcessing(audioBuffer, renderOptions.Output)
	samples := make([]int16, len(audioBuffer))
	for index, sample := range audioBuffer {
		samples[index] = int16(sample)
	}
	return encodePCM16WAV(sampleRate, samples), nil
}

func validateNoteRenderLimits(notes []Note, sampleRate int, layers []Layer) (int, error) {
	if err := validateRuntimeLayerCount(len(layers)); err != nil {
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

func hasChannelRouting(layers []Layer) bool {
	for _, layer := range layers {
		if len(layer.MIDIChannels) > 0 {
			return true
		}
	}
	return false
}

func layerAllowsChannel(layer Layer, noteChannel int, channelRoutingActive bool) bool {
	if !channelRoutingActive {
		return true
	}
	if len(layer.MIDIChannels) == 0 || noteChannel == 0 {
		return false
	}
	for _, channel := range layer.MIDIChannels {
		if channel == noteChannel {
			return true
		}
	}
	return false
}

func generateWaveform(frequency float64, sampleRate int, sampleCount int, layer Layer) []float32 {
	return generateWaveformWithQuality(frequency, sampleRate, sampleCount, layer, true)
}

func generateWaveformWithQuality(frequency float64, sampleRate int, sampleCount int, layer Layer, usePolyBLEP bool) []float32 {
	waveform := make([]float32, sampleCount)
	if layer.VibratoDepthCents <= 0 && layer.Type != "noise" {
		return generateStaticWaveformWithQuality(frequency, sampleRate, sampleCount, layer.Type, layer.Duty, usePolyBLEP)
	}
	phase := 0.0
	noiseValue := float32(1.0)
	lfsr := uint16(0xACE1)
	for index := range waveform {
		t := float64(index) / float64(sampleRate)
		frequencyMultiplier := 1.0
		if layer.VibratoDepthCents > 0 {
			frequencyMultiplier = math.Pow(2, (layer.VibratoDepthCents/1200.0)*math.Sin(2*math.Pi*layer.VibratoRateHz*t))
		}
		phaseIncrement := frequency * frequencyMultiplier / float64(sampleRate)
		waveform[index] = oscillatorSample(layer.Type, phase, phaseIncrement, layer.Duty, usePolyBLEP)
		phase += phaseIncrement
		for phase >= 1.0 {
			phase -= 1.0
			if layer.Type == "noise" {
				lfsr = stepLFSR16(lfsr)
				if lfsr&1 == 1 {
					noiseValue = 1.0
				} else {
					noiseValue = -1.0
				}
			}
		}
		if layer.Type == "noise" {
			waveform[index] = noiseValue
		}
	}
	return waveform
}

func generateStaticWaveform(frequency float64, sampleRate int, sampleCount int, waveType string, dutyCycle float64) []float32 {
	return generateStaticWaveformWithQuality(frequency, sampleRate, sampleCount, waveType, dutyCycle, true)
}

func generateStaticWaveformWithQuality(frequency float64, sampleRate int, sampleCount int, waveType string, dutyCycle float64, usePolyBLEP bool) []float32 {
	waveform := make([]float32, sampleCount)
	floatSampleRate := float32(sampleRate)
	floatFrequency := float32(frequency)
	phaseIncrement := frequency / float64(sampleRate)
	for index := range waveform {
		t := float32(index) / floatSampleRate
		phase := math.Mod(float64(t*floatFrequency), 1.0)
		cyclePosition := float32(phase)
		switch waveType {
		case "sine":
			waveform[index] = float32(math.Sin(float64(2 * math.Pi * floatFrequency * t)))
		case "sawtooth":
			waveform[index] = oscillatorSample(waveType, phase, phaseIncrement, dutyCycle, usePolyBLEP)
		case "triangle":
			waveform[index] = 2.0*float32(math.Abs(float64(2.0*cyclePosition-1.0))) - 1.0
		case "pulse":
			waveform[index] = oscillatorSample(waveType, phase, phaseIncrement, dutyCycle, usePolyBLEP)
		default:
			waveform[index] = 0
		}
	}
	return waveform
}

func oscillatorSample(waveType string, phase float64, phaseIncrement float64, dutyCycle float64, usePolyBLEP bool) float32 {
	switch waveType {
	case "sine":
		return float32(math.Sin(2 * math.Pi * phase))
	case "sawtooth":
		if !usePolyBLEP {
			return float32(2.0*phase - 1.0)
		}
		return float32(2.0*phase - 1.0 - polyBLEP(phase, phaseIncrement))
	case "triangle":
		return 2.0*float32(math.Abs(2.0*phase-1.0)) - 1.0
	case "pulse":
		value := 1.0
		if phase >= dutyCycle {
			value = -1.0
		}
		if !usePolyBLEP {
			return float32(value)
		}
		value += polyBLEP(phase, phaseIncrement)
		value -= polyBLEP(math.Mod(phase-dutyCycle+1.0, 1.0), phaseIncrement)
		return float32(value)
	default:
		return 0
	}
}

func polyBLEP(phase float64, phaseIncrement float64) float64 {
	if phaseIncrement <= 0 {
		return 0
	}
	if phase < phaseIncrement {
		t := phase / phaseIncrement
		return t + t - t*t - 1.0
	}
	if phase > 1.0-phaseIncrement {
		t := (phase - 1.0) / phaseIncrement
		return t*t + t + t + 1.0
	}
	return 0
}

func stepLFSR16(value uint16) uint16 {
	bit := ((value >> 0) ^ (value >> 2) ^ (value >> 3) ^ (value >> 5)) & 1
	return (value >> 1) | (bit << 15)
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

func mixChannelBuffers(channelBuffers map[int][]float32, totalSamples int, buses []ChannelBus) []float32 {
	audioBuffer := make([]float32, totalSamples)
	if len(channelBuffers) == 0 {
		return audioBuffer
	}

	busByChannel := map[int]ChannelBus{}
	soloActive := false
	for _, bus := range buses {
		busByChannel[bus.Channel] = bus
		if bus.Solo {
			soloActive = true
		}
	}

	for channel, channelBuffer := range channelBuffers {
		bus, hasBus := busByChannel[channel]
		if soloActive {
			if !hasBus || !bus.Solo {
				continue
			}
		} else if hasBus && bus.Mute {
			continue
		}

		volume := float32(1.0)
		if hasBus {
			volume = float32(bus.Volume)
		}
		for index, sample := range channelBuffer {
			audioBuffer[index] += sample * volume
		}
	}
	return audioBuffer
}

func applyOutputProcessing(audioBuffer []float32, options OutputOptions) {
	if len(audioBuffer) == 0 {
		return
	}
	masterGain := float32(dbToLinearGain(options.MasterGainDB))
	for index := range audioBuffer {
		audioBuffer[index] *= masterGain
	}
	if options.NormaliseEnabled {
		normaliseAndScale(audioBuffer)
		return
	}
	if options.LimiterEnabled {
		for index, sample := range audioBuffer {
			audioBuffer[index] = float32(math.Tanh(float64(sample)))
		}
	}
	for index := range audioBuffer {
		audioBuffer[index] = clampPCMFloat(audioBuffer[index]) * audioWorkingScale
	}
}

func clampPCMFloat(value float32) float32 {
	if value > 1.0 {
		return 1.0
	}
	if value < -1.0 {
		return -1.0
	}
	return value
}

func normaliseRenderOptions(options RenderOptions) (RenderOptions, error) {
	output := options.Output
	if output.MasterGainDB < MinMasterGainDB || output.MasterGainDB > MaxMasterGainDB {
		return RenderOptions{}, RenderLimitError{
			Message: "Master gain must be between -24.0 and 12.0 dB.",
		}
	}

	buses := make([]ChannelBus, 0, len(options.ChannelBuses))
	seen := map[int]struct{}{}
	for _, bus := range options.ChannelBuses {
		if bus.Channel < 1 || bus.Channel > 16 {
			return RenderOptions{}, RenderLimitError{
				Message: "Channel bus values must be between 1 and 16.",
			}
		}
		if bus.Volume < MinBusVolume || bus.Volume > MaxBusVolume {
			return RenderOptions{}, RenderLimitError{
				Message: "Channel bus volume must be between 0.0 and 2.0.",
			}
		}
		if _, exists := seen[bus.Channel]; exists {
			continue
		}
		seen[bus.Channel] = struct{}{}
		buses = append(buses, bus)
	}
	return RenderOptions{ChannelBuses: buses, Output: output}, nil
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
