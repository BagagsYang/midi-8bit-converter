using Midi8BitSynthesiser.Tests.TestData;

namespace Midi8BitSynthesiser.Tests;

public sealed class PythonLauncherTests
{
    [Fact]
    public void GetPreferredCommands_UsesPlatformSpecificLauncherOrder()
    {
        var commands = PythonLauncher.GetPreferredCommands().Select(command => command.DisplayName).ToArray();

        if (OperatingSystem.IsWindows())
        {
            Assert.Equal(["py -3", "python"], commands);
            return;
        }

        Assert.Equal(["python3", "python"], commands);
    }
}
