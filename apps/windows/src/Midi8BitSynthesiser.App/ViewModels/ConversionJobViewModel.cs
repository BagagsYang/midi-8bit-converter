namespace Midi8BitSynthesiser.App.ViewModels;

public enum ConversionJobState
{
    Queued,
    Processing,
    Completed,
    Failed,
}

public sealed class ConversionJobViewModel : ObservableObject
{
    private ConversionJobState _state = ConversionJobState.Queued;
    private string? _outputPath;
    private string? _message;

    public ConversionJobViewModel(string inputPath)
    {
        InputPath = inputPath;
        FileName = Path.GetFileName(inputPath);
    }

    public string InputPath { get; }

    public string FileName { get; }

    public ConversionJobState State
    {
        get => _state;
        set
        {
            if (SetProperty(ref _state, value))
            {
                OnPropertyChanged(nameof(StateText));
                OnPropertyChanged(nameof(StateDetail));
            }
        }
    }

    public string? OutputPath
    {
        get => _outputPath;
        set
        {
            if (SetProperty(ref _outputPath, value))
            {
                OnPropertyChanged(nameof(StateDetail));
            }
        }
    }

    public string? Message
    {
        get => _message;
        set
        {
            if (SetProperty(ref _message, value))
            {
                OnPropertyChanged(nameof(StateDetail));
            }
        }
    }

    public string StateText => State switch
    {
        ConversionJobState.Queued => "Queued",
        ConversionJobState.Processing => "Processing",
        ConversionJobState.Completed => "Completed",
        ConversionJobState.Failed => "Failed",
        _ => "Queued",
    };

    public string StateDetail => State switch
    {
        ConversionJobState.Completed => OutputPath ?? "Completed",
        ConversionJobState.Failed => Message ?? "Failed",
        _ => Message ?? string.Empty,
    };
}
