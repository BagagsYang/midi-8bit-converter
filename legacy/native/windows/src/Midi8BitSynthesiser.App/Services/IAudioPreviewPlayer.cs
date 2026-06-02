using Midi8BitSynthesiser.Core;

namespace Midi8BitSynthesiser.App.Services;

public interface IAudioPreviewPlayer : IDisposable
{
    Task PlayAsync(WaveLayer layer, CancellationToken cancellationToken);
}
