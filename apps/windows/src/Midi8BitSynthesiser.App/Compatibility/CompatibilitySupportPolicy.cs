using System.Reflection;

namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed record CompatibilitySupportPolicy(
    Version MinimumWindowsVersion,
    string RuntimeIdentifier,
    string DeploymentModel,
    string AppVersion)
{
    public string SupportedArchitecture => RuntimeIdentifier.Split('-').LastOrDefault() ?? "x64";

    public int MinimumWindowsBuild => MinimumWindowsVersion.Build;

    public string MinimumWindowsVersionDisplay => MinimumWindowsVersion.ToString();

    public static CompatibilitySupportPolicy FromAppMetadata()
    {
        var assembly = typeof(App).Assembly;
        return new CompatibilitySupportPolicy(
            ParseVersion(ReadMetadata(assembly, "CompatibilityMinimumWindowsVersion", "10.0.17763.0")),
            ReadMetadata(assembly, "CompatibilityRuntimeIdentifier", "win-x64"),
            ReadMetadata(assembly, "CompatibilityDeploymentModel", "WinUI 3 self-contained desktop app"),
            ReadMetadata(assembly, "CompatibilityVersion", "1.0.0"));
    }

    private static string ReadMetadata(Assembly assembly, string key, string fallback)
    {
        return assembly.GetCustomAttributes<AssemblyMetadataAttribute>()
            .FirstOrDefault(attribute => attribute.Key == key)
            ?.Value
            ?? fallback;
    }

    private static Version ParseVersion(string raw)
        => Version.TryParse(raw, out var version) ? version : new Version(10, 0, 17763, 0);
}
