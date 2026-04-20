# 仓库结构

Language/语言: [English](./repository-layout.md) | 简体中文

该仓库保持为一个单体仓库，目录结构是显式定义的。

## 目录结构

- `apps/web-flask/`：主要 Flask / 浏览器 UI
- `apps/macos/`：Xcode 工程、SwiftUI 应用、macOS 构建辅助工具
- `apps/windows/`：WinUI 3 解决方案、原生 C# 渲染器、安装程序、对齐测试
- `apps/desktop/`：保留占位目录
- `core/python-renderer/`：规范 Python 渲染器
- `assets/previews/`：规范预览 WAV 资源
- `docs/reviews/`：评审产物与报告

## 说明

- 平台特定的 UI、打包和发布逻辑保留在 `apps/` 下。
- 共享渲染器逻辑保留在 `core/` 下。
- 共享二进制或媒体资源保留在 `assets/` 下。
- 根目录文件保留给仓库级配置和文档使用。
