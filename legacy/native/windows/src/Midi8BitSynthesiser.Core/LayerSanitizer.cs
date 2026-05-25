namespace Midi8BitSynthesiser.Core;

public static class LayerSanitizer
{
    public static IReadOnlyList<WaveLayer> Sanitize(IEnumerable<WaveLayer>? layers)
    {
        if (layers is null)
        {
            return [Defaults.DefaultLayer];
        }

        var sanitized = new List<WaveLayer>();
        foreach (var layer in layers)
        {
            Validate(layer);
            if (layer.Volume <= 0)
            {
                continue;
            }

            sanitized.Add(layer);
        }

        return sanitized.Count > 0 ? sanitized : [Defaults.DefaultLayer];
    }

    public static void Validate(WaveLayer layer)
    {
        if (layer.Duty is < 0.01 or > 0.99)
        {
            throw new ArgumentOutOfRangeException(nameof(layer), "Duty must be between 0.01 and 0.99.");
        }

        if (layer.Volume < 0)
        {
            throw new ArgumentOutOfRangeException(nameof(layer), "Volume must be 0 or greater.");
        }
    }
}
