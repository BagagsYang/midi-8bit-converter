using Midi8BitSynthesiser.Core;
using Midi8BitSynthesiser.Tests.TestData;

namespace Midi8BitSynthesiser.Tests;

public sealed class SynthesisRenderEngineTests
{
    private readonly SynthesisRenderEngine _engine = new();

    [Fact]
    public async Task RenderAsync_WritesEmptyWaveFile_ForEmptyMidi()
    {
        using var temp = new TempDirectory();
        var midiPath = MidiTestFileBuilder.CreateMidi(Path.Combine(temp.Path, "empty.mid"), []);
        var outputPath = Path.Combine(temp.Path, "empty.wav");

        await _engine.RenderAsync(new RenderRequest(midiPath, outputPath, 48_000, []), CancellationToken.None);

        var wave = WaveFileAssertions.ReadWaveFile(outputPath);
        Assert.Equal(48_000, wave.SampleRate);
        Assert.Equal(1, wave.Channels);
        Assert.Equal(16, wave.BitsPerSample);
        Assert.Empty(wave.Samples);
    }

    [Fact]
    public async Task RenderAsync_IgnoresDrumNotes()
    {
        using var temp = new TempDirectory();
        var midiPath = MidiTestFileBuilder.CreateMidi(
            Path.Combine(temp.Path, "drums.mid"),
            [new MidiTestFileBuilder.MidiNoteSpec(36, 100, 0, 480, 9)]);
        var outputPath = Path.Combine(temp.Path, "drums.wav");

        await _engine.RenderAsync(new RenderRequest(
            midiPath,
            outputPath,
            48_000,
            [new WaveLayer(WaveType.Pulse, 0.5, 1.0)]), CancellationToken.None);

        var wave = WaveFileAssertions.ReadWaveFile(outputPath);
        Assert.NotEmpty(wave.Samples);
        Assert.Equal(16, wave.BitsPerSample);
        Assert.All(wave.Samples, sample => Assert.Equal<short>(0, sample));
    }

    [Fact]
    public async Task RenderAsync_WritesAudio_ForPitchedNotes()
    {
        using var temp = new TempDirectory();
        var midiPath = MidiTestFileBuilder.CreateSingleNoteMidi(Path.Combine(temp.Path, "lead.mid"));
        var outputPath = Path.Combine(temp.Path, "lead.wav");

        var result = await _engine.RenderAsync(new RenderRequest(
            midiPath,
            outputPath,
            44_100,
            [new WaveLayer(WaveType.Triangle, 0.5, 1.0)]), CancellationToken.None);

        var wave = WaveFileAssertions.ReadWaveFile(outputPath);
        Assert.True(result.Duration > TimeSpan.Zero);
        Assert.Equal(44_100, wave.SampleRate);
        Assert.Equal(1, wave.Channels);
        Assert.Equal(16, wave.BitsPerSample);
        Assert.Contains(wave.Samples, sample => sample != 0);
    }

    [Fact]
    public async Task RenderAsync_RendersOnlyPitchedNotes_WhenDrumsAndPitchedNotesAreMixed()
    {
        using var temp = new TempDirectory();
        var midiPath = MidiTestFileBuilder.CreateMidi(
            Path.Combine(temp.Path, "mixed.mid"),
            [
                new MidiTestFileBuilder.MidiNoteSpec(36, 100, 0, 480, 9),
                new MidiTestFileBuilder.MidiNoteSpec(60, 100, 0, 480, 0),
            ]);
        var outputPath = Path.Combine(temp.Path, "mixed.wav");

        await _engine.RenderAsync(new RenderRequest(
            midiPath,
            outputPath,
            48_000,
            [new WaveLayer(WaveType.Pulse, 0.5, 1.0)]), CancellationToken.None);

        var wave = WaveFileAssertions.ReadWaveFile(outputPath);
        Assert.NotEmpty(wave.Samples);
        Assert.Contains(wave.Samples, sample => sample != 0);
    }
}
