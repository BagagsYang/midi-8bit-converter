# macOS 应用

Language/语言: [English](./README.md) | 简体中文

此目录包含原生 macOS 应用的打包说明和构建说明。

## 构建

1. 安装完整的 Xcode。
2. 重新创建或刷新仓库本地虚拟环境：`python3 -m venv .venv`
3. 安装 macOS 构建依赖：`./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt`
4. 打开 `apps/macos/MIDI8BitSynthesiser.xcodeproj` 并运行 `MIDI8BitSynthesiser` scheme。

## 应用工作方式

- SwiftUI 提供原生 macOS 界面。
- Xcode 构建阶段会运行 `apps/macos/macos/build_desktop_resources.sh`。
- 该脚本会使用 PyInstaller 将 `core/python-renderer/midi_to_wave.py` 冻结为随应用打包的辅助二进制。
- 同一脚本还会将 `assets/previews/` 中的规范预览 WAV 资源复制到应用包中。
- 应用会针对队列中的每个 MIDI 文件直接启动打包后的辅助程序，因此不会涉及 Flask 服务器或浏览器。
