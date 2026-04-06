# Windows App

This folder contains the native Windows desktop rewrite of the MIDI-8bit Synthesiser.

## Responsibilities

- WinUI 3 desktop interface for Windows
- Native queue, layer editing, preview, and export workflow
- C# renderer that is validated against the Python reference renderer
- Windows CI publish pipeline for a portable `win-x64` bundle and installer

## Project Layout

- `src/Midi8BitSynthesiser.Core/`: rendering engine, waveform models, output naming
- `src/Midi8BitSynthesiser.App/`: WinUI 3 shell, file dialog integration, preview playback
- `tests/Midi8BitSynthesiser.Tests/`: unit tests, workflow tests, Python parity tests

## Build On Windows

From the repository root:

1. Install .NET 8 SDK and the Visual Studio components required for WinUI 3 desktop development.
2. Install Python 3 and the reference renderer requirements:
   `python -m pip install -r core/python-renderer/requirements.txt`
3. Restore, build, and test:
   `dotnet restore apps/windows/Midi8BitSynthesiser.sln`
   `dotnet build apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64`
   `dotnet test apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build`
4. Publish the portable bundle:
   `dotnet publish apps/windows/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj -c Release -r win-x64 --self-contained true -p:Platform=x64`

The published folder contains the main `.exe`, runtime files, and preview WAV assets linked from `assets/previews/`.

## Runtime Requirements For End Users

The published Windows release is self-contained.

End users need:
- a supported 64-bit Windows installation
- the published app files from the portable zip or installer

End users do not need:
- the .NET SDK
- a local source checkout
- Python

## Build Requirements For Developers And Reviewers

Build, test, and publish still require:
- .NET 8 SDK
- WinUI 3 compatible Visual Studio components
- Python 3
- `core/python-renderer/requirements.txt` installed for parity tests

## Reviewer Preflight

Before reporting Windows build or runtime failures, confirm the review machine can actually validate the app:

- `dotnet --info`
- `python --version`
- `python -c "import pretty_midi, numpy, scipy"`

The detailed checklist lives in `REVIEWING.md`.

## Review Bundle

To prepare a bundle for an external Windows review, run:

```bash
apps/windows/scripts/create_review_bundle.sh
```

The bundle includes:
- `apps/windows/`
- `core/python-renderer/`
- `assets/previews/`
- `.github/workflows/windows-release.yml`
- `global.json`

## Installer And Portable Release

The Windows release ships in two forms:
- a portable self-contained zip for manual distribution and review
- an Inno Setup installer for ordinary end users

Both are built from the same published `win-x64` output.
