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
Go backend service.

Reviewed repository areas:

- Root documentation and licence files: `README.md`, `README.zh-Hans.md`,
  `LICENSE.md`, `AGENTS.md`, `.gitattributes`, `.gitignore`.
- Current app target code: `frontend/`, `backend/`.
- Shared code and assets: `assets/previews/`.
- Documentation and generated review artefacts: `docs/`, tracked files under
  `output/pdf/`, and tracked files under `tmp/pdfs/`.
Dependency and packaging sources found:

- Go backend:
  - `backend/go.mod`
  - `backend/go.sum`
- Web runtime resources:
  - Vue frontend package metadata in `frontend/package.json` and
    `frontend/package-lock.json`

Dependency sources not found in the current checkout:

- No lockfiles for backend or frontend dependencies.

The npm dependency licence closure for `frontend/` should be refreshed
before distributing built frontend artefacts outside the live source
deployment.

## Repository licence status

Confirmed facts:

- `LICENSE.md` contains the GNU Affero General Public License v3 text.
- `README.md` and `README.zh-Hans.md` state that the project is licensed under
  `AGPL-3.0-or-later`.

Reasonable inferences:

- Repository-owned source, documentation, scripts, templates, localisation
  files, and project-generated assets are intended to be distributed under the
  repository AGPL licence unless a narrower file-level notice says otherwise.

Requires human or legal review:

- The root licence does not relicense third-party packages or build-tool
  runtime files.
- Most source files do not carry individual SPDX headers. That is not by itself
  proof of a licensing defect, but a release-grade audit should decide whether
  to add consistent per-file notices or REUSE-style metadata.

## Dependency licence summary

The repository has no lockfiles, so this is a direct-dependency and observed
metadata summary rather than a complete transitive SBOM.

### Go backend

Source: `backend/go.mod`; resolved through the Go module proxy.

| Package | Licence evidence | Notes |
| --- | --- | --- |
| `gitlab.com/gomidi/midi/v2` | MIT | MIDI file parsing. |
| `modernc.org/sqlite` | BSD-3-Clause | Pure Go SQLite driver. |

### Vue frontend

Source: `frontend/package.json` and `frontend/package-lock.json`. The npm
dependency licence closure should be refreshed before distributing built
frontend artefacts outside the live source deployment.

## Asset and non-code material review

### Confirmed repository assets

- `assets/previews/*.wav` contains six PCM preview WAV files:
  `pulse_10.wav`, `pulse_25.wav`, `pulse_50.wav`, `sawtooth.wav`,
  `sine.wav`, and `triangle.wav`.
- `assets/README.md` states that these files are project-generated preview and
  test assets rendered by this project's own program from maintainer-directed
  MIDI test material, and that they are intended for redistribution with the
  project and app outputs.

Reasonable inferences:

- The preview WAV files are intended to be repository-owned assets covered by
  the repository licence.

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

## Potential licence risks or unknowns

- **No lockfiles:** A release audit must inspect the actual resolved packages
  and artefacts for the release build.
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
  to the distribution channel.

## Release-readiness checklist

- [ ] Generate a release-specific dependency inventory from actual resolved
      packages, not only direct requirement files.
- [ ] Confirm preview WAV source-material provenance if formal asset provenance
      is required.
- [ ] Decide whether tracked generated artefacts under `output/` and `tmp/`
      should remain in the repository.
- [ ] Consider adding per-file SPDX headers or a REUSE-compatible copyright
      inventory for repository-owned source files.

## Recommended future audit cycle

- Run a focused licence check before every public release.
- Re-run this audit whenever `go.mod`, Docker files, or bundled assets change.
- For active web development, run a lightweight quarterly audit that compares
  the current dependency manifests, package metadata, and release artefacts
  against this document.
- Treat the Vue `dist` output plus Go backend as the current active release
  artefact.
