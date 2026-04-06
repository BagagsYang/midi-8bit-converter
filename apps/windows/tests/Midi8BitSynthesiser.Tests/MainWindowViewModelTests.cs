using Midi8BitSynthesiser.App.Compatibility;
using Midi8BitSynthesiser.App.Services;
using Midi8BitSynthesiser.App.ViewModels;
using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.Tests;

public sealed class MainWindowViewModelTests
{
    [Fact]
    public void MoveCommands_ReorderQueue()
    {
        var viewModel = CreateViewModel();
        viewModel.AddFiles(["first.mid", "second.mid", "third.mid"]);

        viewModel.MoveJobDown(viewModel.Queue[0]);
        viewModel.MoveJobUp(viewModel.Queue[2]);

        Assert.Equal(["second.mid", "third.mid", "first.mid"], viewModel.Queue.Select(job => job.FileName));
    }

    [Fact]
    public async Task StartConversionAsync_SetsProcessingStateUntilRendererCompletes()
    {
        var renderGate = new TaskCompletionSource<bool>(TaskCreationOptions.RunContinuationsAsynchronously);
        var renderEngine = new BlockingRenderEngine(renderGate);
        var dialogService = new StubFileDialogService(outputFolder: Path.GetTempPath());
        var viewModel = CreateViewModel(renderEngine, dialogService);
        viewModel.AddFiles(["lead.mid"]);

        var exportTask = viewModel.StartConversionAsync();
        await renderEngine.WaitUntilStartedAsync();

        Assert.True(viewModel.IsProcessing);
        Assert.False(viewModel.CanStartConversion);
        Assert.Equal("Processing...", viewModel.ExportButtonTitle);

        renderGate.SetResult(true);
        await exportTask;

        Assert.False(viewModel.IsProcessing);
        Assert.Equal("Finished 1 file(s).", viewModel.StatusMessage);
        Assert.Equal("1 completed, 0 failed.", viewModel.LastRunSummary);
    }

    [Fact]
    public async Task StartConversionAsync_HandlesCancelledFolderSelection()
    {
        var viewModel = CreateViewModel(fileDialogService: new StubFileDialogService(outputFolder: null));
        viewModel.AddFiles(["lead.mid"]);

        await viewModel.StartConversionAsync();

        Assert.Equal("Export cancelled.", viewModel.StatusMessage);
        Assert.False(viewModel.IsProcessing);
    }

    [Fact]
    public async Task StartConversionAsync_BlocksWhenOutputDirectoryFailsCompatibilityCheck()
    {
        var viewModel = CreateViewModel(
            fileDialogService: new StubFileDialogService(outputFolder: Path.GetTempPath()),
            compatibilityProbe: new StubCompatibilityProbe(
                outputDirectoryReport: CompatibilityReport.Create(
                [
                    new CompatibilityIssue(
                        "output-write",
                        "Selected output folder is not writable",
                        "The selected folder cannot be written.",
                        "Choose a writable folder.",
                        isBlocking: true),
                ])));
        viewModel.AddFiles(["lead.mid"]);

        await viewModel.StartConversionAsync();

        Assert.Equal("Export blocked by compatibility check.", viewModel.StatusMessage);
        Assert.Contains("Choose a writable folder.", viewModel.LastErrorMessage);
        Assert.False(viewModel.IsProcessing);
    }

    [Fact]
    public void Constructor_ShowsStartupWarning_WhenCompatibilityReportContainsWarnings()
    {
        var startupReport = CompatibilityReport.Create(
        [
            new CompatibilityIssue(
                "default-output",
                "Default output folder is not writable",
                "Documents is not writable.",
                "Choose another export folder.",
                isBlocking: false),
        ]);

        var viewModel = CreateViewModel(startupReport: startupReport);

        Assert.Equal("Compatibility warning detected. Review the message before exporting.", viewModel.StatusMessage);
        Assert.Contains("Choose another export folder.", viewModel.LastErrorMessage);
    }

    private static MainWindowViewModel CreateViewModel(
        IRenderEngine? renderEngine = null,
        IFileDialogService? fileDialogService = null,
        IAudioPreviewPlayer? audioPreviewPlayer = null,
        ICompatibilityProbe? compatibilityProbe = null,
        CompatibilityReport? startupReport = null)
    {
        return new MainWindowViewModel(
            renderEngine ?? new PassThroughRenderEngine(),
            fileDialogService ?? new StubFileDialogService(Path.GetTempPath()),
            audioPreviewPlayer ?? new StubAudioPreviewPlayer(),
            compatibilityProbe ?? new StubCompatibilityProbe(),
            startupReport);
    }

    private sealed class StubFileDialogService(string? outputFolder) : IFileDialogService
    {
        public Task<IReadOnlyList<string>> PickMidiFilesAsync(CancellationToken cancellationToken)
            => Task.FromResult<IReadOnlyList<string>>(Array.Empty<string>());

        public Task<string?> PickOutputFolderAsync(string? defaultDirectory, CancellationToken cancellationToken)
            => Task.FromResult(outputFolder);
    }

    private sealed class StubAudioPreviewPlayer : IAudioPreviewPlayer
    {
        public void Dispose()
        {
        }

        public Task PlayAsync(WaveLayer layer, CancellationToken cancellationToken) => Task.CompletedTask;
    }

    private sealed class StubCompatibilityProbe(CompatibilityReport? startupReport = null, CompatibilityReport? outputDirectoryReport = null) : ICompatibilityProbe
    {
        public CompatibilityReport EvaluateStartup() => startupReport ?? CompatibilityReport.Create([]);

        public CompatibilityReport EvaluateOutputDirectory(string? outputDirectory)
            => outputDirectoryReport ?? CompatibilityReport.Create([]);
    }

    private sealed class PassThroughRenderEngine : IRenderEngine
    {
        public Task<RenderResult> RenderAsync(RenderRequest request, CancellationToken cancellationToken)
            => Task.FromResult(new RenderResult(request.OutputPath, TimeSpan.FromSeconds(1)));
    }

    private sealed class BlockingRenderEngine(TaskCompletionSource<bool> gate) : IRenderEngine
    {
        private readonly TaskCompletionSource<bool> _started = new(TaskCreationOptions.RunContinuationsAsynchronously);

        public async Task<RenderResult> RenderAsync(RenderRequest request, CancellationToken cancellationToken)
        {
            _started.SetResult(true);
            await gate.Task.WaitAsync(cancellationToken);
            return new RenderResult(request.OutputPath, TimeSpan.FromSeconds(1));
        }

        public Task WaitUntilStartedAsync() => _started.Task;
    }
}
