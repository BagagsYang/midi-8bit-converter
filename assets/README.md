# Shared Assets

`assets/previews/` is the canonical source of waveform preview WAV files.

Usage:

- `apps/web-flask/` serves these files through a dedicated Flask route.
- `apps/macos/` copies these files into the app bundle at build time.
- `apps/windows/` links these files into the WinUI project at build and publish time.

## Preview Asset Provenance

The files under `assets/previews/` are project-specific preview/test assets used
to let users audition waveform and timbre output.

Their documented workflow is:

- the underlying MIDI test material was generated with LLM assistance based on
  the maintainer's prompts
- those MIDI files were then rendered into the preview WAV files by this
  repository's own program

The WAV files themselves were therefore not directly generated audio from an
LLM. To the maintainer's knowledge, they are not derived from third-party sample
packs or externally licensed audio recordings.

These files are intended to be redistributed with the repository and app outputs
as project preview assets.
