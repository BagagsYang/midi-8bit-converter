using System.Runtime.InteropServices;

namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed class CompatibilityProbe : ICompatibilityProbe
{
    private static readonly string[] RequiredPreviewAssets =
    [
        "pulse_10.wav",
        "pulse_25.wav",
        "pulse_50.wav",
        "sawtooth.wav",
        "sine.wav",
        "triangle.wav",
    ];

    private readonly CompatibilitySupportPolicy _policy;
    private readonly ICompatibilityEnvironment _environment;

    public CompatibilityProbe()
        : this(CompatibilitySupportPolicy.FromAppMetadata(), new SystemCompatibilityEnvironment())
    {
    }

    internal CompatibilityProbe(CompatibilitySupportPolicy policy, ICompatibilityEnvironment environment)
    {
        _policy = policy;
        _environment = environment;
    }

    public CompatibilityReport EvaluateStartup()
    {
        var issues = new List<CompatibilityIssue>();

        if (_environment.OperatingSystemVersion < _policy.MinimumWindowsVersion)
        {
            issues.Add(new CompatibilityIssue(
                "windows-version",
                "Unsupported Windows version",
                $"This release requires Windows {_policy.MinimumWindowsVersionDisplay} or newer, but the current machine reports {_environment.OperatingSystemVersion}.",
                $"Install the app on Windows {_policy.MinimumWindowsVersionDisplay} or newer.",
                isBlocking: true));
        }

        if (_environment.OperatingSystemArchitecture != Architecture.X64)
        {
            issues.Add(new CompatibilityIssue(
                "architecture",
                "Unsupported processor architecture",
                $"This release is published as {_policy.RuntimeIdentifier} and the current operating system architecture is {_environment.OperatingSystemArchitecture}.",
                "Use a supported x64 Windows machine for this release.",
                isBlocking: true));
        }

        var missingAssets = RequiredPreviewAssets
            .Where(asset => !_environment.FileExists(Path.Combine(_environment.BaseDirectory, "Assets", "Previews", asset)))
            .ToArray();
        if (missingAssets.Length > 0)
        {
            issues.Add(new CompatibilityIssue(
                "missing-assets",
                "Bundled preview assets are missing",
                $"The published app is missing required preview files: {string.Join(", ", missingAssets)}.",
                "Reinstall the app or use a release package that contains the full published output.",
                isBlocking: true));
        }

        if (!_environment.TryWriteToDirectory(_environment.TempDirectory, out var tempWriteError))
        {
            issues.Add(new CompatibilityIssue(
                "temp-write",
                "Temporary export folder is not writable",
                $"The app could not write to the temporary folder '{_environment.TempDirectory}'. {tempWriteError}",
                "Make sure the current Windows user can write to the temp folder before launching the app.",
                isBlocking: true));
        }

        var defaultOutputDirectory = _environment.DefaultOutputDirectory;
        if (!_environment.TryWriteToDirectory(defaultOutputDirectory, out var outputWriteError))
        {
            issues.Add(new CompatibilityIssue(
                "default-output",
                "Default output folder is not writable",
                $"The default output location '{defaultOutputDirectory}' is not writable. {outputWriteError}",
                "Choose another writable export folder when you export WAV files.",
                isBlocking: false));
        }

        return CompatibilityReport.Create(issues);
    }

    public CompatibilityReport EvaluateOutputDirectory(string? outputDirectory)
    {
        if (_environment.TryWriteToDirectory(outputDirectory ?? string.Empty, out var error))
        {
            return CompatibilityReport.Create([]);
        }

        return CompatibilityReport.Create(
        [
            new CompatibilityIssue(
                "output-write",
                "Selected output folder is not writable",
                $"The app could not write to '{outputDirectory}'. {error}",
                "Choose a writable folder before exporting WAV files.",
                isBlocking: true),
        ]);
    }
}
