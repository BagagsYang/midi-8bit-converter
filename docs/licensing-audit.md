# Licensing Audit

Date checked: 2026-06-14

This is a practical repository audit, not legal advice. It records facts from
the current checkout, git history, dependency manifests, lockfiles, local module
license metadata, and repository assets. It is intended to support archiving the
public OSS mirror while preserving the obligations attached to historical public
AGPL releases.

## Executive Conclusion

Likely OK to move future development to a private proprietary codebase if Yang
Yi is the sole copyright holder, or otherwise has relicensing rights to all
accepted contributions. The current checkout does not show an obvious external
contributor in git history and does not show GPL, AGPL, LGPL, SSPL, BUSL, or
non-commercial dependency licenses in the npm lockfile or Go module graph.

The main practical obligations before closed-source redistribution are notice
and provenance work:

- Keep `LICENSE.md` and historical AGPL notices for old public releases.
- Ship third-party notices for dependencies, icons, fonts, and bundled assets.
- Confirm source provenance for icon artwork, favicon/header images, MIDI
  fixtures, and generated WAV fixtures.
- Generate a release-specific dependency license report for any binary,
  installer, static frontend bundle, container, or dependency tree that is
  distributed.

## Scope Reviewed

Reviewed repository areas:

- Root legal and policy files: `LICENSE.md`, `COPYRIGHT.md`, `README.md`,
  `README.zh-Hans.md`, `CONTRIBUTING.md`, `CLA.md`, `CLA-signatures/README.md`,
  `AGENTS.md`, and `CLAUDE.md`.
- Frontend dependency inputs: `frontend/package.json` and
  `frontend/package-lock.json`.
- Backend dependency inputs: `backend/go.mod` and `backend/go.sum`.
- Go workspace input: `go.work`.
- Application-owned assets and fixtures under `assets/`, `frontend/public/`,
  and `backend/testdata/python-baseline/`.
- Inline icon definitions in `frontend/src/icons.ts`.
- Legal template attribution in `CLA.md`.

Ignored/generated dependency folders such as `frontend/node_modules/` and
`dist/pro/**` were not treated as repository-owned source. They were inspected
only as local evidence where useful.

## Repository License Status

Confirmed facts:

- `LICENSE.md` contains the GNU Affero General Public License version 3 text.
- `README.md` and `README.zh-Hans.md` state that the public project is licensed
  under `AGPL-3.0-or-later`.
- `COPYRIGHT.md` states `Copyright (c) 2025-2026 Yang Yi. All rights reserved.`
- `COPYRIGHT.md` already distinguishes current AGPL public areas from
  proprietary overlay areas.
- The public mirror language states that `bagags/octabit` is an OSS mirror from
  a private upstream monorepo and is not an open contribution target.

Practical conclusion:

- Historical public releases distributed as AGPL-3.0-or-later cannot be
  retroactively withdrawn from existing recipients.
- Future private development can use a proprietary license for repository-owned
  code only if all relevant copyright and contribution rights are controlled by
  the private project owner.
- Third-party components are not relicensed by changing the repository license.

## Copyright and Contributor Provenance

`git shortlog -sne --all` output at the time of this audit:

```text
   171  bagags <the.yeshi.studio@gmail.com>
    17  BagagsYang <the.yeshi.studio@gmail.com>
     9  BagagsYang <168657912+BagagsYang@users.noreply.github.com>
     5  Yi Yang <168657912+bagags@users.noreply.github.com>
     1  Yi Yang <168657912+BagagsYang@users.noreply.github.com>
```

No tracked `CLA-signatures/*.json` files were present. If all listed author
identities are Yang Yi-controlled identities, the local history supports the
single-author/relicensing assumption. If any identity represents another person
or organization, obtain an assignment, CLA, or written relicensing permission
before closing future development.

## Dependency License Summary

### npm

Source: `frontend/package-lock.json`.

Current lockfile package count: 165.

| License | Package count |
| --- | ---: |
| MIT | 126 |
| MIT-0 | 2 |
| ISC | 9 |
| Apache-2.0 | 4 |
| BSD-2-Clause | 3 |
| BSD-3-Clause | 2 |
| BlueOak-1.0.0 | 5 |
| MPL-2.0 | 12 |
| CC0-1.0 | 1 |
| 0BSD | 1 |

Runtime packages are MIT, BSD-2-Clause, BSD-3-Clause, or ISC. The MPL-2.0 npm
packages are `lightningcss` and platform packages used as build dependencies.

No npm package license metadata in the current lockfile was identified as GPL,
AGPL, LGPL, SSPL, BUSL, or non-commercial.

### Go

Source: `backend/go.mod`, `backend/go.sum`, and local module license files
resolved by `go list -m -json all`.

Direct Go modules:

| Module | Version | License evidence |
| --- | --- | --- |
| `gitlab.com/gomidi/midi/v2` | v2.3.23 | MIT-like |
| `modernc.org/sqlite` | v1.50.1 | BSD-3-Clause-like |

Notable transitive module license evidence:

- `github.com/hashicorp/golang-lru/v2` is MPL-2.0-like.
- `github.com/google/pprof` is Apache-2.0.
- `golang.org/x/*` modules include BSD-3-Clause-like license files and PATENTS
  files.
- `modernc.org/*` transitive modules inspected in the local module cache are
  BSD-3-Clause-like.

No Go module license evidence in the current resolved module graph was
identified as GPL, AGPL, LGPL, SSPL, BUSL, or non-commercial.

## Third-Party Code and Materials

### Inline icons

`frontend/src/icons.ts` embeds selected SVG path data. The code labels most
icons as Lucide-style icons and the GitHub icon path as Hugeicons.

- Lucide: ISC License; Copyright (c) 2026 Lucide Icons and Contributors.
- Feather-derived Lucide icons: MIT License; Copyright (c) 2013-present Cole
  Bemis.
- Hugeicons free icon package metadata reports MIT, with attribution to
  Hugeicons.

This is a notice/provenance item, not a closed-source blocker. Preserve notices
or replace the copied icon paths with project-owned artwork before
distribution.

### Fonts

`frontend/index.html` loads IBM Plex Sans and IBM Plex Mono from Google Fonts.
IBM Plex is licensed under the SIL Open Font License 1.1 with copyright notice:
`Copyright (c) 2017 IBM Corp. with Reserved Font Name "Plex"`.

The repository does not currently vendor IBM Plex font files. If future builds
bundle font files, include the OFL text and do not relicense the font software.

### Assets and fixtures

Confirmed tracked assets and fixtures include:

- `assets/previews/*.wav`
- `assets/icons/**`
- `assets/readme/octabit-readme-header.png`
- `frontend/public/*`
- `backend/testdata/python-baseline/**`

`assets/README.md` documents the preview WAVs as project-generated preview/test
assets rendered by this project's own program from maintainer-directed MIDI test
material. `backend/testdata/python-baseline/README.md` documents the backend
MIDI/WAV fixtures as parity fixtures for Go conformance tests.

No separate third-party asset license file was found for these bundled assets.
Before proprietary distribution, confirm and record the source provenance of the
icon artwork, favicon/header images, MIDI fixtures, and generated WAV fixtures.

### Legal and documentation templates

`CLA.md` states that the OctaBit Individual Contributor License Agreement is
adapted from the Apache Software Foundation Individual Contributor License
Agreement v2.2. Preserve that attribution if the CLA remains in archived public
materials or is reused in future private repository documentation.

### External project copying

Searches for `ryohey`, `signal`, copying/adaptation markers, and related
license terms did not find evidence that current repository code was copied from
`ryohey/signal`. Generic `signal` hits were ordinary Go signal handling or
product prose. This does not prove no design inspiration occurred; it means no
copied-source marker was found in the current checkout or git history.

## Historical AGPL Obligations

Historical public releases and tags distributed under AGPL-3.0-or-later remain
available to recipients under that license. Existing recipients retain the AGPL
rights they already received.

For historical AGPL versions:

- Keep the AGPL license text available.
- Keep copyright and license notices intact.
- Keep corresponding-source information available for public network service
  deployments of AGPL-covered versions.
- Do not describe future proprietary relicensing as revoking rights previously
  granted for old public versions.

Known tags in this checkout include `v1.0.0`, `v2.0.0` through `v2.4.0`,
`v3.0.0`, and Pro-family tags `pro-v0.1.0`, `pro-v1.0.0`, and `pro-v1.1.0`.

## Required Actions Before Going Closed-Source

- Add and maintain `THIRD_PARTY_NOTICES.md`.
- Keep `LICENSE.md` in this archived public mirror.
- Update README and copyright notices to say that the public mirror is archived
  while historical AGPL releases remain under AGPL-3.0-or-later.
- Add a proprietary notice template for the future private repository without
  replacing the public mirror's AGPL license.
- Generate a release-specific dependency license report for shipped artefacts,
  not just source manifests.
- Preserve license texts for npm and Go dependencies in any distributed package
  that includes those dependencies.
- Confirm and record provenance for bundled visual/audio/MIDI assets and
  fixtures.

## Residual Risk

Low-to-medium. The dependency graph is compatible with closed-source
development based on the inspected metadata, but the repository still needs
release-grade attribution and asset provenance records. The strongest gating
fact is contributor ownership: future proprietary development is cleanest only
if Yang Yi owns, or has relicensing rights to, all project-owned code and
content.
