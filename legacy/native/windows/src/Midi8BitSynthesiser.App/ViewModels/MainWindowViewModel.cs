using System.Collections.ObjectModel;
using System.Collections.Specialized;
using System.ComponentModel;
using Midi8BitSynthesiser.App;
using Midi8BitSynthesiser.App.Compatibility;
using Midi8BitSynthesiser.App.Services;
using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.App.ViewModels;

public sealed class MainWindowViewModel : ObservableObject
{
    private readonly IRenderEngine _renderEngine;
    private readonly IFileDialogService _fileDialogService;
    private readonly IAudioPreviewPlayer _audioPreviewPlayer;
    private readonly ICompatibilityProbe _compatibilityProbe;

    private int _sampleRate = 48_000;
    private bool _isProcessing;
    private string _statusMessage = LocalizedStrings.Get(
        "StatusInitial",
        "Add MIDI files, choose your layer blend, then export a WAV batch.");
    private string? _lastRunSummary;
    private string? _lastErrorMessage;

    public MainWindowViewModel(
        IRenderEngine renderEngine,
        IFileDialogService fileDialogService,
        IAudioPreviewPlayer audioPreviewPlayer,
        ICompatibilityProbe compatibilityProbe,
        CompatibilityReport? startupReport = null)
    {
        _renderEngine = renderEngine;
        _fileDialogService = fileDialogService;
        _audioPreviewPlayer = audioPreviewPlayer;
        _compatibilityProbe = compatibilityProbe;

        Queue.CollectionChanged += OnQueueChanged;
        Layers.CollectionChanged += OnLayersChanged;
        Layers.Add(new WaveLayerViewModel(WaveType.Pulse, 0.5, 1.0));
        RefreshLayerMetadata();

        if (startupReport is { Status: CompatibilityStatus.Warning })
        {
            StatusMessage = LocalizedStrings.Get(
                "StatusCompatibilityWarning",
                "Compatibility warning detected. Review the message before exporting.");
            LastErrorMessage = startupReport.DisplayMessage;
        }
    }

    public ObservableCollection<ConversionJobViewModel> Queue { get; } = [];

    public ObservableCollection<WaveLayerViewModel> Layers { get; } = [];

    public IReadOnlyList<int> SampleRates { get; } = [44_100, 48_000, 96_000];

    public int SampleRate
    {
        get => _sampleRate;
        set
        {
            if (SetProperty(ref _sampleRate, value))
            {
                OnPropertyChanged(nameof(SampleRateTitle));
            }
        }
    }

    public bool IsProcessing
    {
        get => _isProcessing;
        private set
        {
            if (SetProperty(ref _isProcessing, value))
            {
                OnPropertyChanged(nameof(CanAddLayer));
                OnPropertyChanged(nameof(CanRemoveLayer));
                OnPropertyChanged(nameof(CanClearQueue));
                OnPropertyChanged(nameof(CanStartConversion));
                OnPropertyChanged(nameof(ExportButtonTitle));
            }
        }
    }

    public string StatusMessage
    {
        get => _statusMessage;
        private set => SetProperty(ref _statusMessage, value);
    }

    public string? LastRunSummary
    {
        get => _lastRunSummary;
        private set
        {
            if (SetProperty(ref _lastRunSummary, value))
            {
                OnPropertyChanged(nameof(HasSummary));
            }
        }
    }

    public string? LastErrorMessage
    {
        get => _lastErrorMessage;
        private set
        {
            if (SetProperty(ref _lastErrorMessage, value))
            {
                OnPropertyChanged(nameof(HasError));
            }
        }
    }

    public bool CanAddLayer => Layers.Count < 3 && !IsProcessing;

    public bool CanRemoveLayer => Layers.Count > 1 && !IsProcessing;

    public bool CanStartConversion => Queue.Count > 0 && !IsProcessing;

    public bool CanClearQueue => Queue.Count > 0 && !IsProcessing;

    public int AudibleLayerCount => Layers.Count(layer => layer.Volume > 0);

    public string ExportButtonTitle => IsProcessing
        ? LocalizedStrings.Get("ExportButtonProcessing", "Processing...")
        : LocalizedStrings.Get("ExportButtonReady", "Export WAV");

    public string QueueSummary => Queue.Count.ToString();

    public string LayerSummary => Layers.Count.ToString();

    public bool HasQueue => Queue.Count > 0;

    public bool ShowEmptyQueueState => !HasQueue;

    public bool HasError => !string.IsNullOrWhiteSpace(LastErrorMessage);

    public bool HasSummary => !string.IsNullOrWhiteSpace(LastRunSummary);

    public string SampleRateTitle => SampleRate switch
    {
        44_100 => LocalizedStrings.Get("SampleRate44100", "44.1 kHz"),
        48_000 => LocalizedStrings.Get("SampleRate48000", "48 kHz"),
        96_000 => LocalizedStrings.Get("SampleRate96000", "96 kHz"),
        _ => LocalizedStrings.Format("SampleRateCustomFormat", "{0} Hz", SampleRate),
    };

    public async Task ImportFilesAsync(CancellationToken cancellationToken = default)
    {
        if (IsProcessing)
        {
            return;
        }

        var paths = await _fileDialogService.PickMidiFilesAsync(cancellationToken);
        AddFiles(paths);
    }

    public void AddFiles(IEnumerable<string> paths)
    {
        if (IsProcessing)
        {
            return;
        }

        var midiPaths = paths
            .Where(path => SupportedExtensions.Contains(Path.GetExtension(path), StringComparer.OrdinalIgnoreCase))
            .ToList();

        if (midiPaths.Count == 0)
        {
            LastErrorMessage = LocalizedStrings.Get(
                "ErrorUnsupportedFileType",
                "Only .mid and .midi files can be queued.");
            return;
        }

        foreach (var path in midiPaths)
        {
            Queue.Add(new ConversionJobViewModel(path));
        }

        LastErrorMessage = null;
        LastRunSummary = null;
        StatusMessage = LocalizedStrings.Format(
            "StatusFilesQueuedFormat",
            "{0} file(s) queued for export.",
            Queue.Count);
    }

    public void ClearQueue()
    {
        Queue.Clear();
        LastRunSummary = null;
        LastErrorMessage = null;
        StatusMessage = LocalizedStrings.Get(
            "StatusQueueCleared",
            "Queue cleared. Add MIDI files to start another batch.");
    }

    public void RemoveJob(ConversionJobViewModel job)
    {
        if (IsProcessing)
        {
            return;
        }

        Queue.Remove(job);
    }

    public void MoveJobUp(ConversionJobViewModel job) => MoveJob(job, -1);

    public void MoveJobDown(ConversionJobViewModel job) => MoveJob(job, 1);

    public void AddLayer()
    {
        if (!CanAddLayer)
        {
            return;
        }

        var defaults = new[]
        {
            new WaveLayerViewModel(WaveType.Sine, 0.5, 0.5),
            new WaveLayerViewModel(WaveType.Triangle, 0.5, 0.5),
        };

        Layers.Add(defaults[Math.Min(Layers.Count - 1, defaults.Length - 1)]);
    }

    public void RemoveLayer(WaveLayerViewModel layer)
    {
        if (!CanRemoveLayer || !Layers.Contains(layer) || !layer.CanRemove)
        {
            return;
        }

        Layers.Remove(layer);
    }

    public async Task PlayPreviewAsync(WaveLayerViewModel layer, CancellationToken cancellationToken = default)
    {
        try
        {
            await _audioPreviewPlayer.PlayAsync(layer.ToCoreModel(), cancellationToken);
            LastErrorMessage = null;
        }
        catch (Exception ex)
        {
            LastErrorMessage = ex.Message;
        }
    }

    public async Task StartConversionAsync(CancellationToken cancellationToken = default)
    {
        if (!CanStartConversion)
        {
            return;
        }

        var defaultDirectory = Path.GetDirectoryName(Queue[0].InputPath);
        var outputDirectory = await _fileDialogService.PickOutputFolderAsync(defaultDirectory, cancellationToken);
        if (string.IsNullOrWhiteSpace(outputDirectory))
        {
            StatusMessage = LocalizedStrings.Get("StatusExportCancelled", "Export cancelled.");
            return;
        }

        var outputCompatibility = _compatibilityProbe.EvaluateOutputDirectory(outputDirectory);
        if (outputCompatibility.IsBlocked)
        {
            StatusMessage = LocalizedStrings.Get(
                "StatusExportBlockedByCompatibility",
                "Export blocked by compatibility check.");
            LastErrorMessage = outputCompatibility.DisplayMessage;
            return;
        }

        var preparedLayers = LayerSanitizer.Sanitize(Layers.Select(layer => layer.ToCoreModel()));

        foreach (var job in Queue)
        {
            job.State = ConversionJobState.Queued;
            job.OutputPath = null;
            job.Message = null;
        }

        IsProcessing = true;
        LastRunSummary = null;
        LastErrorMessage = null;

        var completedCount = 0;
        var failedCount = 0;

        try
        {
            for (var index = 0; index < Queue.Count; index++)
            {
                cancellationToken.ThrowIfCancellationRequested();

                var job = Queue[index];
                var outputPath = FileNameBuilder.BuildOutputPath(job.InputPath, outputDirectory, preparedLayers);

                job.State = ConversionJobState.Processing;
                StatusMessage = LocalizedStrings.Format(
                    "StatusProcessingFormat",
                    "Processing {0} of {1}: {2}",
                    index + 1,
                    Queue.Count,
                    job.FileName);

                try
                {
                    await _renderEngine.RenderAsync(
                        new RenderRequest(job.InputPath, outputPath, SampleRate, preparedLayers),
                        cancellationToken);

                    job.State = ConversionJobState.Completed;
                    job.OutputPath = outputPath;
                    completedCount++;
                }
                catch (Exception ex)
                {
                    job.State = ConversionJobState.Failed;
                    job.Message = ex.Message;
                    LastErrorMessage = ex.Message;
                    failedCount++;
                }
            }
        }
        finally
        {
            IsProcessing = false;
        }

        StatusMessage = LocalizedStrings.Format(
            "StatusFinishedFormat",
            "Finished {0} file(s).",
            Queue.Count);
        LastRunSummary = LocalizedStrings.Format(
            "SummaryCompletedFailedFormat",
            "{0} completed, {1} failed.",
            completedCount,
            failedCount);
    }

    private void MoveJob(ConversionJobViewModel job, int offset)
    {
        if (IsProcessing)
        {
            return;
        }

        var currentIndex = Queue.IndexOf(job);
        if (currentIndex < 0)
        {
            return;
        }

        var targetIndex = currentIndex + offset;
        if (targetIndex < 0 || targetIndex >= Queue.Count)
        {
            return;
        }

        Queue.Move(currentIndex, targetIndex);
    }

    private void OnQueueChanged(object? sender, NotifyCollectionChangedEventArgs e)
    {
        OnPropertyChanged(nameof(CanStartConversion));
        OnPropertyChanged(nameof(CanClearQueue));
        OnPropertyChanged(nameof(QueueSummary));
        OnPropertyChanged(nameof(HasQueue));
        OnPropertyChanged(nameof(ShowEmptyQueueState));
    }

    private void OnLayersChanged(object? sender, NotifyCollectionChangedEventArgs e)
    {
        if (e.OldItems is not null)
        {
            foreach (WaveLayerViewModel layer in e.OldItems)
            {
                layer.PropertyChanged -= OnLayerPropertyChanged;
            }
        }

        if (e.NewItems is not null)
        {
            foreach (WaveLayerViewModel layer in e.NewItems)
            {
                layer.PropertyChanged += OnLayerPropertyChanged;
            }
        }

        RefreshLayerMetadata();
    }

    private void OnLayerPropertyChanged(object? sender, PropertyChangedEventArgs e)
    {
        if (e.PropertyName is nameof(WaveLayerViewModel.Type) or nameof(WaveLayerViewModel.Duty) or nameof(WaveLayerViewModel.Volume))
        {
            OnPropertyChanged(nameof(AudibleLayerCount));
        }
    }

    private void RefreshLayerMetadata()
    {
        for (var index = 0; index < Layers.Count; index++)
        {
            Layers[index].DisplayIndex = index + 1;
            Layers[index].CanRemove = index > 0;
        }

        OnPropertyChanged(nameof(CanAddLayer));
        OnPropertyChanged(nameof(CanRemoveLayer));
        OnPropertyChanged(nameof(AudibleLayerCount));
        OnPropertyChanged(nameof(LayerSummary));
    }

    private static readonly string[] SupportedExtensions = [".mid", ".midi"];
}
