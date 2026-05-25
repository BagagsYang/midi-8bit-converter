namespace Midi8BitSynthesiser.App.Compatibility;

public interface ICompatibilityProbe
{
    CompatibilityReport EvaluateStartup();

    CompatibilityReport EvaluateOutputDirectory(string? outputDirectory);
}
