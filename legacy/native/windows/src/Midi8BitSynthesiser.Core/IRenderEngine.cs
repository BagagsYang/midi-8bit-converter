namespace Midi8BitSynthesiser.Core;

public interface IRenderEngine
{
    Task<RenderResult> RenderAsync(RenderRequest request, CancellationToken cancellationToken);
}
