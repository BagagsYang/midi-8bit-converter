using System.Runtime.InteropServices;

namespace Midi8BitSynthesiser.App.Compatibility;

internal interface ICompatibilityEnvironment
{
    Version OperatingSystemVersion { get; }

    Architecture OperatingSystemArchitecture { get; }

    string BaseDirectory { get; }

    string TempDirectory { get; }

    string DefaultOutputDirectory { get; }

    bool FileExists(string path);

    bool TryWriteToDirectory(string path, out string? errorMessage);
}
