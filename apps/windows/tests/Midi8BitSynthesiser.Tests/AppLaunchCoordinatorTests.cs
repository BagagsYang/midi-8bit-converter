using Midi8BitSynthesiser.App.Compatibility;

namespace Midi8BitSynthesiser.Tests;

public sealed class AppLaunchCoordinatorTests
{
    [Fact]
    public void CreateDecision_BlocksMainWindow_WhenStartupReportIsBlocked()
    {
        var report = CompatibilityReport.Create(
        [
            new CompatibilityIssue(
                "windows-version",
                "Unsupported Windows version",
                "The machine is too old.",
                "Upgrade Windows.",
                isBlocking: true),
        ]);

        var decision = new AppLaunchCoordinator(new StubCompatibilityProbe(report)).CreateDecision();

        Assert.False(decision.ShouldLaunchMainWindow);
        Assert.Equal(CompatibilityStatus.Blocked, decision.Report.Status);
    }

    [Fact]
    public void CreateDecision_Proceeds_WhenStartupReportIsSupported()
    {
        var decision = new AppLaunchCoordinator(new StubCompatibilityProbe(CompatibilityReport.Create([]))).CreateDecision();

        Assert.True(decision.ShouldLaunchMainWindow);
        Assert.Equal(CompatibilityStatus.Supported, decision.Report.Status);
    }

    [Fact]
    public void CompatibilityReport_DisplayMessageReflectsExactIssueText()
    {
        var report = CompatibilityReport.Create(
        [
            new CompatibilityIssue(
                "temp-write",
                "Temporary export folder is not writable",
                "The app could not write to the temp folder.",
                "Grant write access to the temp folder.",
                isBlocking: true),
        ]);

        Assert.Contains("Temporary export folder is not writable", report.DisplayMessage);
        Assert.Contains("Grant write access to the temp folder.", report.DisplayMessage);
    }

    private sealed class StubCompatibilityProbe(CompatibilityReport startupReport) : ICompatibilityProbe
    {
        public CompatibilityReport EvaluateStartup() => startupReport;

        public CompatibilityReport EvaluateOutputDirectory(string? outputDirectory) => CompatibilityReport.Create([]);
    }
}
