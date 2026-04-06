using Midi8BitSynthesiser.Core;
using NAudio.Wave;

namespace Midi8BitSynthesiser.App.Services;

public sealed class PreviewAudioPlayer : IAudioPreviewPlayer
{
    private WaveOutEvent? _output;
    private AudioFileReader? _reader;

    public Task PlayAsync(WaveLayer layer, CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();

        var resourceName = GetPreviewResourceName(layer);
        var previewPath = Path.Combine(AppContext.BaseDirectory, "Assets", "Previews", $"{resourceName}.wav");
        if (!File.Exists(previewPath))
        {
            throw new FileNotFoundException($"Missing bundled preview sample: {resourceName}.wav", previewPath);
        }

        StopCurrentPlayback();

        _reader = new AudioFileReader(previewPath);
        _output = new WaveOutEvent();
        _output.Init(_reader);
        _output.Play();

        return Task.CompletedTask;
    }

    public void Dispose()
    {
        StopCurrentPlayback();
    }

    private void StopCurrentPlayback()
    {
        _output?.Stop();
        _output?.Dispose();
        _reader?.Dispose();
        _output = null;
        _reader = null;
    }

    private static string GetPreviewResourceName(WaveLayer layer)
    {
        return layer.Type switch
        {
            WaveType.Pulse when layer.Duty < 0.18 => "pulse_10",
            WaveType.Pulse when layer.Duty < 0.38 => "pulse_25",
            WaveType.Pulse => "pulse_50",
            WaveType.Sine => "sine",
            WaveType.Sawtooth => "sawtooth",
            WaveType.Triangle => "triangle",
            _ => "pulse_50",
        };
    }
}
