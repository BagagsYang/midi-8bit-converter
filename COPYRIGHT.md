# Copyright and Licensing Notice

Copyright (c) 2025-2026 Yang Yi. All rights reserved.

This repository records the transition from a public AGPL-licensed OSS mirror
to future private proprietary development. It preserves the license status of
historical public releases and documents the proprietary notice intended for
future private work.

---

## Historical Public AGPL Releases

Historical public releases and source snapshots of OctaBit that were published
under GNU Affero General Public License, version 3.0 or any later version
published by the Free Software Foundation, remain licensed under
AGPL-3.0-or-later. The full license text is in `LICENSE.md`.

```
backend/
frontend/
assets/
docs/
deploy/production/
CLA-signatures/
CONTRIBUTING.md
CONTRIBUTING.zh-Hans.md
```

SPDX-License-Identifier: `AGPL-3.0-or-later`

Existing recipients keep the AGPL rights they received with those historical
public versions. This notice does not revoke or narrow rights previously granted
under AGPL-3.0-or-later.

---

## Future Proprietary Development

Future development of OctaBit and OctaBit Pro may continue in a private
proprietary codebase. Repository-owned code and content in that future private
codebase may be distributed under proprietary terms when the copyright holder
has the necessary rights to all included contributions.

The current Pro/private-only directories in this monorepo are proprietary
software. No license is granted for use, copying, modification, or distribution
of these files except under a separate written agreement with the copyright
holder.

```
overlays/
scripts/pro/
deploy/pro/
```

SPDX-License-Identifier: `LicenseRef-Proprietary`

---

## Public Source Availability

Under the terms of the AGPL-3.0, the complete corresponding source code for the
historical open-source portions of OctaBit was made available at:

<https://github.com/bagags/octabit>

If that public mirror is archived, archive status does not alter the license
terms of versions already published under AGPL-3.0-or-later.

---

## Third-Party Components

This repository may include third-party software components governed by their
own license terms. Changing the project license or moving future development to
a proprietary repository does not relicense third-party components.

See `THIRD_PARTY_NOTICES.md`, `LICENSE.md`, dependency manifests, lockfiles, and
individual source files for details.

---

## Contributions

The public OSS mirror is not an open contribution target. Historical invited
contributions to AGPL-covered public areas required a signed Contributor License
Agreement (`CLA.md`) and were licensed under AGPL-3.0-or-later unless another
written agreement applied.

Contributions to the proprietary directories are not accepted from external
parties.

---

## Proprietary Notice Template

The following template is intended for future private proprietary repositories
or proprietary distribution packages. It does not replace `LICENSE.md` in this
archived public mirror.

```text
Copyright (c) 2025-2026 Yang Yi. All rights reserved.

This software and associated documentation are proprietary and confidential.
No permission is granted to use, copy, modify, distribute, sublicense, or
create derivative works except under a separate written agreement with the
copyright holder.

Third-party components included with or used by this software remain governed
by their own license terms. See THIRD_PARTY_NOTICES.md.

Historical public releases of OctaBit made available under AGPL-3.0-or-later
remain governed by the license terms that accompanied those releases.
```
