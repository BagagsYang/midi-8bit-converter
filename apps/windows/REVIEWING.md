# Windows Review Checklist

Use this checklist before asking a Windows-based reviewer or agent to validate the app.

## Preflight On The Review Machine

The reviewer should stop immediately if any of these fail:

```powershell
dotnet --info
python --version
python -c "import pretty_midi, numpy, scipy"
```

Required outcomes:
- `dotnet --info` shows a .NET 8 SDK
- Python 3 is installed and on `PATH`
- The Python reference renderer dependencies import successfully

If the reviewer cannot pass the preflight, they can still do code inspection, but they should not report restore/build/test failures as app defects.

## Runtime Versus Build Requirements

Keep these separate in the report:

- End-user runtime:
  - does not require the .NET SDK
  - uses the portable zip or installer output
- Reviewer/developer build:
  - does require the .NET 8 SDK
  - does require Python for parity tests

## Bundle Contents

The review bundle should include:
- `apps/windows/`
- `core/python-renderer/midi_to_wave.py`
- `core/python-renderer/requirements.txt`
- `core/python-renderer/README.md`
- `assets/previews/`
- `.github/workflows/windows-release.yml`
- `global.json`

## Create The Bundle

From the repository root:

```bash
apps/windows/scripts/create_review_bundle.sh
```

That script creates `windows-review-bundle.zip` at the repository root and includes the workflow file so the reviewer can validate the intended CI configuration.

## Compatibility Checks To Report

When the Windows app is runnable, verify:
- startup on a supported x64 Windows machine without the .NET SDK installed
- behavior when bundled preview assets are missing
- behavior when `%TEMP%` is not writable
- behavior when the selected export folder is not writable
- installer rejection on unsupported Windows versions or architectures

## Clean-Machine Release Check

For a release candidate, prefer one validation pass on a clean supported Windows VM:
- do not install the .NET SDK
- install or unzip the release
- launch the app and confirm the compatibility message is clear
- export a WAV successfully
