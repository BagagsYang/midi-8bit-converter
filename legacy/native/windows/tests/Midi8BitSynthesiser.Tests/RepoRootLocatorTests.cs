using Midi8BitSynthesiser.Tests.TestData;

namespace Midi8BitSynthesiser.Tests;

public sealed class RepoRootLocatorTests
{
    [Fact]
    public void BuildMetadata_ResolvesRepoAndPythonRendererPaths()
    {
        var repoRoot = RepoRootLocator.FindRepoRoot();
        var pythonRendererRoot = RepoRootLocator.FindPythonRendererRoot();
        var pythonRendererScriptPath = RepoRootLocator.FindPythonRendererScriptPath();

        Assert.True(Directory.Exists(repoRoot));
        Assert.True(Directory.Exists(pythonRendererRoot));
        Assert.True(File.Exists(pythonRendererScriptPath));
    }
}
