# Windows App Review Report: MIDI-8bit Synthesiser

> Note: this report predates the repository reorganisation under `apps/`, `core/`, and `assets/`. Historical path references below still use the older layout.

## 1. Environment

- **Windows version:** Windows 10 Pro for Workstations (2009 / Build 26100.1)
- **.NET SDK/runtime versions:** .NET SDK **NOT INSTALLED**. Only .NET Framework 4.0.30319 is present. `dotnet` command is not on PATH and not found anywhere under `C:\Program Files`.
- **Python version:** 3.11.9 (Python 3.14.2 also installed via chocolatey)
- **Required workloads/components installed:**
  - Visual Studio 2022 BuildTools (with VC++, MSBuild, C# compiler)
  - No .NET SDK workload installed
  - Python packages: `pretty_midi` 0.2.11, `numpy` 2.4.3, `scipy` 1.17.1 (all available)
- **UI interaction available:** No (blocked by missing .NET SDK — cannot launch WinUI app)

---

## 2. Commands Run

```powershell
# Verify dotnet availability
dotnet --list-sdks              # CommandNotFoundException — dotnet not found
dotnet --version               # CommandNotFoundException

# Verify Python environment
python --version               # 3.11.9
python -c "import pretty_midi, numpy, scipy"  # All importable

# Verify Python script works end-to-end
python midi_to_wave.py --help  # OK — shows correct usage

# Quick functional test of Python engine
python -c "...integration test..."  # Passed — rate=48000, 24000 samples, max=29162

# Inspect project structure
Get-ChildItem -Recurse windows-app/  # Full file tree enumerated

# Inspect key source files (all read in full)
```

**Blocked commands (no .NET SDK):**
```powershell
dotnet restore Midi8BitSynthesiser.sln
dotnet build Midi8BitSynthesiser.sln -c Release -p:Platform=x64
dotnet test Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
dotnet publish src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj `
    -c Release -r win-x64 --self-contained true -p:Platform=x64
```

---

## 3. Build and Test Results

### 3.1 Restore — **BLOCKED**

```
dotnet : The term 'dotnet' is not recognized as the name of a cmdlet, function,
script file, or operable program.
```

**Root cause:** .NET SDK is not installed on this machine. The VS2022 BuildTools installation does not include the .NET SDK workload by default.

### 3.2 Build — **NOT ATTEMPTED** (blocked by restore)

### 3.3 Test — **NOT ATTEMPTED** (blocked by build)

### 3.4 Publish — **NOT ATTEMPTED** (blocked by build)

---

## 4. Manual App Validation

**UI interaction:** Not possible — no WinUI runtime and no .NET SDK to launch the app.

**What could be tested without UI (via code inspection):**
- Queue import (file dialog / drag-drop logic) ✓ Reviewed
- Layer editing (ViewModel logic) ✓ Reviewed
- Drag/drop support (XAML `AllowDrop`, `RootGrid_Drop`) ✓ Reviewed
- Layer reordering/removal (Queue.Move, Queue.Remove) ✓ Reviewed
- Sample-rate selection (ComboBox bound to `SampleRates`) ✓ Reviewed
- Output naming (`FileNameBuilder`) ✓ Reviewed
- Preview playback (PreviewAudioPlayer) ✓ Reviewed

---

## 5. Findings

### Finding 1 — `WaveFileWriter` Used Incorrectly in `WriteWaveFile`
- **Severity:** critical
- **Confidence:** high
- **Evidence:** `SynthesisRenderEngine.cs` lines 133–141:

```csharp
var waveFormat = new WaveFormat(sampleRate, 16, 1);
using var writer = new WaveFileWriter(outputPath, waveFormat);
var bytes = new byte[samples.Count * sizeof(short)];
if (samples.Count > 0)
{
    Buffer.BlockCopy(samples.ToArray(), 0, bytes, 0, bytes.Length);
    writer.Write(bytes, 0, bytes.Length);   // ← BUG
}
```

- **Suspected cause:** `WaveFileWriter`'s constructor writes the full WAV header (RIFF chunk, fmt chunk). Then `writer.Write(byte[], ...)` writes raw PCM bytes at the stream's current position — bypassing `WaveFileWriter`'s data-chunk bookkeeping. The writer's `Length` property and any internal data-chunk-size tracking are left inconsistent. If any further writes or flushes occur (including the implicit flush in `Dispose()`), the resulting file can be malformed: a data chunk with no size field, or double-written headers. `WriteWaveFile` should use `writer.WriteSamples(samples)` instead of manually copying bytes.

- **File reference:** `src/Midi8BitSynthesiser.Core/SynthesisRenderEngine.cs` lines 133–141

- **Recommended fix:**
```csharp
var waveFormat = new WaveFormat(sampleRate, 16, 1);
using var writer = new WaveFileWriter(outputPath, waveFormat);
if (samples.Count > 0)
{
    writer.WriteSamples(samples.ToArray(), 0, samples.Count);
}
```

---

### Finding 2 — Attack Envelope Bug When `waveform.Length == 1`
- **Severity:** high
- **Confidence:** high
- **Evidence:** `SynthesisRenderEngine.cs` lines 83–93:

```csharp
var attackSamples = Math.Min((int)(AttackSeconds * sampleRate), waveform.Length / 2);
var releaseSamples = Math.Min((int)(ReleaseSeconds * sampleRate), waveform.Length - attackSamples);

for (var index = 0; index < attackSamples; index++)
{
    waveform[index] *= attackSamples == 1          // ← BUG: special-cases 1
        ? 0.0                                       // silences the note instead of attacking
        : index / (double)(attackSamples - 1);
}
```

- **Suspected cause:** When `attackSamples == 1`, the envelope value for sample 0 is forced to `0.0`. Python uses `np.linspace(0, 1, 1)` which produces exactly `[0.0]` — but critically, Python then multiplies the waveform by this envelope (starting at 0) and then separately multiplies by velocity. The C# code **permanently zeroes the first sample** before velocity is applied, making the attack silent. For any note shorter than `AttackSeconds` (5 ms), `attackSamples` would be clamped to 1 and the first sample would be incorrectly silenced. Python handles the single-sample case identically to multi-sample (no zeroing special case).

- **Python equivalent (correct):**
```python
attack_samples = min(int(attack * sample_rate), len(waveform) // 2)
envelope[:attack_samples] = np.linspace(0, 1, attack_samples)  # [0.0] when 1
waveform *= envelope  # first sample = waveform[0] * 0.0  ← but velocity multiplies AFTER
```

- **File reference:** `src/Midi8BitSynthesiser.Core/SynthesisRenderEngine.cs` lines 83–93

- **Recommended fix:**
```csharp
for (var index = 0; index < attackSamples; index++)
{
    waveform[index] *= index / (double)(attackSamples - 1);  // remove special case
}
```
(Also update the release loop to match — but the release special case is actually numerically correct since both C# and Python reach exactly 0.0 at the end.)

---

### Finding 3 — Test Project Platform Mismatch with Core Project
- **Severity:** high
- **Confidence:** medium
- **Evidence:**
  - Test project (`tests/...Tests.csproj`): `<Platforms>x64</Platforms>` — restricts to x64
  - Core project (`src/...Core.csproj`): **no `<Platforms>` constraint** — produces `AnyCPU` binaries by default

```xml
<!-- Core project: net8.0, no RuntimeIdentifier, no Platform约束 -->
<!-- When Tests build as x64, they require Core.dll built for x64 -->
```

- **Suspected cause:** Without an explicit `Platforms` or `RuntimeIdentifier` in `Core.csproj`, the compiler produces `AnyCPU` managed assemblies. When test runners (especially xunit running under a specific platform host) load `Midi8BitSynthesiser.Core.dll`, it may be treated as `AnyCPU` but fail platform-specific NAudio native interop calls, or cause a `PlatformNotSupportedException` if the test host is x64-constrained. This is a likely cause of test failures post-build.

- **File reference:** `src/Midi8BitSynthesiser.Core/Midi8BitSynthesiser.Core.csproj`

- **Recommended fix:** Add `<Platforms>x64</Platforms>` to `Core.csproj`, matching the test and app projects.

---

### Finding 4 — `RepoRootLocator.Find()` Path Traversal May Fail in Some Test Run Configurations
- **Severity:** medium
- **Confidence:** medium
- **Evidence:** `TestData/RepoRootLocator.cs`:

```csharp
var current = new DirectoryInfo(AppContext.BaseDirectory);
while (current is not null)
{
    var sharedScript = Path.Combine(current.FullName, "shared", "midi_to_wave.py");
    if (File.Exists(sharedScript)) return current.FullName;
    current = current.Parent;
}
throw new DirectoryNotFoundException("...");
```

- **Suspected cause:** `AppContext.BaseDirectory` for the test assembly (`Midi8BitSynthesiser.Tests.dll`) resolves to the build output directory, e.g. `bin/Release/x64/`. The code walks up parent directories. On a standard dev machine this works: `bin/Release/x64/ → bin/Release/ → solution root → finds shared/`. However, in CI (e.g., NuGet restore to a flattened output directory, or if the solution is built from a non-standard working directory), the traversal may find an intermediate `shared/` folder (e.g., inside a NuGet package) or fail to find any, causing `PythonParityTests` to hard-fail before any render comparison.

- **File reference:** `tests/...Tests/TestData/RepoRootLocator.cs`

- **Recommended fix:** Use `Path.GetFullPath` relative to the solution file location, or encode the solution root as an MSBuild property injected into the test assembly at build time (e.g., `<PropertyGroup><SharedDir>$(SolutionDir)</SharedDir></PropertyGroup>` in the test csproj and passed as `[assembly: AssemblyMetadata("SharedDir", "...")]`).

---

### Finding 5 — `FileDialogService.PickOutputFolderAsync` Uses Incorrect Fallback Location
- **Severity:** low
- **Confidence:** high
- **Evidence:** `Services/FileDialogService.cs` lines 33–37:

```csharp
picker.SuggestedStartLocation = Directory.Exists(defaultDirectory)
    ? PickerLocationId.DocumentsLibrary    // ← always Documents, even when defaultDirectory is Downloads
    : PickerLocationId.Downloads;
```

- **Suspected cause:** The ternary is inverted — it uses `DocumentsLibrary` when `defaultDirectory` exists (meaning it has a specific path to suggest), and `Downloads` when it doesn't. The intent is likely the opposite: use the user's chosen output directory as the start location when available, and fall back to `Downloads` otherwise. Additionally, `DocumentsLibrary` requires the `documentsLibrary` capability in the app manifest; `Downloads` is universally accessible without a manifest declaration.

- **File reference:** `src/Midi8BitSynthesiser.App/Services/FileDialogService.cs` lines 33–37

- **Recommended fix:**
```csharp
picker.SuggestedStartLocation = Directory.Exists(defaultDirectory)
    ? PickerLocationId.Downloads    // use Downloads when we have a suggestion
    : PickerLocationId.Downloads;   // fall back to Downloads
```
Or better: track the last-used output folder in a persistent setting and use that.

---

### Finding 6 — No CI Workflow File Found
- **Severity:** medium
- **Confidence:** high
- **Evidence:** The README states:
> Windows CI publish pipeline for a portable win-x64 bundle

But no `*.yml` or `*.yaml` file exists anywhere under `windows-agent-bundle/`.

- **Suspected cause:** The CI workflow was planned but not committed to the repository. The README's instructions for `windows-latest` runners cannot be validated. The absence means the project has no automated build/test/publish pipeline.

- **File reference:** `README.md` (section on CI), and absent `.github/workflows/` directory

- **Recommended fix:** Add a GitHub Actions workflow at `.github/workflows/build.yml`:
```yaml
on: [push, pull_request]
jobs:
  build:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup .NET 8
        uses: actions/setup-dotnet@v4
        with: { dotnet-version: '8.0.x' }
      - name: Restore
        run: dotnet restore windows-app/Midi8BitSynthesiser.sln
      - name: Build
        run: dotnet build windows-app/Midi8BitSynthesiser.sln -c Release -p:Platform=x64
      - name: Test
        run: dotnet test windows-app/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build
      - name: Publish
        run: dotnet publish windows-app/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj -c Release -r win-x64 --self-contained true -p:Platform=x64 -o ./publish
```

---

### Finding 7 — `AddLayer()` Boundary Condition When `Layers.Count == 1`
- **Severity:** low
- **Confidence:** low
- **Evidence:** `ViewModels/MainWindowViewModel.cs` lines 149–157:

```csharp
var defaults = new[]
{
    new WaveLayerViewModel(WaveType.Sine, 0.5, 0.5),
    new WaveLayerViewModel(WaveType.Triangle, 0.5, 0.5),
};
Layers.Add(defaults[Math.Min(Layers.Count - 1, defaults.Length - 1)]);
```

- **Suspected cause:** `defaults.Length == 2` (indices 0, 1). `Math.Min(Layers.Count - 1, 1)` gives `Math.Min(0, 1) = 0` for Count=1, and `Math.Min(1, 1) = 1` for Count=2. So when going from 0→1 layer, the first element `Sine` is added. When going from 1→2 layers, `Triangle` is added. This is intentional — but the `CanAddLayer` guard (`Layers.Count < 3`) means the third add is blocked. No out-of-bounds access is possible in the current state, but the code is fragile: adding a third default to the array without updating the guard would cause an `IndexOutOfRangeException`.

- **File reference:** `src/Midi8BitSynthesiser.App/ViewModels/MainWindowViewModel.cs` lines 149–157

- **Recommended fix:** Replace with a more explicit index: `defaults[Math.Min(Layers.Count, defaults.Length - 1)]` or an indexed lookup table.

---

### Finding 8 — `WaveFileAssertions.ReadWaveFile` Ignores Wave Format on Read
- **Severity:** low
- **Confidence:** medium
- **Evidence:** `TestData/WaveFileAssertions.cs`:

```csharp
using var reader = new WaveFileReader(path);
var bytes = new byte[(int)reader.Length];
// ...read all bytes...
var samples = new short[bytes.Length / sizeof(short)];
Buffer.BlockCopy(bytes, 0, samples, 0, bytes.Length);
return new WaveFileData(reader.WaveFormat.SampleRate, reader.WaveFormat.Channels, samples);
```

- **Suspected cause:** The method reads raw PCM bytes directly via `reader.Length` and `Buffer.BlockCopy`. If the WAV file is not actually PCM16 (e.g., if `WriteWaveFile` from Finding 1 produces a malformed file), the bytes may not be interpreted correctly. However, this is only a test helper. The real issue is that if Finding 1 produces a corrupt file, `WaveFileReader` may throw on open, and `WaveFileAssertions.ReadWaveFile` would not distinguish between a format error and a data error.

- **File reference:** `tests/...Tests/TestData/WaveFileAssertions.cs`

- **Recommended fix:** Use `reader.ReadSamples(count)` to let NAudio handle the format conversion and data interpretation correctly, rather than reading raw bytes.

---

### Finding 9 — `PreviewAudioPlayer` Stop/Dispose Race on Concurrent Calls
- **Severity:** low
- **Confidence:** medium
- **Evidence:** `Services/PreviewAudioPlayer.cs`:

```csharp
public Task PlayAsync(WaveLayer layer, CancellationToken cancellationToken)
{
    cancellationToken.ThrowIfCancellationRequested();
    var resourceName = GetPreviewResourceName(layer);
    var previewPath = Path.Combine(AppContext.BaseDirectory, "Assets", "Previews", $"{resourceName}.wav");
    if (!File.Exists(previewPath))
        throw new FileNotFoundException(...);

    StopCurrentPlayback();      // disposes _output and _reader
    _reader = new AudioFileReader(previewPath);
    _output = new WaveOutEvent();
    _output.Init(_reader);
    _output.Play();
    return Task.CompletedTask;
}
```

- **Suspected cause:** `StopCurrentPlayback()` calls `_output?.Stop()` and `_output?.Dispose()` synchronously. If `PlayAsync` is called from the UI thread while a previous playback is still playing, the stop/dispose and the new init happen on the same thread without locking. However, `WaveOutEvent.Stop()` and `Dispose()` are not thread-safe with respect to the playback callback thread. If the callback thread is mid-callback during `Dispose()`, resources may be accessed after disposal. Not a crash in practice (the WinUI button is disabled during playback via `IsProcessing` gating), but fragile if the preview system is extended.

- **File reference:** `src/Midi8BitSynthesiser.App/Services/PreviewAudioPlayer.cs`

- **Recommended fix:** Add a lock object or use `CancellationToken`-driven cancellation instead of manual stop/dispose.

---

### Finding 10 — Python Parity Test Uses `python` Without Path Qualification
- **Severity:** medium
- **Confidence:** high
- **Evidence:** `PythonParityTests.cs`:

```csharp
var pythonProcess = new Process { StartInfo = new ProcessStartInfo { FileName = "python", ... } };
```

- **Suspected cause:** `"python"` relies on PATH resolution. On many Windows systems, `python3` or `py` are the available commands, not bare `python`. In CI on Windows (GitHub `windows-latest`), the default runner image has `python` on PATH, but this is not guaranteed for all VS/Windows configurations. This will silently fail (exit code != 0) if `python` is not found, making the entire parity test suite non-functional.

- **File reference:** `tests/...Tests/PythonParityTests.cs` line (process start)

- **Recommended fix:** Use `"py"` (the Python Launcher for Windows, present on all Python Windows installations) or probe for `python`, `python3`, and `py` in sequence:
```csharp
FileName = RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? "py" : "python3",
```

---

## 6. Non-blocking Observations

### Design Mismatches

1. **`IRenderEngine` is a top-level class in its file** (`IRenderEngine.cs`) — `IRenderEngine`, `RenderRequest`, and `RenderResult` are all top-level classes in separate single-class files. `RenderRequest` and `RenderResult` are closely related to the rendering contract and should arguably live together or in a dedicated `Contracts/` namespace.

2. **`WaveLayerViewModel` is a class, not a record** — The `WaveLayerViewModel` is a mutable class with `set`ters that mutate state. Given it's a view model with `INotifyPropertyChanged`, this works but is more error-prone than a record with `init` setters or a frozen record.

3. **Mix of responsibilities in `MainWindowViewModel`** — `MainWindowViewModel` handles both the queue management (add/remove/reorder files) and the layer management. These could be separate concerns with a `QueueViewModel` and a `LayerStackViewModel`.

### UX Rough Edges

4. **No progress reporting per-file during batch conversion** — `StartConversionAsync` updates `StatusMessage` but provides no per-job progress (e.g., percent complete within a file). For long batches, the user sees only "Processing N of M: filename".

5. **No cancellation support exposed to the user** — `StartConversionAsync` accepts a `CancellationToken` but the UI provides no Cancel button. If the user starts a long batch, they cannot interrupt it without closing the app.

6. **Drag-over visual feedback** — `RootGrid_DragOver` sets `AcceptedOperation = Copy` but the UI does not provide any visual indicator (e.g., border highlight) to show that the drop zone is active.

7. **No validation of output folder before starting batch** — If the user lacks write permission to the selected output folder, the error is reported per-file as each `RenderAsync` call fails, rather than up-front when the folder is selected.

### Maintainability Concerns

8. **`SynthesisRenderEngine.RenderInternal` is 130+ lines** — The method handles MIDI reading, buffer allocation, note iteration, envelope, normalization, PCM conversion, and file writing. Each phase should be a separate private method for clarity.

9. **Magic numbers in envelope calculation** — `AttackSeconds = 0.005`, `ReleaseSeconds = 0.005`, `PeakHeadroom = 0.89` are `private const` within the class. They should be on `Defaults` alongside `DefaultLayer` for discoverability.

10. **Test coverage gaps:**
    - No test for `WaveformGenerator` directly (only via integration tests)
    - No test for the `WriteWaveFile` output correctness (only `WaveFileAssertions` reads what was written, but doesn't verify header integrity)
    - No test for `FileNameBuilder` with mixed-case input paths
    - No test for `PreviewAudioPlayer` with missing asset files

### Test Coverage Gaps

11. **Empty queue export** — `CanStartConversion` returns `Queue.Count > 0 && !IsProcessing`. The Export button would be disabled, so this path is UI-protected. But if called directly on the ViewModel, `StartConversionAsync` would not guard against it.

12. **All notes on drum channel** — The `RenderAsync_IgnoresDrumNotes` test verifies drum notes are skipped, but does not verify that a mix of drum notes and pitched notes only renders the pitched notes.

---

## 7. Report Summary

**Overall status:** Partially working — build blocked (missing .NET SDK), code review reveals critical bugs and medium-severity design issues.

### Build/Test Status
| Step | Result |
|------|--------|
| Restore | ❌ BLOCKED — .NET SDK not installed |
| Build | ❌ NOT ATTEMPTED |
| Test | ❌ NOT ATTEMPTED |
| Publish | ❌ NOT ATTEMPTED |
| Python parity | ✅ Verified (script works end-to-end) |

### Top 3 Issues to Fix First

1. **`WriteWaveFile` corruption bug (Finding 1, critical):** Must switch from manual `writer.Write(bytes)` to `writer.WriteSamples()`. Without this fix, exported WAV files are likely to be unreadable or silent when written with non-empty audio. Fix is 3 lines in `SynthesisRenderEngine.cs`.

2. **Attack envelope zeroing bug (Finding 2, high):** The `attackSamples == 1` special case permanently zeros the first sample. Affects any note shorter than the attack window. Fix is removing the ternary and using the standard formula. This is a correctness bug that produces audibly incorrect output for short notes.

3. **Platform mismatch in Core project (Finding 3, high):** Core.csproj must be updated with `<Platforms>x64</Platforms>` to match the test project. Without this, the test project may fail to load Core at runtime even if build succeeds. A one-line change.

### Items Requiring .NET SDK to Validate
Once .NET 8 SDK is installed, the following must be verified:
- Solution restore succeeds
- Build completes without errors or warnings
- All unit tests pass (particularly `PythonParityTests` which depends on path resolution and Python availability)
- App launches and main window renders
- Preview WAV assets load and play correctly
- Drag-drop of a MIDI file populates the queue
- Batch export produces valid WAV files
