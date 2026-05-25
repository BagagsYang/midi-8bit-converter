using Midi8BitSynthesiser.App.Compatibility;
using Midi8BitSynthesiser.App.Services;
using Midi8BitSynthesiser.App.ViewModels;
using Midi8BitSynthesiser.Core;
using Microsoft.UI.Xaml;
using Microsoft.UI.Xaml.Controls;
using Microsoft.UI.Xaml.Input;
using Windows.ApplicationModel.DataTransfer;
using Windows.Storage;

namespace Midi8BitSynthesiser.App;

public sealed partial class MainWindow : Window
{
    private readonly PreviewAudioPlayer _previewAudioPlayer;

    public MainWindowViewModel ViewModel { get; }

    public MainWindow()
        : this(new CompatibilityProbe(), CompatibilityReport.Create([]))
    {
    }

    internal MainWindow(ICompatibilityProbe compatibilityProbe, CompatibilityReport startupReport)
    {
        InitializeComponent();
        Title = LocalizedStrings.Get("MainWindowTitle", "OctaBit");

        _previewAudioPlayer = new PreviewAudioPlayer();
        ViewModel = new MainWindowViewModel(
            new SynthesisRenderEngine(),
            new FileDialogService(this),
            _previewAudioPlayer,
            compatibilityProbe,
            startupReport);
        Closed += MainWindow_Closed;
    }

    private async void ImportFilesButton_Click(object sender, RoutedEventArgs e)
    {
        await ViewModel.ImportFilesAsync();
    }

    private async void ExportButton_Click(object sender, RoutedEventArgs e)
    {
        await ViewModel.StartConversionAsync();
    }

    private void AddLayerButton_Click(object sender, RoutedEventArgs e)
    {
        ViewModel.AddLayer();
    }

    private async void PreviewLayerButton_Click(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: WaveLayerViewModel layer })
        {
            await ViewModel.PlayPreviewAsync(layer);
        }
    }

    private void RemoveLayerButton_Click(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: WaveLayerViewModel layer })
        {
            ViewModel.RemoveLayer(layer);
        }
    }

    private void RemoveJobButton_Click(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: ConversionJobViewModel job })
        {
            ViewModel.RemoveJob(job);
        }
    }

    private void MoveJobUpButton_Click(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: ConversionJobViewModel job })
        {
            ViewModel.MoveJobUp(job);
        }
    }

    private void MoveJobDownButton_Click(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: ConversionJobViewModel job })
        {
            ViewModel.MoveJobDown(job);
        }
    }

    private async void ClearQueueButton_Click(object sender, RoutedEventArgs e)
    {
        var dialog = new ContentDialog
        {
            Title = LocalizedStrings.Get("ClearQueueDialogTitle", "Clear the entire queue?"),
            Content = LocalizedStrings.Get(
                "ClearQueueDialogContent",
                "This resets the current batch while keeping your sound design settings."),
            PrimaryButtonText = LocalizedStrings.Get("ClearQueueDialogPrimaryButton", "Clear Queue"),
            CloseButtonText = LocalizedStrings.Get("ClearQueueDialogCloseButton", "Cancel"),
            DefaultButton = ContentDialogButton.Close,
            XamlRoot = ((FrameworkElement)Content).XamlRoot,
        };

        var result = await dialog.ShowAsync();
        if (result == ContentDialogResult.Primary)
        {
            ViewModel.ClearQueue();
        }
    }

    private void RootGrid_DragOver(object sender, DragEventArgs e)
    {
        e.AcceptedOperation = DataPackageOperation.Copy;
    }

    private async void RootGrid_Drop(object sender, DragEventArgs e)
    {
        if (!e.DataView.Contains(StandardDataFormats.StorageItems))
        {
            return;
        }

        var items = await e.DataView.GetStorageItemsAsync();
        ViewModel.AddFiles(items.OfType<StorageFile>().Select(file => file.Path));
    }

    private void MainWindow_Closed(object sender, WindowEventArgs args)
    {
        _previewAudioPlayer.Dispose();
    }
}
