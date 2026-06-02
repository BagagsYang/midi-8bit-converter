# Shared assets

Language/语言: English | [简体中文](./README.zh-Hans.md)

`assets/previews/` is the canonical source of waveform preview WAV files.

## Usage

- `legacy/web-flask/` serves these files through a dedicated Flask route.
- The retained `legacy/native/macos/` code copies these files into the app bundle at build time.
- The retained `legacy/native/windows/` code links these files into the WinUI project at build and publish time.

## Preview asset provenance

The preview WAV files in `assets/previews/` are project-generated preview/test assets created for waveform and timbre auditioning. They were rendered by this project's own program from maintainer-directed MIDI test material. To the maintainer's knowledge, they are not derived from third-party sample packs or externally licensed audio recordings, and they are intended to be redistributed with the project and its app outputs.
