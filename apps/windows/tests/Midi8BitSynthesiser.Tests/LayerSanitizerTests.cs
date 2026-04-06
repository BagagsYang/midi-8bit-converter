using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.Tests;

public sealed class LayerSanitizerTests
{
    [Fact]
    public void Sanitize_UsesDefaultLayer_WhenLayersAreMissingOrMuted()
    {
        var mutedLayers = new[]
        {
            new WaveLayer(WaveType.Sine, 0.5, 0.0),
            new WaveLayer(WaveType.Triangle, 0.5, 0.0),
        };

        var fromNull = LayerSanitizer.Sanitize(null);
        var fromMuted = LayerSanitizer.Sanitize(mutedLayers);

        Assert.Single(fromNull);
        Assert.Equal(Defaults.DefaultLayer, fromNull[0]);
        Assert.Single(fromMuted);
        Assert.Equal(Defaults.DefaultLayer, fromMuted[0]);
    }

    [Theory]
    [InlineData(0.0)]
    [InlineData(1.0)]
    public void Validate_RejectsDutyOutsideSupportedRange(double duty)
    {
        var layer = new WaveLayer(WaveType.Pulse, duty, 1.0);

        Assert.Throws<ArgumentOutOfRangeException>(() => LayerSanitizer.Validate(layer));
    }

    [Fact]
    public void Validate_RejectsNegativeVolume()
    {
        var layer = new WaveLayer(WaveType.Pulse, 0.5, -0.1);

        Assert.Throws<ArgumentOutOfRangeException>(() => LayerSanitizer.Validate(layer));
    }
}
