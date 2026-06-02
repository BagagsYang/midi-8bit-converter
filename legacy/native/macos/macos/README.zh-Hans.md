# macOS 应用

Language/语言: [English](./README.md) | 简体中文

此目录包含原生 macOS 应用的打包说明和构建说明。

## Deprecated/paused 状态

此原生 macOS 应用已 deprecated/paused。它不是主要开发目标；项目当前聚焦于 Web 服务。代码保留用于参考。

## 构建

1. 安装完整的 Xcode。
2. 重新创建或刷新仓库本地虚拟环境：`python3 -m venv .venv`
3. 安装 macOS 构建依赖：`./.venv/bin/python3 -m pip install -r legacy/native/macos/requirements-build.txt`
4. 打开 `legacy/native/macos/MIDI8BitSynthesiser.xcodeproj` 并运行 `MIDI8BitSynthesiser` scheme。

## 启动

- 通过 Xcode、Finder 或 `open -na <path-to-app>` 启动构建后的 `.app` 应用包。
- 手动测试时不要直接执行 `MIDI8BitSynthesiser.app/Contents/MacOS/MIDI8BitSynthesiser`。在较新的 macOS 版本中，该路径可能会在 AppKit/HIServices 内部中止，导致误导性的崩溃报告，即使正常的应用包启动路径可以工作。

## 应用工作方式

- SwiftUI 提供原生 macOS 界面。
- Xcode 构建阶段会运行 `legacy/native/macos/macos/build_desktop_resources.sh`。
- 该脚本会使用 PyInstaller 将 `legacy/python-renderer/midi_to_wave.py` 冻结为随应用打包的辅助二进制。
- 同一脚本还会将 `assets/previews/` 中的规范预览 WAV 资源复制到应用包中。
- 应用会针对队列中的每个 MIDI 文件直接启动打包后的辅助程序，因此不会涉及 Flask 服务器或浏览器。

## 当前层控制

- 每一层仍包含波形、脉冲宽度和基础音量控制。
- 每一层现在可以选择启用频率增益曲线；导出时，该曲线会根据每个音符的基频进行计算。
- 内联曲线编辑器使用对数频率 x 轴和 dB y 轴，每层最多支持 8 个可拖动控制点。
- 导出命名与当前 Web/core 行为一致：单层导出保留波形后缀，多层导出使用 `_mix`，带曲线的导出会追加稳定的哈希后缀。
- 现有预览按钮仍然是原始波形预览；它尚不会将频率曲线渲染进预览声音。

## 测试

- 共享的 `MIDI8BitSynthesiser` scheme 现在包含一个轻量级 `MIDI8BitSynthesiserTests` XCTest 目标，用于纯模型、载荷和文件名逻辑。
