using System.Runtime.InteropServices;
using Midi8BitSynthesiser.App.Compatibility;

namespace Midi8BitSynthesiser.Tests;

public sealed class CompatibilityProbeTests
{
    private static readonly CompatibilitySupportPolicy Policy = new(
        new Version(10, 0, 17763, 0),
        "win-x64",
        "WinUI 3 self-contained desktop app",
        "1.0.0");

    [Fact]
    public void EvaluateStartup_ReturnsSupported_ForValidX64Environment()
    {
        using var temp = new TempDirectory();
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 19045, 0),
            operatingSystemArchitecture: Architecture.X64,
            baseDirectory: CreatePublishedAssets(temp.Path),
            tempDirectory: temp.Path,
            defaultOutputDirectory: temp.Path);

        var report = new CompatibilityProbe(Policy, environment).EvaluateStartup();

        Assert.Equal(CompatibilityStatus.Supported, report.Status);
        Assert.Empty(report.Issues);
    }

    [Fact]
    public void EvaluateStartup_BlocksUnsupportedWindowsBuild()
    {
        using var temp = new TempDirectory();
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 17762, 0),
            operatingSystemArchitecture: Architecture.X64,
            baseDirectory: CreatePublishedAssets(temp.Path),
            tempDirectory: temp.Path,
            defaultOutputDirectory: temp.Path);

        var report = new CompatibilityProbe(Policy, environment).EvaluateStartup();

        Assert.Equal(CompatibilityStatus.Blocked, report.Status);
        Assert.Contains(report.Issues, issue => issue.Code == "windows-version" && issue.IsBlocking);
    }

    [Fact]
    public void EvaluateStartup_BlocksNonX64Architectures()
    {
        using var temp = new TempDirectory();
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 19045, 0),
            operatingSystemArchitecture: Architecture.Arm64,
            baseDirectory: CreatePublishedAssets(temp.Path),
            tempDirectory: temp.Path,
            defaultOutputDirectory: temp.Path);

        var report = new CompatibilityProbe(Policy, environment).EvaluateStartup();

        Assert.Equal(CompatibilityStatus.Blocked, report.Status);
        Assert.Contains(report.Issues, issue => issue.Code == "architecture" && issue.IsBlocking);
    }

    [Fact]
    public void EvaluateStartup_BlocksMissingPreviewAssets()
    {
        using var temp = new TempDirectory();
        Directory.CreateDirectory(Path.Combine(temp.Path, "Assets", "Previews"));
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 19045, 0),
            operatingSystemArchitecture: Architecture.X64,
            baseDirectory: temp.Path,
            tempDirectory: temp.Path,
            defaultOutputDirectory: temp.Path);

        var report = new CompatibilityProbe(Policy, environment).EvaluateStartup();

        Assert.Equal(CompatibilityStatus.Blocked, report.Status);
        Assert.Contains(report.Issues, issue => issue.Code == "missing-assets" && issue.IsBlocking);
    }

    [Fact]
    public void EvaluateStartup_WarnsWhenDefaultOutputDirectoryIsNotWritable()
    {
        using var temp = new TempDirectory();
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 19045, 0),
            operatingSystemArchitecture: Architecture.X64,
            baseDirectory: CreatePublishedAssets(temp.Path),
            tempDirectory: temp.Path,
            defaultOutputDirectory: Path.Combine(temp.Path, "readonly"))
        {
            NonWritableDirectories = { Path.Combine(temp.Path, "readonly") },
        };
        Directory.CreateDirectory(environment.DefaultOutputDirectory);

        var report = new CompatibilityProbe(Policy, environment).EvaluateStartup();

        Assert.Equal(CompatibilityStatus.Warning, report.Status);
        Assert.Contains(report.Issues, issue => issue.Code == "default-output" && !issue.IsBlocking);
    }

    [Fact]
    public void EvaluateOutputDirectory_BlocksUnwritableFolder()
    {
        using var temp = new TempDirectory();
        var blockedDirectory = Path.Combine(temp.Path, "blocked");
        Directory.CreateDirectory(blockedDirectory);
        var environment = new FakeCompatibilityEnvironment(
            operatingSystemVersion: new Version(10, 0, 19045, 0),
            operatingSystemArchitecture: Architecture.X64,
            baseDirectory: CreatePublishedAssets(temp.Path),
            tempDirectory: temp.Path,
            defaultOutputDirectory: temp.Path)
        {
            NonWritableDirectories = { blockedDirectory },
        };

        var report = new CompatibilityProbe(Policy, environment).EvaluateOutputDirectory(blockedDirectory);

        Assert.Equal(CompatibilityStatus.Blocked, report.Status);
        Assert.Contains(report.Issues, issue => issue.Code == "output-write" && issue.IsBlocking);
    }

    private static string CreatePublishedAssets(string root)
    {
        var previewsDirectory = Path.Combine(root, "Assets", "Previews");
        Directory.CreateDirectory(previewsDirectory);
        foreach (var asset in new[]
                 {
                     "pulse_10.wav",
                     "pulse_25.wav",
                     "pulse_50.wav",
                     "sawtooth.wav",
                     "sine.wav",
                     "triangle.wav",
                 })
        {
            File.WriteAllText(Path.Combine(previewsDirectory, asset), asset);
        }

        return root;
    }

    private sealed class FakeCompatibilityEnvironment : ICompatibilityEnvironment
    {
        public FakeCompatibilityEnvironment(
            Version operatingSystemVersion,
            Architecture operatingSystemArchitecture,
            string baseDirectory,
            string tempDirectory,
            string defaultOutputDirectory)
        {
            OperatingSystemVersion = operatingSystemVersion;
            OperatingSystemArchitecture = operatingSystemArchitecture;
            BaseDirectory = baseDirectory;
            TempDirectory = tempDirectory;
            DefaultOutputDirectory = defaultOutputDirectory;
        }

        public Version OperatingSystemVersion { get; }

        public Architecture OperatingSystemArchitecture { get; }

        public string BaseDirectory { get; }

        public string TempDirectory { get; }

        public string DefaultOutputDirectory { get; }

        public HashSet<string> NonWritableDirectories { get; } = [];

        public bool FileExists(string path) => File.Exists(path);

        public bool TryWriteToDirectory(string path, out string? errorMessage)
        {
            if (NonWritableDirectories.Contains(path))
            {
                errorMessage = "Write access denied.";
                return false;
            }

            errorMessage = null;
            return Directory.Exists(path);
        }
    }
}
