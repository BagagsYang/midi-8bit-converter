using Melanchall.DryWetMidi.Core;
using Melanchall.DryWetMidi.Interaction;
using NAudio.Wave;

namespace Midi8BitSynthesiser.Core;

public sealed class SynthesisRenderEngine : IRenderEngine
{
    private const double AttackSeconds = 0.005;
    private const double ReleaseSeconds = 0.005;
    private const double PeakHeadroom = 0.89;
    private const int DrumChannel = 9;

    public Task<RenderResult> RenderAsync(RenderRequest request, CancellationToken cancellationToken)
    {
        ArgumentNullException.ThrowIfNull(request);
        ArgumentException.ThrowIfNullOrWhiteSpace(request.MidiPath);
        ArgumentException.ThrowIfNullOrWhiteSpace(request.OutputPath);

        if (request.SampleRate <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(request), "Sample rate must be greater than zero.");
        }

        return Task.Run(() => RenderInternal(request, cancellationToken), cancellationToken);
    }

    private static RenderResult RenderInternal(RenderRequest request, CancellationToken cancellationToken)
    {
        var layers = LayerSanitizer.Sanitize(request.Layers);
        var midiFile = MidiFile.Read(request.MidiPath);
        var tempoMap = midiFile.GetTempoMap();
        var notes = midiFile.GetNotes().ToList();
        var totalDurationSeconds = notes.Count == 0
            ? 0.0
            : notes.Max(note => ToSeconds(note.TimeAs<MetricTimeSpan>(tempoMap)) + ToSeconds(note.LengthAs<MetricTimeSpan>(tempoMap)));

        var totalSamples = (int)Math.Ceiling(totalDurationSeconds * request.SampleRate);
        if (totalSamples == 0)
        {
            WriteWaveFile(request.OutputPath, request.SampleRate, Array.Empty<short>());
            return new RenderResult(request.OutputPath, TimeSpan.Zero);
        }

        var audioBuffer = new double[totalSamples];

        foreach (var note in notes)
        {
            cancellationToken.ThrowIfCancellationRequested();

            var startSeconds = ToSeconds(note.TimeAs<MetricTimeSpan>(tempoMap));
            var durationSeconds = ToSeconds(note.LengthAs<MetricTimeSpan>(tempoMap));
            var startSample = (int)(startSeconds * request.SampleRate);
            var noteSamples = (int)(request.SampleRate * durationSeconds);

            if (noteSamples <= 0 || startSample >= audioBuffer.Length)
            {
                continue;
            }

            if ((int)note.Channel == DrumChannel)
            {
                continue;
            }

            var frequency = NoteUtilities.ToFrequency(note.NoteNumber);
            var velocity = note.Velocity / 127.0;
            var mixed = new double[noteSamples];

            for (var sampleIndex = 0; sampleIndex < noteSamples; sampleIndex++)
            {
                var time = sampleIndex / (double)request.SampleRate;
                double sampleValue = 0;

                foreach (var layer in layers)
                {
                    sampleValue += WaveformGenerator.Generate(
                        frequency,
                        time,
                        layer.Type,
                        layer.Duty) * layer.Volume;
                }

                mixed[sampleIndex] = sampleValue;
            }

            ApplyEnvelope(mixed, request.SampleRate);

            var writableSamples = Math.Min(mixed.Length, audioBuffer.Length - startSample);
            for (var sampleIndex = 0; sampleIndex < writableSamples; sampleIndex++)
            {
                audioBuffer[startSample + sampleIndex] += mixed[sampleIndex] * velocity;
            }
        }

        Normalize(audioBuffer);
        var pcm = ConvertToPcm16(audioBuffer);
        WriteWaveFile(request.OutputPath, request.SampleRate, pcm);

        return new RenderResult(request.OutputPath, TimeSpan.FromSeconds(totalDurationSeconds));
    }

    private static void ApplyEnvelope(double[] waveform, int sampleRate)
    {
        if (waveform.Length == 0)
        {
            return;
        }

        var attackSamples = Math.Min((int)(AttackSeconds * sampleRate), waveform.Length / 2);
        var releaseSamples = Math.Min((int)(ReleaseSeconds * sampleRate), waveform.Length - attackSamples);

        // Match the Python reference implementation for single-sample envelopes:
        // np.linspace(0, 1, 1) => [0.0] and np.linspace(1, 0, 1) => [1.0].
        for (var index = 0; index < attackSamples; index++)
        {
            waveform[index] *= attackSamples == 1
                ? 0.0
                : index / (double)(attackSamples - 1);
        }

        for (var index = 0; index < releaseSamples; index++)
        {
            var sampleIndex = waveform.Length - releaseSamples + index;
            waveform[sampleIndex] *= releaseSamples == 1
                ? 1.0
                : 1.0 - (index / (double)(releaseSamples - 1));
        }
    }

    private static void Normalize(double[] audioBuffer)
    {
        var max = 0.0;
        foreach (var sample in audioBuffer)
        {
            var abs = Math.Abs(sample);
            if (abs > max)
            {
                max = abs;
            }
        }

        if (max <= 0)
        {
            return;
        }

        var scale = PeakHeadroom / max;
        for (var index = 0; index < audioBuffer.Length; index++)
        {
            audioBuffer[index] *= scale;
        }
    }

    private static short[] ConvertToPcm16(double[] audioBuffer)
    {
        var pcm = new short[audioBuffer.Length];
        for (var index = 0; index < audioBuffer.Length; index++)
        {
            var clamped = Math.Clamp(audioBuffer[index], -1.0, 1.0);
            pcm[index] = (short)(clamped * short.MaxValue);
        }

        return pcm;
    }

    private static void WriteWaveFile(string outputPath, int sampleRate, short[] samples)
    {
        Directory.CreateDirectory(Path.GetDirectoryName(outputPath)!);

        var waveFormat = new WaveFormat(sampleRate, 16, 1);
        using var writer = new WaveFileWriter(outputPath, waveFormat);
        if (samples.Length > 0)
        {
            writer.WriteSamples(samples, 0, samples.Length);
        }
    }

    private static double ToSeconds(MetricTimeSpan timeSpan) => timeSpan.TotalMicroseconds / 1_000_000d;

    private static class NoteUtilities
    {
        public static double ToFrequency(int noteNumber) => 440.0 * Math.Pow(2.0, (noteNumber - 69) / 12.0);
    }

    private static class WaveformGenerator
    {
        public static double Generate(double frequency, double time, WaveType waveType, double dutyCycle)
        {
            var phase = (time * frequency) % 1.0;

            return waveType switch
            {
                WaveType.Sine => Math.Sin(2 * Math.PI * frequency * time),
                WaveType.Sawtooth => (2.0 * phase) - 1.0,
                WaveType.Triangle => (2.0 * Math.Abs((2.0 * phase) - 1.0)) - 1.0,
                WaveType.Pulse => phase < dutyCycle ? 1.0 : -1.0,
                _ => 0.0,
            };
        }
    }
}
