using Microsoft.UI.Xaml;
using Windows.Storage.Pickers;
using WinRT.Interop;

namespace Midi8BitSynthesiser.App.Services;

public sealed class FileDialogService : IFileDialogService
{
    private readonly Window _window;

    public FileDialogService(Window window)
    {
        _window = window;
    }

    public async Task<IReadOnlyList<string>> PickMidiFilesAsync(CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();

        var picker = new FileOpenPicker();
        picker.FileTypeFilter.Add(".mid");
        picker.FileTypeFilter.Add(".midi");
        InitializeWithWindow.Initialize(picker, WindowNative.GetWindowHandle(_window));

        var files = await picker.PickMultipleFilesAsync();
        return files?.Select(file => file.Path).ToArray() ?? Array.Empty<string>();
    }

    public async Task<string?> PickOutputFolderAsync(string? defaultDirectory, CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        _ = defaultDirectory;

        var picker = new FolderPicker();
        picker.FileTypeFilter.Add("*");
        // Unpackaged WinUI pickers accept library hints here, not arbitrary file-system paths.
        picker.SuggestedStartLocation = PickerLocationId.Downloads;
        InitializeWithWindow.Initialize(picker, WindowNative.GetWindowHandle(_window));

        var folder = await picker.PickSingleFolderAsync();
        return folder?.Path;
    }
}
