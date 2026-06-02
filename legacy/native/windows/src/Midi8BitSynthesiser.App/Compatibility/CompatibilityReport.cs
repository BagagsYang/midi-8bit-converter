using Midi8BitSynthesiser.App;

namespace Midi8BitSynthesiser.App.Compatibility;

public sealed record CompatibilityReport(CompatibilityStatus Status, IReadOnlyList<CompatibilityIssue> Issues)
{
    public bool IsBlocked => Status == CompatibilityStatus.Blocked;

    public bool HasIssues => Issues.Count > 0;

    public string Headline => Status switch
    {
        CompatibilityStatus.Blocked => LocalizedStrings.Get(
            "CompatibilityReportBlockedHeadline",
            "This PC does not meet the current Windows release requirements."),
        CompatibilityStatus.Warning => LocalizedStrings.Get(
            "CompatibilityReportWarningHeadline",
            "This PC can run the app, but one or more compatibility warnings were detected."),
        _ => LocalizedStrings.Get(
            "CompatibilityReportSupportedHeadline",
            "This PC satisfies the current Windows release requirements."),
    };

    public string DisplayMessage =>
        Issues.Count == 0
            ? LocalizedStrings.Get("CompatibilityReportNoIssues", "No compatibility issues were detected.")
            : string.Join(
                $"{Environment.NewLine}{Environment.NewLine}",
                Issues.Select(issue =>
                    $"{issue.Title}{Environment.NewLine}{issue.Description}{Environment.NewLine}{LocalizedStrings.Get("CompatibilityReportHowToFixPrefix", "How to fix:")} {issue.Remediation}"));

    public static CompatibilityReport Create(IEnumerable<CompatibilityIssue> issues)
    {
        var issueList = issues.ToList();
        var status = issueList.Any(issue => issue.IsBlocking)
            ? CompatibilityStatus.Blocked
            : issueList.Count > 0
                ? CompatibilityStatus.Warning
                : CompatibilityStatus.Supported;

        return new CompatibilityReport(status, issueList);
    }
}
