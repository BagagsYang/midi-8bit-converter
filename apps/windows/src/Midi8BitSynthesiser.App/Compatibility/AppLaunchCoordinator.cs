namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed class AppLaunchCoordinator
{
    private readonly ICompatibilityProbe _compatibilityProbe;

    public AppLaunchCoordinator(ICompatibilityProbe compatibilityProbe)
    {
        _compatibilityProbe = compatibilityProbe;
    }

    public AppLaunchDecision CreateDecision()
    {
        var report = _compatibilityProbe.EvaluateStartup();
        return new AppLaunchDecision(!report.IsBlocked, report);
    }
}
