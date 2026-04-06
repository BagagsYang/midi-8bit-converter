using Midi8BitSynthesiser.App.Compatibility;
using Microsoft.UI.Xaml;

namespace Midi8BitSynthesiser.App;

public partial class App : Application
{
    private Window? _window;

    public App()
    {
        InitializeComponent();
    }

    protected override void OnLaunched(LaunchActivatedEventArgs args)
    {
        var compatibilityProbe = new CompatibilityProbe();
        var decision = new AppLaunchCoordinator(compatibilityProbe).CreateDecision();

        _window = decision.ShouldLaunchMainWindow
            ? new MainWindow(compatibilityProbe, decision.Report)
            : new CompatibilityWindow(decision.Report);
        _window.Activate();
    }
}
