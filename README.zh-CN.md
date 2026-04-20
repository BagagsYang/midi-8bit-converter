# MIDI-8bit Synthesiser

Language/语言: [English](./README.md) | 简体中文

该仓库是为 MIDI-8bit Synthesiser 产品族重新整理后的单体仓库。各平台应用位于 `apps/` 下，Python 参考渲染器位于 `core/` 下，共享预览资源位于 `assets/` 下。

## 目录结构

| 目录 | 职责 |
| --- | --- |
| `apps/web-flask/` | 旧版 Flask / 浏览器 UI |
| `apps/macos/` | 原生 macOS SwiftUI 应用和 Xcode 工程 |
| `apps/windows/` | 原生 Windows WinUI 3 解决方案、C# 渲染器、安装程序 |
| `apps/desktop/` | 为未来桌面打包工作保留的占位目录 |
| `core/python-renderer/` | 规范 Python MIDI 转 WAV 渲染器与对齐参考实现 |
| `assets/previews/` | 所有应用共用的规范波形预览 WAV 资源 |
| `docs/` | 评审记录与仓库结构说明 |

## 共享渲染器

- 规范渲染器入口：`core/python-renderer/midi_to_wave.py`
- 稳定输入：MIDI 路径、输出 WAV 路径、采样率、波形层
- 稳定输出：渲染后的 WAV 文件，或明确的错误信息
- Windows 有意保留原生 C# 实现，并通过对齐测试与 Python 渲染器进行校验

## 构建说明

在仓库根目录创建仓库本地环境：

```bash
python3 -m venv .venv
```

仅安装你当前处理的应用所需依赖：

- Web UI：`./.venv/bin/python3 -m pip install -r apps/web-flask/requirements.txt`
- macOS 辅助构建：`./.venv/bin/python3 -m pip install -r apps/macos/requirements-build.txt`
- Windows 对齐测试：`./.venv/bin/python3 -m pip install -r core/python-renderer/requirements.txt`

各应用的专用说明位于：

- `apps/web-flask/README.md`
- `apps/macos/macos/README.md`
- `apps/windows/README.md`

仓库结构说明位于 `docs/repository-layout.md`。

## 许可证

本项目采用 GNU Affero General Public License v3.0 或更新版本（`AGPL-3.0-or-later`）授权。完整详情请参阅 [LICENSE](LICENSE.md) 文件。
