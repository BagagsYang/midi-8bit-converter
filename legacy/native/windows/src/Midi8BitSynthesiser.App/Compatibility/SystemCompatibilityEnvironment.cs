using System.Runtime.InteropServices;
using Midi8BitSynthesiser.App;

namespace Midi8BitSynthesiser.App.Compatibility;

internal sealed class SystemCompatibilityEnvironment : ICompatibilityEnvironment
{
    public Version OperatingSystemVersion => Environment.OSVersion.Version;

    public Architecture OperatingSystemArchitecture => RuntimeInformation.OSArchitecture;

    public string BaseDirectory => AppContext.BaseDirectory;

    public string TempDirectory => Path.GetTempPath();

    public string DefaultOutputDirectory => ResolveDefaultOutputDirectory();

    public bool FileExists(string path) => File.Exists(path);

    public bool TryWriteToDirectory(string path, out string? errorMessage)
    {
        errorMessage = null;

        if (string.IsNullOrWhiteSpace(path))
        {
            errorMessage = LocalizedStrings.Get("CompatibilityEnvironmentEmptyPath", "The folder path is empty.");
            return false;
        }

        if (!Directory.Exists(path))
        {
            errorMessage = LocalizedStrings.Format(
                "CompatibilityEnvironmentMissingFolderFormat",
                "The folder does not exist: {0}",
                path);
            return false;
        }

        var probePath = Path.Combine(path, $".compatibility-write-test-{Guid.NewGuid():N}.tmp");

        try
        {
            File.WriteAllText(probePath, "compatibility");
            File.Delete(probePath);
            return true;
        }
        catch (Exception ex)
        {
            errorMessage = ex.Message;
            return false;
        }
    }

    private static string ResolveDefaultOutputDirectory()
    {
        var userProfile = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
        if (!string.IsNullOrWhiteSpace(userProfile))
        {
            var downloadsDirectory = Path.Combine(userProfile, "Downloads");
            if (Directory.Exists(downloadsDirectory))
            {
                return downloadsDirectory;
            }
        }

        var documentsDirectory = Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments);
        if (!string.IsNullOrWhiteSpace(documentsDirectory) && Directory.Exists(documentsDirectory))
        {
            return documentsDirectory;
        }

        return Path.GetTempPath();
    }
}
