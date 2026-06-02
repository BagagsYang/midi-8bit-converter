namespace Midi8BitSynthesiser.App.Compatibility;

public sealed record CompatibilityIssue(
    string Code,
    string Title,
    string Description,
    string Remediation,
    bool IsBlocking);
