# Windows app

Language/语言: English | [简体中文](./README.zh-CN.md)

This folder contains the retained native Windows desktop rewrite for OctaBit.

## Deprecation status

This native Windows app is deprecated/paused. It is not the main development
target; the project currently focuses on the web service. The code is retained
for reference.

## Retained responsibilities

- WinUI 3 desktop interface for Windows
- Native queue, layer editing, preview, and export workflow
- C# renderer that is validated against the Python reference renderer

## Project layout

- `src/Midi8BitSynthesiser.Core/`: rendering engine, waveform models, output naming
- `src/Midi8BitSynthesiser.App/`: WinUI 3 shell, file dialog integration, preview playback
- `tests/Midi8BitSynthesiser.Tests/`: unit tests, workflow tests, Python parity tests

## Build on Windows

From the repository root:

1. Install .NET 8 SDK and the Visual Studio components required for WinUI 3 desktop development.
2. Install Python 3 and the reference renderer requirements: `python -m pip install -r legacy/python-renderer/requirements.txt`
3. Restore, build, and test:
   - `dotnet restore legacy/native/windows/Midi8BitSynthesiser.sln`
   - `dotnet build legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64`
   - `dotnet test legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build`
No maintained Windows release workflow or CI publish pipeline remains in this
repository. If native packaging work is revived, recreate the publish/release
steps from the current project and installer files instead of relying on a
removed workflow.

## Build requirements for developers and reviewers

Build and test still require:

- .NET 8 SDK
- WinUI 3 compatible Visual Studio components
- Python 3
- `legacy/python-renderer/requirements.txt` installed for parity tests

## Reviewer preflight

Before reporting Windows build or runtime failures, confirm the review machine can actually validate the app:

- `dotnet --info`
- `python --version`
- `python -c "import pretty_midi, numpy, scipy"`

The detailed checklist lives in `REVIEWING.md`.

## Review bundle

To prepare a bundle for an external Windows review, run:

```bash
legacy/native/windows/scripts/create_review_bundle.sh
```

The bundle includes:

- `legacy/native/windows/`
- `legacy/python-renderer/`
- `assets/previews/`
- `global.json`

Historical installer files remain under `installer/`, but there is no
maintained repository-level Windows release workflow at this time.
