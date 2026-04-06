using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.App.ViewModels;

public sealed class WaveLayerViewModel : ObservableObject
{
    private WaveType _type;
    private double _duty;
    private double _volume;
    private int _displayIndex;
    private bool _canRemove;

    public WaveLayerViewModel(WaveType type, double duty, double volume)
    {
        _type = type;
        _duty = duty;
        _volume = volume;
    }

    public IReadOnlyList<WaveType> WaveTypes { get; } = Enum.GetValues<WaveType>();

    public WaveType Type
    {
        get => _type;
        set
        {
            if (SetProperty(ref _type, value))
            {
                OnPropertyChanged(nameof(IsPulse));
            }
        }
    }

    public double Duty
    {
        get => _duty;
        set
        {
            if (SetProperty(ref _duty, value))
            {
                OnPropertyChanged(nameof(DutyLabel));
            }
        }
    }

    public double Volume
    {
        get => _volume;
        set
        {
            if (SetProperty(ref _volume, value))
            {
                OnPropertyChanged(nameof(VolumeLabel));
            }
        }
    }

    public int DisplayIndex
    {
        get => _displayIndex;
        set
        {
            if (SetProperty(ref _displayIndex, value))
            {
                OnPropertyChanged(nameof(Title));
            }
        }
    }

    public bool CanRemove
    {
        get => _canRemove;
        set => SetProperty(ref _canRemove, value);
    }

    public bool IsPulse => Type == WaveType.Pulse;

    public string Title => $"Layer {DisplayIndex}";

    public string DutyLabel => $"Pulse Width: {Duty:F2}";

    public string VolumeLabel => $"Volume: {Volume:F1}";

    public WaveLayer ToCoreModel() => new(Type, Duty, Volume);
}
