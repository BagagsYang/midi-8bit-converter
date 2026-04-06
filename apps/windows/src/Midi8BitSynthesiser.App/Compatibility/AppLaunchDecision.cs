namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed record AppLaunchDecision(bool ShouldLaunchMainWindow, CompatibilityReport Report);
