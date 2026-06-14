# Third-Party Notices

Date checked: 2026-06-14

This file records third-party software and content identified in this checkout
before archiving the public OSS mirror. It is not legal advice and does not
replace the license text that accompanies each dependency or asset.

OctaBit's own historical public source releases remain licensed under
AGPL-3.0-or-later as described in `LICENSE.md` and `COPYRIGHT.md`. Third-party
components remain governed by their own license terms.

## Dependency Sources

Current dependency inputs:

- npm: `frontend/package.json` and `frontend/package-lock.json`
- Go: `backend/go.mod` and `backend/go.sum`
- Go workspace: `go.work`

The repository does not vendor npm packages or Go modules. If a future release
ships binaries, installers, static frontend bundles, containers, or packaged
dependency trees, generate a release-specific notice bundle from the exact
artefacts being shipped.

## npm Packages

The npm inventory below was read from `frontend/package-lock.json`.

License summary:

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

Runtime npm packages:

| Package | Version | License |
| --- | --- | --- |
| `@babel/helper-string-parser` | 7.27.1 | MIT |
| `@babel/helper-validator-identifier` | 7.28.5 | MIT |
| `@babel/parser` | 7.29.3 | MIT |
| `@babel/types` | 7.29.0 | MIT |
| `@jridgewell/sourcemap-codec` | 1.5.5 | MIT |
| `@vue/compiler-core` | 3.5.34 | MIT |
| `@vue/compiler-dom` | 3.5.34 | MIT |
| `@vue/compiler-sfc` | 3.5.34 | MIT |
| `@vue/compiler-ssr` | 3.5.34 | MIT |
| `@vue/devtools-api` | 6.6.4 | MIT |
| `@vue/reactivity` | 3.5.34 | MIT |
| `@vue/runtime-core` | 3.5.34 | MIT |
| `@vue/runtime-dom` | 3.5.34 | MIT |
| `@vue/server-renderer` | 3.5.34 | MIT |
| `@vue/shared` | 3.5.34 | MIT |
| `csstype` | 3.2.3 | MIT |
| `entities` | 7.0.1 | BSD-2-Clause |
| `estree-walker` | 2.0.2 | MIT |
| `magic-string` | 0.30.21 | MIT |
| `nanoid` | 3.3.12 | MIT |
| `picocolors` | 1.1.1 | ISC |
| `postcss` | 8.5.15 | MIT |
| `source-map-js` | 1.2.1 | BSD-3-Clause |
| `vue` | 3.5.34 | MIT |
| `vue-router` | 4.6.4 | MIT |

Notable build, test, optional, or data packages with non-MIT/ISC/BSD terms:

| Package(s) | License | Notes |
| --- | --- | --- |
| `typescript` | Apache-2.0 | Build/type-check dependency. |
| `detect-libc`, `expect-type`, `xml-name-validator` | Apache-2.0 | Development/test dependencies. |
| `glob`, `lru-cache`, `minipass`, `path-scurry`, `glob/node_modules/minimatch` | BlueOak-1.0.0 | Development dependencies. |
| `lightningcss` and platform packages | MPL-2.0 | Build dependency. Preserve MPL notices and source availability for any modified MPL-covered files. |
| `mdn-data` | CC0-1.0 | Development dependency data. |
| `tslib` | 0BSD | Optional dependency. |
| `@csstools/color-helpers`, `@csstools/css-syntax-patches-for-csstree` | MIT-0 | Development dependencies. |

No npm package license metadata in the current lockfile was identified as GPL,
AGPL, LGPL, SSPL, BUSL, or non-commercial.

## Go Modules

The Go inventory below was read from `backend/go.mod`, `backend/go.sum`, and
local module license files resolved by `go list -m -json all`.

| Module | Version | License evidence |
| --- | --- | --- |
| `github.com/dustin/go-humanize` | v1.0.1 | MIT-like |
| `github.com/google/pprof` | v0.0.0-20250317173921-a4b03ec1a45e | Apache-2.0 |
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause-like |
| `github.com/hashicorp/golang-lru/v2` | v2.0.7 | MPL-2.0-like |
| `github.com/mattn/go-isatty` | v0.0.20 | MIT-like |
| `github.com/ncruces/go-strftime` | v1.0.0 | MIT-like |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause-like |
| `gitlab.com/gomidi/midi/v2` | v2.3.23 | MIT-like |
| `golang.org/x/mod` | v0.33.0 | BSD-3-Clause-like; PATENTS file present |
| `golang.org/x/sync` | v0.20.0 | BSD-3-Clause-like; PATENTS file present |
| `golang.org/x/sys` | v0.42.0 | BSD-3-Clause-like; PATENTS file present |
| `golang.org/x/tools` | v0.42.0 | BSD-3-Clause-like; PATENTS file present |
| `modernc.org/cc/v4` | v4.28.2 | BSD-3-Clause-like |
| `modernc.org/ccgo/v4` | v4.34.0 | BSD-3-Clause-like |
| `modernc.org/fileutil` | v1.4.0 | BSD-3-Clause-like |
| `modernc.org/gc/v2` | v2.6.5 | BSD-3-Clause-like |
| `modernc.org/gc/v3` | v3.1.2 | BSD-3-Clause-like |
| `modernc.org/goabi0` | v0.2.0 | BSD-3-Clause-like |
| `modernc.org/libc` | v1.72.3 | BSD-3-Clause-like |
| `modernc.org/mathutil` | v1.7.1 | BSD-3-Clause-like |
| `modernc.org/memory` | v1.11.0 | BSD-3-Clause-like |
| `modernc.org/opt` | v0.2.0 | BSD-3-Clause-like |
| `modernc.org/sortutil` | v1.2.1 | BSD-3-Clause-like |
| `modernc.org/sqlite` | v1.50.1 | BSD-3-Clause-like |
| `modernc.org/strutil` | v1.2.1 | BSD-3-Clause-like |
| `modernc.org/token` | v1.1.0 | BSD-3-Clause-like |

No Go module license evidence in the current resolved module graph was
identified as GPL, AGPL, LGPL, SSPL, BUSL, or non-commercial.

## Icons

`frontend/src/icons.ts` embeds inline SVG path data used by the web UI.

Lucide icons:

- Source project: Lucide
- License: ISC License
- Copyright notice: Copyright (c) 2026 Lucide Icons and Contributors
- Some Lucide icons are derived from Feather.

Feather-derived Lucide icons:

- Source project: Feather
- License: MIT License
- Copyright notice: Copyright (c) 2013-present Cole Bemis

Hugeicons GitHub icon:

- Source project: Hugeicons free icon set
- License evidence: `@hugeicons/core-free-icons` npm metadata reports MIT
- Attribution: Created by Hugeicons

The repository currently embeds only selected SVG path data, not the full icon
packages. Preserve these notices when redistributing built frontend artefacts or
replace the copied path data with project-owned icons before distribution.

## Fonts

`frontend/index.html` loads IBM Plex Sans and IBM Plex Mono from Google Fonts.

- Font family: IBM Plex
- Copyright notice: Copyright (c) 2017 IBM Corp. with Reserved Font Name "Plex"
- License: SIL Open Font License 1.1

The current source does not vendor font files. If future builds bundle fonts,
ship the OFL notice with the bundled font files and do not relicense the font
software under the proprietary project license.

## Bundled Project Assets and Fixtures

The repository contains project assets and test fixtures:

- `assets/previews/*.wav`
- `assets/icons/**`
- `assets/readme/octabit-readme-header.png`
- `frontend/public/*` favicon and web manifest assets
- `backend/testdata/python-baseline/**`

`assets/README.md` documents the waveform preview WAV files as project-generated
preview/test assets rendered by this project's own program from
maintainer-directed MIDI test material. The backend Python baseline README
describes the MIDI/WAV fixtures as parity inputs for Go conformance tests.

No separate third-party license was found in the repository for these bundled
assets or fixtures. Before a proprietary release, confirm and record source
provenance for icon artwork, favicon/header images, MIDI fixtures, and rendered
WAV fixtures.

## Legal and Documentation Templates

`CLA.md` states that the OctaBit Individual Contributor License Agreement is
adapted from the Apache Software Foundation Individual Contributor License
Agreement v2.2. Preserve that attribution if the CLA remains in archived public
materials or is reused in future private repository documentation.

## Historical Public Releases

Historical public releases and tags that were distributed under
AGPL-3.0-or-later remain governed by AGPL-3.0-or-later. Do not remove AGPL
license text, copyright notices, or corresponding-source information for those
historical releases.

## Future Proprietary Releases

A future private proprietary repository may apply a proprietary license to
repository-owned code only if the copyright holder has the necessary rights to
all included contributions. Third-party components listed here, and any future
third-party components, remain governed by their own license terms.
