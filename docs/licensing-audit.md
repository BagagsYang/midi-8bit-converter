# Licensing audit

Date checked: 2026-05-19

This is a repository documentation audit, not legal advice. It records what was
confirmed from the current checkout, installed package metadata, and selected
package-registry metadata. Items that require judgement about compatibility,
redistribution, or ownership are marked for human or legal review.

## Audit scope

This audit covered the repository root and the current tracked file set, with
local ignored build/cache folders inspected only to understand build outputs.
The project direction is currently focused on the Vue production frontend plus
Flask/Gunicorn backend service; native macOS and Windows code plus historical
packaging files remain documented here only because they are retained for
reference or possible future revival.

Reviewed repository areas:

- Root documentation and licence files: `README.md`, `README.zh-CN.md`,
  `LICENSE.md`, `AGENTS.md`, `.gitattributes`, `.gitignore`, `.dockerignore`.
- Current app target and retained app code: `apps/web-vue/`, `apps/web-flask/`,
  `apps/macos/`, `apps/windows/`, and the placeholder `apps/desktop/`.
- Shared code and assets: `core/python-renderer/` and `assets/previews/`.
- Documentation and generated review artefacts: `docs/`, tracked files under
  `output/pdf/`, and tracked files under `tmp/pdfs/`.
- Deployment and packaging: `deploy/production/`, `deploy/web-flask/Dockerfile`,
  `compose.web.yml`, `apps/windows/installer/Midi8BitSynthesiser.iss`,
  `apps/windows/scripts/create_review_bundle.sh`, and the macOS Xcode build
  phase script.
Dependency and packaging sources found:

- Python requirements:
  - `core/python-renderer/requirements.txt`
  - `apps/web-flask/requirements.txt`
  - `apps/macos/requirements-build.txt`
- .NET and Windows packaging:
  - `global.json`
  - `apps/windows/Directory.Packages.props`
  - `apps/windows/Midi8BitSynthesiser.sln`
  - Windows app, core, and test `.csproj` files
  - `apps/windows/installer/Midi8BitSynthesiser.iss`
- macOS packaging:
  - `apps/macos/MIDI8BitSynthesiser.xcodeproj/project.pbxproj`
  - `apps/macos/macos/build_desktop_resources.sh`
- Web runtime resources:
  - Vue frontend package metadata in `apps/web-vue/package.json` and
    `apps/web-vue/package-lock.json`
  - CDN Bootstrap CSS in `apps/web-flask/templates/index.html`
  - Google-hosted IBM Plex Sans and IBM Plex Mono font CSS in the same template
- Docker and deployment:
  - `deploy/web-flask/Dockerfile`
  - `compose.web.yml`
  - `.dockerignore`
Dependency sources not found in the current checkout:

- No `pyproject.toml`, `poetry.lock`, `uv.lock`, `Pipfile`, or Python lockfile.
- No Swift Package Manager, CocoaPods, or Carthage dependency file.
- No checked-in NuGet lockfile or restored `project.assets.json`.

The npm dependency licence closure for `apps/web-vue/` should be refreshed
before distributing built frontend artefacts outside the live source
deployment.

## Repository licence status

Confirmed facts:

- `LICENSE.md` contains the GNU Affero General Public License v3 text.
- `README.md` and `README.zh-CN.md` state that the project is licensed under
  `AGPL-3.0-or-later`.
- `core/python-renderer/midi_to_wave.py` and
  `core/python-renderer/tests/test_midi_to_wave.py` include copyright and SPDX
  headers for `AGPL-3.0-or-later`.

Reasonable inferences:

- Repository-owned source, documentation, scripts, templates, localisation
  files, and project-generated assets are intended to be distributed under the
  repository AGPL licence unless a narrower file-level notice says otherwise.

Requires human or legal review:

- The root licence does not relicense third-party packages, CDN resources,
  operating-system components, Microsoft runtime components, Docker base-image
  contents, or build-tool runtime files.
- Most source files do not carry individual SPDX headers. That is not by itself
  proof of a licensing defect, but a release-grade audit should decide whether
  to add consistent per-file notices or REUSE-style metadata.

## Dependency licence summary

The repository has no lockfiles, so this is a direct-dependency and observed
metadata summary rather than a complete transitive SBOM.

### Python renderer runtime

Source: `core/python-renderer/requirements.txt`; installed metadata checked in
the repo-local virtual environment.

| Package | Version | Licence evidence | Notes |
| --- | ---: | --- | --- |
| `importlib_resources` | 6.5.2 | Apache-2.0 licence file/classifier | Runtime dependency. |
| `mido` | 1.3.3 | MIT metadata | Runtime dependency. |
| `numpy` | 2.4.3 | `BSD-3-Clause AND 0BSD AND MIT AND Zlib AND CC0-1.0` metadata | Installed wheel also carries bundled native-library notices. |
| `packaging` | 26.0 | `Apache-2.0 OR BSD-2-Clause` metadata | Also used by Gunicorn. |
| `pretty_midi` | 0.2.11 | MIT metadata | Runtime dependency. |
| `scipy` | 1.17.1 | BSD-3-Clause-style licence file/classifier | Installed wheel notice references OpenBLAS, LAPACK, GCC runtime, and libquadmath material. Review actual bundled artefacts before binary distribution. |
| `setuptools` | 82.0.1 | MIT metadata | Listed in renderer requirements. |
| `six` | 1.17.0 | MIT metadata | Runtime dependency. |

### Web Flask runtime

Source: `apps/web-flask/requirements.txt`; installed metadata checked locally
where available. `gunicorn` was not installed in the checked virtual
environment, so its licence was checked from PyPI metadata for version 23.0.0.

| Package | Version | Licence evidence | Notes |
| --- | ---: | --- | --- |
| `blinker` | 1.9.0 | MIT licence file/classifier | Flask dependency. |
| `click` | 8.3.1 | BSD-3-Clause metadata | Flask dependency. |
| `Flask` | 3.1.3 | BSD-3-Clause metadata | Direct web dependency. |
| `gunicorn` | 23.0.0 | MIT PyPI metadata | Used by the Docker/Gunicorn deployment path. |
| `itsdangerous` | 2.2.0 | BSD-3-Clause-style licence file | Flask dependency. |
| `Jinja2` | 3.1.6 | BSD-3-Clause-style licence file | Flask dependency. |
| `MarkupSafe` | 3.0.3 | BSD-3-Clause metadata | Jinja dependency. |
| `Werkzeug` | 3.1.7 | BSD-3-Clause metadata | Flask dependency. |

### macOS helper build

Source: `apps/macos/requirements-build.txt`; installed metadata checked in the
repo-local virtual environment.

| Package | Version | Licence evidence | Notes |
| --- | ---: | --- | --- |
| `altgraph` | 0.17.5 | MIT metadata | PyInstaller dependency. |
| `macholib` | 1.16.4 | MIT metadata | PyInstaller dependency. |
| `pyinstaller` | 6.19.0 | GPL-2.0-or-later with PyInstaller bootloader exception in installed licence file | Used to freeze the Python renderer into the macOS helper binary. Distribution review must inspect the generated helper and bundled files. |
| `pyinstaller-hooks-contrib` | 2026.3 | GPL-2.0-or-later for standard hooks; Apache-2.0 for runtime hooks | Relevant to generated PyInstaller helper contents and notices. |

### Windows and .NET dependencies

Source: `apps/windows/Directory.Packages.props` and project files; NuGet
package metadata checked from registry `.nuspec` files where no local NuGet
cache was present.

| Package | Version | Licence evidence | Notes |
| --- | ---: | --- | --- |
| `Melanchall.DryWetMidi` | 8.0.3 | MIT NuGet metadata | Direct dependency of the C# renderer and tests. |
| `NAudio` | 2.3.0 | MIT NuGet metadata | Direct dependency. Its `.nuspec` declares several `NAudio.*` package dependencies for runtime targets. |
| `Microsoft.WindowsAppSDK` | 1.8.260317003 | Package `license.txt`, not an SPDX expression | Direct app dependency. The app is configured for self-contained WinUI 3 publishing, so this is a release-blocking legal review item before AGPL binary redistribution. |
| `Microsoft.NET.Test.Sdk` | 18.3.0 | MIT NuGet metadata | Test dependency. Its transitive test-platform packages are not fully enumerated in this audit. |
| `xunit` | 2.9.3 | Apache-2.0 NuGet metadata | Test dependency. |
| `xunit.runner.visualstudio` | 3.1.5 | Apache-2.0 NuGet metadata | Test dependency with `PrivateAssets=all`. |

Important Windows App SDK observation:

- The checked `Microsoft.WindowsAppSDK` package licence permits redistribution
  of binplaced files under conditions, but also includes restrictions around
  subjecting Microsoft distributable code to licences that require source
  disclosure or modification rights. This audit does not decide compatibility
  with the repository's AGPL distribution goal; it flags the Windows
  self-contained app as requiring human/legal review before release.
- The package includes `NOTICE.txt` with third-party notice material, including
  `Newtonsoft.Json 13.0.1 - MIT`.
- The package depends on multiple `Microsoft.WindowsAppSDK.*` packages. The
  final restored/published package closure must be reviewed from actual release
  artefacts, not only the top-level package.

### Web CDN and font resources

Source: legacy Flask template `apps/web-flask/templates/index.html`;
registry/upstream metadata checked for the specific named resources.

| Resource | Version/source | Licence evidence | Notes |
| --- | --- | --- | --- |
| Bootstrap CSS | `bootstrap@5.3.0` from jsDelivr/npm | MIT npm metadata | Loaded from CDN at runtime; no local Bootstrap files are bundled. |
| IBM Plex Sans and IBM Plex Mono | Google Fonts CSS loading IBM-hosted font family names | SIL Open Font License 1.1 from IBM Plex upstream licence | Fonts are fetched from Google-hosted URLs at runtime; no font files are checked in. |
| Lucide inline SVG icons | Copied SVG path data for `play`, `x`, `sun`, `moon-star`, and `languages` from the Lucide icon set | ISC licence from <https://lucide.dev/license> and <https://github.com/lucide-icons/lucide/blob/main/LICENSE>; `x` is also listed as Feather-derived under MIT | Vendored inline in `apps/web-flask/static/js/lucide-icons.js`; no npm package, CDN, or build pipeline is used. |
| Hugeicons GitHub icon | Copied SVG path data for the `github` icon from <https://hugeicons.com/icon/github>; source SVG at <https://cdn.hugeicons.com/icons/github-stroke-rounded.svg> | The icon page states stroke rounded icons are free for unlimited personal and commercial use and links to the Hugeicons licence agreement at <https://hugeicons.com/license-agreement>. | Vendored inline in `apps/web-flask/static/js/lucide-icons.js`; no npm package, CDN, or build pipeline is used. |

If these resources are vendored later, include their licence texts and notices
in the repository and release artefacts. If they remain CDN resources, review
privacy, availability, and supply-chain expectations separately from licensing.

### Docker, CI, and build tools

Confirmed facts:

- The Docker image starts from `python:3.12-slim` and installs
  `apps/web-flask/requirements.txt`.
- No maintained CI workflow file for Windows release automation exists in the
  current checkout.
- `global.json` pins the .NET SDK line to `8.0.100` with `latestFeature`
  roll-forward.

Requires release review:

- A distributed Docker image includes the Python base image, Debian packages,
  Python itself, and all installed Python wheels. Its notices are not captured
  by repository files alone.
- Inno Setup is used to produce the Windows installer, but this audit did not
  inspect the generated installer binary.

## Asset and non-code material review

### Confirmed repository assets

- `assets/previews/*.wav` contains six PCM preview WAV files:
  `pulse_10.wav`, `pulse_25.wav`, `pulse_50.wav`, `sawtooth.wav`,
  `sine.wav`, and `triangle.wav`.
- `assets/README.md` states that these files are project-generated preview and
  test assets rendered by this project's own program from maintainer-directed
  MIDI test material, and that they are intended for redistribution with the
  project and app outputs.
- `apps/macos/macos/build_desktop_resources.sh` copies the canonical preview
  files into the macOS app bundle.
- The Windows app project links `assets/previews/*.wav` into build and publish
  output.
- `apps/web-flask/` serves preview WAV files from the shared asset folder.

Reasonable inferences:

- The preview WAV files are intended to be repository-owned assets covered by
  the repository licence.
- The duplicate tracked files under
  `apps/windows/src/Midi8BitSynthesiser.App/Assets/Previews/` are byte-identical
  to `assets/previews/*.wav`, but the current Windows project file links the
  canonical shared files for build and publish.

Requires human review:

- The original maintainer-directed MIDI test material used to render the
  preview WAV files is not checked in. Confirm its provenance before a formal
  release if audio provenance needs to be documented at source-material level.

### Generated documentation artefacts

Tracked generated artefacts exist under:

- `output/pdf/repo-structure-evaluation.pdf`
- `tmp/pdfs/repo-structure-evaluation.html`
- `tmp/pdfs/rendered/repo-structure-evaluation.png`

Confirmed facts:

- These files are tracked even though `output/` and `tmp/` are ignored for
  future generated files.
- The HTML/PDF content appears to be an old repository-structure evaluation and
  contains historical directory names that no longer describe the current
  layout.
- No third-party image, font, or audio file is embedded in the inspected HTML or
  PDF content.

Requires human review:

- Decide whether these generated artefacts should remain tracked. If retained,
  treat them as repository documentation and keep their provenance and licence
  status aligned with the rest of the docs.

### Copied or vendored code

The Flask web UI vendors copied Lucide SVG path data in
`apps/web-flask/static/js/lucide-icons.js`; its source and licence notices are
recorded in the file comment and in the web resource table above.

No separate vendored source tree, checked-in font file, checked-in standalone
icon file, checked-in MIDI example file, or checked-in image asset was found
outside the generated review artefacts listed above. The Vue package metadata
records npm dependencies but does not vendor their source into the repository.

## Potential licence risks or unknowns

- **Windows self-contained release:** `Microsoft.WindowsAppSDK` uses a
  Microsoft licence file rather than an SPDX open-source expression. Its terms
  and notice file must be reviewed before distributing an AGPL-licensed Windows
  zip or installer that includes self-contained Windows App SDK files.
- **No lockfiles:** Python and .NET dependency closures are not frozen by
  lockfiles. A release audit must inspect the actual resolved packages and
  artefacts for the release build.
- **Native Python wheels:** NumPy and SciPy wheels can include OpenBLAS, LAPACK,
  GCC runtime, libquadmath, and other native material with their own notices.
  Review the exact wheels shipped in Docker, macOS helpers, or other bundles.
- **PyInstaller output:** The macOS helper build uses PyInstaller and community
  hooks. The generated helper binary should be inspected for embedded runtime
  hooks, Python files, bootloader files, and third-party notices.
- **Docker image contents:** The web Docker image includes base-image and
  installed-package material not represented by repository source files alone.
- **Installer tooling:** The Inno Setup installer script was reviewed, but the
  generated installer was not.
- **External web resources:** Bootstrap and IBM Plex fonts are loaded from
  external CDNs at runtime. If vendored, their notices must be added. If kept
  remote, privacy and availability should be reviewed.
- **Preview WAV provenance:** `assets/README.md` documents maintainer
  provenance, but the input MIDI/test material is not present in the repository.
- **Tracked ignored artefacts:** `output/pdf/` and `tmp/pdfs/` contain tracked
  generated documents despite being under ignored output paths.

## Required attribution and notice obligations

For source-only distribution:

- Keep `LICENSE.md` with the source distribution.
- Preserve existing copyright and SPDX notices in files that have them.
- Do not remove third-party licence texts from dependency packages if they are
  copied into the repository in future.

For binary, installer, app bundle, or Docker image distribution:

- Include the repository AGPL licence and source-offer information appropriate
  to the distribution channel, including network-use obligations for the Flask
  app.
- Include third-party notices for all shipped Python wheels and their bundled
  native libraries, especially NumPy and SciPy.
- Include PyInstaller, PyInstaller bootloader exception, and
  pyinstaller-hooks-contrib notices when a PyInstaller-built helper is shipped.
- Include NuGet package notices for shipped Windows dependencies, especially
  `Microsoft.WindowsAppSDK` licence and `NOTICE.txt`, `NAudio`,
  `Melanchall.DryWetMidi`, and any restored transitive packages.
- Include Docker base-image, Python, Debian package, and installed wheel notices
  for distributed container images.
- Include Bootstrap MIT and IBM Plex OFL-1.1 notices if those assets are
  vendored into the repository or release artefacts.
- Include Lucide ISC notice, and the Feather MIT notice for Feather-derived
  glyphs such as `x`, when distributing source or release artefacts that contain
  the inline SVG icon helper.

## Release-readiness checklist

- [ ] Generate a release-specific dependency inventory from actual resolved
      packages, not only direct requirement files.
- [ ] Produce third-party notice files for each released artefact. For the
      current web focus, this means at least the source archive and Docker
      image; native app bundles/installers need this review before any future
      revival or redistribution.
- [ ] Review `Microsoft.WindowsAppSDK` redistribution terms and AGPL
      compatibility before publishing the self-contained Windows release.
- [ ] Inspect the generated Windows publish folder and installer for all
      included Microsoft, NuGet, and runtime components.
- [ ] Inspect the generated macOS app bundle and PyInstaller helper for embedded
      Python, native libraries, hooks, and notices.
- [ ] Inspect the Docker image package list and Python wheel set before
      publishing an image.
- [ ] Confirm preview WAV source-material provenance if formal asset provenance
      is required.
- [ ] Decide whether tracked generated artefacts under `output/` and `tmp/`
      should remain in the repository.
- [ ] Consider adding per-file SPDX headers or a REUSE-compatible copyright
      inventory for repository-owned source files.
- [ ] Re-check CDN resources, integrity, and notice strategy if the web app is
      distributed for offline or production use.

## Recommended future audit cycle

- Run a focused licence check before every public release.
- Re-run this audit whenever `requirements*.txt`, `.csproj`,
  `Directory.Packages.props`, Docker files, installer scripts, PyInstaller
  settings, or bundled assets change. If CI workflows are reintroduced later,
  include them in the next audit pass.
- For active web development, run a lightweight quarterly audit that compares
  the current dependency manifests, package metadata, and release artefacts
  against this document.
- Treat the Vue `dist` output plus Flask/Gunicorn backend as the current active
  release artefact. Treat the Docker image as a backend/fallback artefact, and
  the Windows self-contained app and macOS PyInstaller helper as separate
  release artefacts if native work is revived.
