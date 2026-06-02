namespace Midi8BitSynthesiser.Core;

public sealed record RenderRequest(
    string MidiPath,
    string OutputPath,
    int SampleRate,
    IReadOnlyList<WaveLayer> Layers);
