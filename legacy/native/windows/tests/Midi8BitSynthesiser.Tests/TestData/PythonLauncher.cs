using System.ComponentModel;
using System.Diagnostics;
using System.Runtime.InteropServices;

namespace Midi8BitSynthesiser.Tests.TestData;

public static class PythonLauncher
{
    public static IReadOnlyList<PythonCommand> GetPreferredCommands()
    {
        return RuntimeInformation.IsOSPlatform(OSPlatform.Windows)
            ? [new PythonCommand("py", ["-3"]), new PythonCommand("python", [])]
            : [new PythonCommand("python3", []), new PythonCommand("python", [])];
    }

    public static Process StartProcess(
        string workingDirectory,
        string scriptPath,
        IReadOnlyList<string> scriptArguments)
    {
        var attemptedCommands = new List<string>();

        foreach (var command in GetPreferredCommands())
        {
            var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = command.FileName,
                    WorkingDirectory = workingDirectory,
                    RedirectStandardError = true,
                    RedirectStandardOutput = true,
                    UseShellExecute = false,
                },
            };

            foreach (var prefixArgument in command.PrefixArguments)
            {
                process.StartInfo.ArgumentList.Add(prefixArgument);
            }

            process.StartInfo.ArgumentList.Add(scriptPath);
            foreach (var scriptArgument in scriptArguments)
            {
                process.StartInfo.ArgumentList.Add(scriptArgument);
            }

            try
            {
                process.Start();
                return process;
            }
            catch (Win32Exception)
            {
                attemptedCommands.Add(command.DisplayName);
                process.Dispose();
            }
        }

        throw new InvalidOperationException(
            $"Could not start Python parity renderer. Attempted launchers: {string.Join(", ", attemptedCommands)}.");
    }

    public sealed record PythonCommand(string FileName, IReadOnlyList<string> PrefixArguments)
    {
        public string DisplayName => PrefixArguments.Count == 0
            ? FileName
            : $"{FileName} {string.Join(" ", PrefixArguments)}";
    }
}
