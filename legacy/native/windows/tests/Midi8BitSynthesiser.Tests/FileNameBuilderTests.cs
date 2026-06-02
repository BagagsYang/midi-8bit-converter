using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.Tests;

public sealed class FileNameBuilderTests
{
    [Fact]
    public void BuildOutputPath_UsesWaveName_ForSingleAudibleLayer()
    {
        var output = FileNameBuilder.BuildOutputPath(
            @"C:\input\lead.mid",
            @"C:\output",
            [new WaveLayer(WaveType.Sawtooth, 0.5, 1.0)]);

        Assert.Equal(@"C:\output\lead_sawtooth.wav", output);
    }

    [Fact]
    public void BuildOutputPath_UsesMixSuffix_ForMultipleAudibleLayers()
    {
        var output = FileNameBuilder.BuildOutputPath(
            @"C:\input\lead.mid",
            @"C:\output",
            [new WaveLayer(WaveType.Pulse, 0.5, 1.0), new WaveLayer(WaveType.Sine, 0.5, 0.3)]);

        Assert.Equal(@"C:\output\lead_mix.wav", output);
    }
}
