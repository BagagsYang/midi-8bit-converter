using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Controls;

namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed class CompatibilityWindow : Window
{
    public CompatibilityWindow(CompatibilityReport report)
    {
        Title = "MIDI-8bit Synthesiser Compatibility Check";

        var closeButton = new Button
        {
            Content = "Close",
            HorizontalAlignment = HorizontalAlignment.Left,
        };
        closeButton.Click += (_, _) => Close();

        Content = new ScrollViewer
        {
            Content = new StackPanel
            {
                Spacing = 16,
                Padding = new Thickness(24),
                Children =
                {
                    new TextBlock
                    {
                        Text = "Compatibility Check",
                        FontSize = 28,
                        FontWeight = Microsoft.UI.Text.FontWeights.SemiBold,
                    },
                    new TextBlock
                    {
                        Text = report.Headline,
                        TextWrapping = TextWrapping.WrapWholeWords,
                    },
                    new TextBlock
                    {
                        Text = report.DisplayMessage,
                        TextWrapping = TextWrapping.WrapWholeWords,
                    },
                    new TextBlock
                    {
                        Text = "This Windows release is self-contained. End users do not need the .NET SDK to run it, but this machine still must meet the release compatibility requirements.",
                        TextWrapping = TextWrapping.WrapWholeWords,
                    },
                    closeButton,
                },
            },
        };
    }
}
