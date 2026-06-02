namespace Midi8BitSynthesiser.App.Services;

public interface IFileDialogService
{
    Task<IReadOnlyList<string>> PickMidiFilesAsync(CancellationToken cancellationToken);

    Task<string?> PickOutputFolderAsync(string? defaultDirectory, CancellationToken cancellationToken);
}
