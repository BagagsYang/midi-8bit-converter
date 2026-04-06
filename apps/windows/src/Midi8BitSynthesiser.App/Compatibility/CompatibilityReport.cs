namespace Midi8BitSynthesiser.App.Compatibility;

public sealed record CompatibilityReport(CompatibilityStatus Status, IReadOnlyList<CompatibilityIssue> Issues)
{
    public bool IsBlocked => Status == CompatibilityStatus.Blocked;

    public bool HasIssues => Issues.Count > 0;

    public string Headline => Status switch
    {
        CompatibilityStatus.Blocked => "This PC does not meet the current Windows release requirements.",
        CompatibilityStatus.Warning => "This PC can run the app, but one or more compatibility warnings were detected.",
        _ => "This PC satisfies the current Windows release requirements.",
    };

    public string DisplayMessage =>
        Issues.Count == 0
            ? "No compatibility issues were detected."
            : string.Join(
                $"{Environment.NewLine}{Environment.NewLine}",
                Issues.Select(issue =>
                    $"{issue.Title}{Environment.NewLine}{issue.Description}{Environment.NewLine}How to fix: {issue.Remediation}"));

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
