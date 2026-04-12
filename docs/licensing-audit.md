# Licensing Audit

Date checked: 2026-04-12

This report supports the current repository-level AGPL adoption step. The
repository now carries AGPL licensing materials for the repository-owned code
updated in this phase. The remaining questions documented here are mainly about
third-party dependencies and future app-distribution analysis, not about
whether the repository has adopted AGPL at the repository level.

This report distinguishes between:

- facts established from tracked repository files, and
- assessments that depend on installed-package metadata, package registry pages,
  or later bundle/publish review.

It is a documentation audit, not a legal opinion.

## Repository evidence reviewed

Tracked files reviewed for this update:

- `core/python-renderer/requirements.txt`
- `apps/web-flask/requirements.txt`
- `apps/macos/requirements-build.txt`
- `apps/windows/Directory.Packages.props`
- `apps/windows/src/Midi8BitSynthesiser.Core/Midi8BitSynthesiser.Core.csproj`
- `apps/windows/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj`
- `apps/windows/tests/Midi8BitSynthesiser.Tests/Midi8BitSynthesiser.Tests.csproj`
- `global.json`
- `assets/README.md`
- `assets/previews/*.wav`

## Facts established from repository contents

- `assets/README.md` describes `assets/previews/` as the canonical source of the
  shared waveform preview WAV files.
- The Windows app project file does not rely solely on a local `Assets/Previews`
  source folder for publish input. Instead,
  `apps/windows/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj`
  explicitly includes `..\..\..\..\assets\previews\*.wav`, links those files as
  `Assets\Previews\%(Filename)%(Extension)`, and marks them for both output and
  publish copy.
- The same Windows project file is configured for self-contained publishing via
  `RuntimeIdentifier=win-x64`, `SelfContained=true`,
  `PublishSelfContained=true`, and `WindowsAppSDKSelfContained=true`.
- The repository manifests identify the dependency sets that matter for later
  distribution review:
  - Python runtime dependencies in `core/python-renderer/requirements.txt`
  - Flask/web dependencies in `apps/web-flask/requirements.txt`
  - macOS helper-build dependencies in `apps/macos/requirements-build.txt`
  - Windows app and test dependencies in the `.csproj` files plus
    `apps/windows/Directory.Packages.props`
- The preview WAV files under `assets/previews/` are shared repository assets
  used across app outputs according to `assets/README.md` and the Windows
  project file.
- `assets/README.md` now records a maintainer-provided provenance description
  for the preview WAV files: they are project-specific preview/test assets; the
  underlying MIDI material was generated with LLM assistance from the
  maintainer's prompts; and the WAV files themselves were rendered by this
  repository's own program.
- The same provenance description states that, to the maintainer's knowledge,
  the preview WAV files are not derived from third-party sample packs or
  externally licensed audio recordings, and that they are intended to be
  redistributed with repository and app outputs as project preview assets.

## Dependency assessments that rely on external metadata

The licence descriptions in this section are not established from tracked
repository files alone. They are based on installed package metadata and, for
some .NET packages, package-registry pages previously consulted during this
audit work. They are useful for triage, but they should not be treated as if the
repository itself proves them.

### Core Python renderer runtime

- `importlib_resources` 6.5.2: apparent Apache-2.0 from installed package
  metadata. No immediate concern identified for this repository-level step.
- `mido` 1.3.3: apparent MIT from installed package metadata. No immediate
  concern identified for this repository-level step.
- `numpy` 2.4.3: installed package metadata reports
  `BSD-3-Clause AND 0BSD AND MIT AND Zlib AND CC0-1.0`. No immediate concern
  identified for this repository-level step.
- `packaging` 26.0: installed package metadata reports
  `Apache-2.0 OR BSD-2-Clause`. No immediate concern identified for this
  repository-level step.
- `pretty_midi` 0.2.11: apparent MIT from installed package metadata. No
  immediate concern identified for this repository-level step.
- `scipy` 1.17.1: installed package metadata indicates a BSD-style core licence
  and references additional bundled native-library notices in the installed
  wheel. This does not by itself block the repository-level AGPL step, but it
  does mean later binary-distribution review should confirm what notices and
  obligations apply to any shipped bundles.
- `setuptools` 82.0.1: apparent MIT from installed package metadata. No
  immediate concern identified for this repository-level step.
- `six` 1.17.0: apparent MIT from installed package metadata. No immediate
  concern identified for this repository-level step.

### Web Flask dependencies

- `Flask` 3.1.3, `Werkzeug` 3.1.7, `click` 8.3.1, and `MarkupSafe` 3.0.3:
  apparent BSD-3-Clause from installed package metadata. No immediate concern
  identified for this repository-level step.
- `itsdangerous` 2.2.0 and `Jinja2` 3.1.6: apparent BSD-style licensing from
  installed package metadata. No immediate concern identified for this
  repository-level step.
- `blinker` 1.9.0: apparent MIT from installed package metadata. No immediate
  concern identified for this repository-level step.

### macOS helper-build dependencies

- `altgraph` 0.17.5 and `macholib` 1.16.4: apparent MIT from installed package
  metadata. No immediate concern identified for this repository-level step.
- `pyinstaller` 6.19.0: installed package metadata describes GPL-2.0-or-later
  with the PyInstaller special exception. This should be treated as a later
  packaging/distribution review item for macOS helper artifacts, not as an
  established blocker to the repository-level AGPL step.
- `pyinstaller-hooks-contrib` 2026.3: the installed package's bundled `LICENSE`
  file describes mixed licensing, with standard hooks under GPL-2.0-or-later
  and runtime hooks under Apache-2.0. This is relevant mainly to future helper
  bundle analysis and should be revisited when reviewing what macOS build output
  actually ships.

### Windows/.NET dependencies

- `Melanchall.DryWetMidi` 8.0.3: previously noted from the NuGet package page as
  MIT. That assessment is external to the repository.
- `NAudio` 2.3.0: previously noted from the NuGet package page as MIT. That
  assessment is external to the repository.
- `Microsoft.NET.Test.Sdk` 18.3.0: previously noted from the NuGet package page
  as MIT. Test dependency only.
- `xunit` 2.9.3: previously noted from the NuGet package page as Apache-2.0.
  Test dependency only.
- `xunit.runner.visualstudio` 3.1.5: treated here as lower-risk because the test
  project marks it `PrivateAssets=all`, but the exact licence should still be
  confirmed during any formal distribution-grade inventory. The current licence
  description is not proven by tracked repository files alone.
- `Microsoft.WindowsAppSDK` 1.8.260317003: the repository clearly shows that the
  Windows app depends on this package and is configured for self-contained
  publishing, but the licence characterisation itself comes from external
  package-page material rather than tracked repository files. This makes the
  Windows publish path a significant redistribution/licensing hotspot. It should
  receive separate legal or packaging review before drawing conclusions about
  AGPL-based redistribution of the Windows self-contained app. This report does
  not treat incompatibility as definitively proven from repository evidence
  alone.

## Bundled asset observations

- `assets/previews/` is the clearest repository-evidenced shared asset set in
  scope for this audit.
- The strongest repository evidence for Windows app packaging is the `.csproj`
  link to `assets/previews/*.wav`, not any inference about a separate Windows
  asset source tree.
- The repository does contain files under
  `apps/windows/src/Midi8BitSynthesiser.App/Assets/Previews/`, but this report
  does not rely on that path to explain Windows build/publish behaviour because
  the project file already points directly to the canonical files under
  `assets/previews/`.
- `assets/README.md` now records the maintainer's provenance explanation for
  these files. On that basis, they are better described as documented
  project-generated preview/test assets than as unexplained third-party audio
  material.
- This documentation update does not attempt to make a broader legal ownership
  determination; it records the stated provenance and intended redistribution
  context so future audits do not treat the files as being of unknown origin.

## Manual follow-up required

- Perform a dedicated Windows redistribution review for the self-contained
  publish path, grounded in the existing project settings and the
  `Microsoft.WindowsAppSDK` dependency. This is a future distribution question,
  not a conclusion already proven solely from tracked repository files.
- Review `scipy` and any bundled native-library notices in the context of actual
  shipped app bundles or helper artifacts.
- Review macOS helper packaging output for `pyinstaller` and
  `pyinstaller-hooks-contrib` so that any notices or licence conditions are
  handled at distribution time.
- Keep `assets/README.md` and this audit aligned with the documented preview
  asset provenance, and document any future generated or replacement preview
  assets with the same level of specificity.
- Confirm the exact licence of `xunit.runner.visualstudio` when preparing any
  formal distribution-grade dependency inventory, while continuing to treat it
  as lower-risk because it appears test-only in this repository.
