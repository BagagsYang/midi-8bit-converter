# AGENTS.md

## Repository overview
This repository is a monorepo.

- apps/web-flask: primary Flask web UI
- apps/macos: native SwiftUI macOS app
- apps/windows: native WinUI 3 Windows app
- core/python-renderer: canonical Python renderer
- assets/previews: shared waveform preview assets

## Common setup
Run commands from the repository root unless noted otherwise.

```bash
python3 -m venv .venv
```

Install only the dependencies needed for the area you are touching:

```bash
./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt
./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt
./.venv/bin/python3 -m pip install -r core/python-renderer/requirements.txt
```

## Workflows and commands

### Web Flask
- Run locally: `./.venv/bin/python3 apps/web-flask/app.py`
- macOS launcher: `apps/web-flask/Launch_Synthesiser.command`
- Windows launcher: `apps\web-flask\Launch_Synthesiser.bat`
- Tests: `./.venv/bin/python3 -m unittest discover -s apps/web-flask/tests`

### Python renderer
- Entrypoint: `core/python-renderer/midi_to_wave.py`
- Tests: `./.venv/bin/python3 -m unittest discover -s core/python-renderer/tests`
- CLI supports input MIDI path, output WAV path, `--type`, `--duty`, `--rate`, and `--layers-json`.

### macOS
- Build through Xcode with `apps/macos/MIDI8BitSynthesiser.xcodeproj` and the `MIDI8BitSynthesiser` scheme.
- The Xcode build phase runs `apps/macos/macos/build_desktop_resources.sh`.
- TODO: Add a command-line `xcodebuild` workflow after it is verified in repo usage.

### Windows
- Preflight: `dotnet --info`, `python --version`, `python -c "import pretty_midi, numpy, scipy"`
- Restore: `dotnet restore apps/windows/Midi8BitSynthesiser.sln`
- Build: `dotnet build apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64`
- Test: `dotnet test apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build`
- Publish: `dotnet publish apps/windows/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj -c Release -r win-x64 --self-contained true -p:Platform=x64`
- Review bundle: `apps/windows/scripts/create_review_bundle.sh`

## Localisation rules
- Do not duplicate app source trees for Chinese.
- Keep one codebase per platform.
- Extract user-facing strings into platform-appropriate localisation resources.
- Documentation may have parallel Chinese files named `*.zh-CN.md`.
- Prefer English as the fallback locale.
- Preserve existing behaviour unless the task explicitly asks for UI changes.

## Web app rules
- Avoid keeping large inline JS in templates when adding localisation.
- Prefer `i18n/*.json` plus separate JS helpers.
- Do not change synthesis logic in `core/python-renderer` for localisation-only tasks.
- Prefer `apps/web-flask/Launch_Synthesiser.command` or `apps/web-flask/Launch_Synthesiser.bat` when asks for running a server.

## macOS rules
- Use Apple-native localisation files.
- Do not hardcode Chinese strings directly in Swift views.

## Windows rules
- Use WinUI resource files (`.resw`) for localisation.
- Do not hardcode Chinese strings directly in XAML if a resource key can be used.

## Validation
After making changes, run only the checks relevant to the touched app and report what was and was not run.
