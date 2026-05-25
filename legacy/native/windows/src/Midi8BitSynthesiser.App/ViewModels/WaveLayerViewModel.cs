using Midi8BitSynthesiser.App;
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
        WaveTypeOptions =
        [
            new WaveTypeOption(WaveType.Pulse, LocalizedStrings.Get("WaveTypePulse", "Pulse")),
            new WaveTypeOption(WaveType.Sine, LocalizedStrings.Get("WaveTypeSine", "Sine")),
            new WaveTypeOption(WaveType.Sawtooth, LocalizedStrings.Get("WaveTypeSawtooth", "Sawtooth")),
            new WaveTypeOption(WaveType.Triangle, LocalizedStrings.Get("WaveTypeTriangle", "Triangle")),
        ];
    }

    public IReadOnlyList<WaveTypeOption> WaveTypeOptions { get; }

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

    public string Title => LocalizedStrings.Format("WaveLayerTitleFormat", "Layer {0}", DisplayIndex);

    public string DutyLabel => LocalizedStrings.Format("WaveLayerDutyLabelFormat", "Pulse Width: {0:F2}", Duty);

    public string VolumeLabel => LocalizedStrings.Format("WaveLayerVolumeLabelFormat", "Volume: {0:F1}", Volume);

    public WaveLayer ToCoreModel() => new(Type, Duty, Volume);
}
