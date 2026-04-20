using System.Runtime.InteropServices;
using Midi8BitSynthesiser.App;

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
                LocalizedStrings.Get("CompatibilityIssueWindowsVersionTitle", "Unsupported Windows version"),
                LocalizedStrings.Format(
                    "CompatibilityIssueWindowsVersionDescriptionFormat",
                    "This release requires Windows {0} or newer, but the current machine reports {1}.",
                    _policy.MinimumWindowsVersionDisplay,
                    _environment.OperatingSystemVersion),
                LocalizedStrings.Format(
                    "CompatibilityIssueWindowsVersionRemediationFormat",
                    "Install the app on Windows {0} or newer.",
                    _policy.MinimumWindowsVersionDisplay),
                isBlocking: true));
        }

        if (_environment.OperatingSystemArchitecture != Architecture.X64)
        {
            issues.Add(new CompatibilityIssue(
                "architecture",
                LocalizedStrings.Get("CompatibilityIssueArchitectureTitle", "Unsupported processor architecture"),
                LocalizedStrings.Format(
                    "CompatibilityIssueArchitectureDescriptionFormat",
                    "This release is published as {0} and the current operating system architecture is {1}.",
                    _policy.RuntimeIdentifier,
                    _environment.OperatingSystemArchitecture),
                LocalizedStrings.Get(
                    "CompatibilityIssueArchitectureRemediation",
                    "Use a supported x64 Windows machine for this release."),
                isBlocking: true));
        }

        var missingAssets = RequiredPreviewAssets
            .Where(asset => !_environment.FileExists(Path.Combine(_environment.BaseDirectory, "Assets", "Previews", asset)))
            .ToArray();
        if (missingAssets.Length > 0)
        {
            issues.Add(new CompatibilityIssue(
                "missing-assets",
                LocalizedStrings.Get("CompatibilityIssueMissingAssetsTitle", "Bundled preview assets are missing"),
                LocalizedStrings.Format(
                    "CompatibilityIssueMissingAssetsDescriptionFormat",
                    "The published app is missing required preview files: {0}.",
                    string.Join(", ", missingAssets)),
                LocalizedStrings.Get(
                    "CompatibilityIssueMissingAssetsRemediation",
                    "Reinstall the app or use a release package that contains the full published output."),
                isBlocking: true));
        }

        if (!_environment.TryWriteToDirectory(_environment.TempDirectory, out var tempWriteError))
        {
            issues.Add(new CompatibilityIssue(
                "temp-write",
                LocalizedStrings.Get(
                    "CompatibilityIssueTempWriteTitle",
                    "Temporary export folder is not writable"),
                LocalizedStrings.Format(
                    "CompatibilityIssueTempWriteDescriptionFormat",
                    "The app could not write to the temporary folder '{0}'. {1}",
                    _environment.TempDirectory,
                    tempWriteError),
                LocalizedStrings.Get(
                    "CompatibilityIssueTempWriteRemediation",
                    "Make sure the current Windows user can write to the temp folder before launching the app."),
                isBlocking: true));
        }

        var defaultOutputDirectory = _environment.DefaultOutputDirectory;
        if (!_environment.TryWriteToDirectory(defaultOutputDirectory, out var outputWriteError))
        {
            issues.Add(new CompatibilityIssue(
                "default-output",
                LocalizedStrings.Get(
                    "CompatibilityIssueDefaultOutputTitle",
                    "Default output folder is not writable"),
                LocalizedStrings.Format(
                    "CompatibilityIssueDefaultOutputDescriptionFormat",
                    "The default output location '{0}' is not writable. {1}",
                    defaultOutputDirectory,
                    outputWriteError),
                LocalizedStrings.Get(
                    "CompatibilityIssueDefaultOutputRemediation",
                    "Choose another writable export folder when you export WAV files."),
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
                LocalizedStrings.Get(
                    "CompatibilityIssueOutputWriteTitle",
                    "Selected output folder is not writable"),
                LocalizedStrings.Format(
                    "CompatibilityIssueOutputWriteDescriptionFormat",
                    "The app could not write to '{0}'. {1}",
                    outputDirectory,
                    error),
                LocalizedStrings.Get(
                    "CompatibilityIssueOutputWriteRemediation",
                    "Choose a writable folder before exporting WAV files."),
                isBlocking: true),
        ]);
    }
}
