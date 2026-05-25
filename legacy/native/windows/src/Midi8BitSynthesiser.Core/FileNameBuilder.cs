namespace Midi8BitSynthesiser.Core;

public static class FileNameBuilder
{
    public static string BuildOutputPath(string inputPath, string outputDirectory, IReadOnlyList<WaveLayer> layers)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(inputPath);
        ArgumentException.ThrowIfNullOrWhiteSpace(outputDirectory);

        var sanitizedLayers = LayerSanitizer.Sanitize(layers);
        var suffix = sanitizedLayers.Count > 1
            ? "mix"
            : sanitizedLayers[0].Type.ToString().ToLowerInvariant();
        var filename = Path.GetFileNameWithoutExtension(inputPath);

        return Path.Combine(outputDirectory, $"{filename}_{suffix}.wav");
    }
}
