using System.Text.Json;
using Midi8BitSynthesiser.Core;
using Midi8BitSynthesiser.Tests.TestData;

namespace Midi8BitSynthesiser.Tests;

public sealed class PythonParityTests
{
    [Fact]
    public async Task RenderAsync_MatchesPythonEngineWithinOneLsb()
    {
        using var temp = new TempDirectory();
        var repoRoot = RepoRootLocator.FindRepoRoot();
        var midiPath = MidiTestFileBuilder.CreateMidi(
            Path.Combine(temp.Path, "parity.mid"),
            [
                new MidiTestFileBuilder.MidiNoteSpec(60, 100, 0, 480, 0),
                new MidiTestFileBuilder.MidiNoteSpec(64, 96, 480, 480, 0),
            ]);

        var layers = new[]
        {
            new WaveLayer(WaveType.Pulse, 0.25, 0.8),
            new WaveLayer(WaveType.Triangle, 0.5, 0.4),
        };

        var windowsOutputPath = Path.Combine(temp.Path, "rendered-csharp.wav");
        var pythonOutputPath = Path.Combine(temp.Path, "rendered-python.wav");

        await AssertMatchesPythonAsync(repoRoot, midiPath, windowsOutputPath, pythonOutputPath, 48_000, layers);
    }

    [Theory]
    [InlineData(1_000)]
    [InlineData(2_000)]
    public async Task RenderAsync_MatchesPythonForShortEnvelopeEdgeCases(int sampleRate)
    {
        using var temp = new TempDirectory();
        var repoRoot = RepoRootLocator.FindRepoRoot();
        var midiPath = MidiTestFileBuilder.CreateMidi(
            Path.Combine(temp.Path, $"short-{sampleRate}.mid"),
            [
                new MidiTestFileBuilder.MidiNoteSpec(72, 100, 0, 1, 0),
            ]);
        var windowsOutputPath = Path.Combine(temp.Path, $"short-{sampleRate}-csharp.wav");
        var pythonOutputPath = Path.Combine(temp.Path, $"short-{sampleRate}-python.wav");

        await AssertMatchesPythonAsync(
            repoRoot,
            midiPath,
            windowsOutputPath,
            pythonOutputPath,
            sampleRate,
            [new WaveLayer(WaveType.Pulse, 0.5, 1.0)]);
    }

    private static async Task AssertMatchesPythonAsync(
        string repoRoot,
        string midiPath,
        string windowsOutputPath,
        string pythonOutputPath,
        int sampleRate,
        IReadOnlyList<WaveLayer> layers)
    {
        var renderEngine = new SynthesisRenderEngine();
        await renderEngine.RenderAsync(
            new RenderRequest(midiPath, windowsOutputPath, sampleRate, layers),
            CancellationToken.None);

        using var pythonProcess = PythonLauncher.StartProcess(
            repoRoot,
            RepoRootLocator.FindPythonRendererScriptPath(),
            [
                midiPath,
                pythonOutputPath,
                "--rate",
                sampleRate.ToString(),
                "--layers-json",
                JsonSerializer.Serialize(layers.Select(layer => new
                {
                    type = layer.Type.ToString().ToLowerInvariant(),
                    duty = layer.Duty,
                    volume = layer.Volume,
                })),
            ]);

        var stderr = await pythonProcess.StandardError.ReadToEndAsync();
        var stdout = await pythonProcess.StandardOutput.ReadToEndAsync();
        await pythonProcess.WaitForExitAsync();

        Assert.True(
            pythonProcess.ExitCode == 0,
            $"Python parity render failed with exit code {pythonProcess.ExitCode}.{Environment.NewLine}STDOUT:{Environment.NewLine}{stdout}{Environment.NewLine}STDERR:{Environment.NewLine}{stderr}");

        var csharpWave = WaveFileAssertions.ReadWaveFile(windowsOutputPath);
        var pythonWave = WaveFileAssertions.ReadWaveFile(pythonOutputPath);

        Assert.Equal(pythonWave.SampleRate, csharpWave.SampleRate);
        Assert.Equal(pythonWave.Channels, csharpWave.Channels);
        Assert.Equal(pythonWave.BitsPerSample, csharpWave.BitsPerSample);
        Assert.Equal(pythonWave.Samples.Length, csharpWave.Samples.Length);

        for (var index = 0; index < csharpWave.Samples.Length; index++)
        {
            var delta = Math.Abs(csharpWave.Samples[index] - pythonWave.Samples[index]);
            Assert.True(delta <= 1, $"PCM delta exceeded 1 LSB at sample {index}: {delta}");
        }
    }
}
