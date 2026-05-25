# 共享资源

Language/语言: [English](./README.md) | 简体中文

`assets/previews/` 是波形预览 WAV 文件的规范来源。

## 用途

- `legacy/web-flask/` 通过专用 Flask 路由提供这些文件。
- 保留的 `legacy/native/macos/` 代码会在构建时将这些文件复制到应用包中。
- 保留的 `legacy/native/windows/` 代码会在构建和发布时将这些文件链接到 WinUI 工程中。

## 预览资源来源

`assets/previews/` 中的预览 WAV 文件是项目生成的预览/测试资源，用于试听波形和音色。它们由本项目自己的程序根据维护者指定的 MIDI 测试素材渲染生成。据维护者所知，它们并非来自第三方采样包或外部授权录音，并且预期可随本项目及其应用输出一起重新分发。
