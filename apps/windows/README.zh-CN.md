# Windows 应用

Language/语言: [English](./README.md) | 简体中文

此目录包含 MIDI-8bit Synthesiser 的原生 Windows 桌面重写版本。

## 职责

- 面向 Windows 的 WinUI 3 桌面界面
- 原生队列、层编辑、预览与导出工作流
- 通过 Python 参考渲染器校验的 C# 原生渲染器
- 用于生成可移植 `win-x64` 发布包和安装程序的 Windows CI 发布流水线

## 项目结构

- `src/Midi8BitSynthesiser.Core/`：渲染引擎、波形模型、输出命名
- `src/Midi8BitSynthesiser.App/`：WinUI 3 外壳、文件对话框集成、预览播放
- `tests/Midi8BitSynthesiser.Tests/`：单元测试、工作流测试、Python 对齐测试

## 在 Windows 上构建

在仓库根目录执行：

1. 安装 .NET 8 SDK，以及 WinUI 3 桌面开发所需的 Visual Studio 组件。
2. 安装 Python 3，并安装参考渲染器依赖：`python -m pip install -r core/python-renderer/requirements.txt`
3. 还原、构建并测试：
   - `dotnet restore apps/windows/Midi8BitSynthesiser.sln`
   - `dotnet build apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64`
   - `dotnet test apps/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build`
4. 发布可移植包：`dotnet publish apps/windows/src/Midi8BitSynthesiser.App/Midi8BitSynthesiser.App.csproj -c Release -r win-x64 --self-contained true -p:Platform=x64`

发布目录包含主 `.exe`、运行时文件，以及从 `assets/previews/` 链接进来的预览 WAV 资源。

## 面向最终用户的运行要求

已发布的 Windows 版本为自包含发布。

最终用户需要：

- 受支持的 64 位 Windows 安装
- 来自可移植 zip 包或安装程序的已发布应用文件

最终用户不需要：

- .NET SDK
- 本地源码检出
- Python

## 面向开发者和评审者的构建要求

构建、测试和发布仍然需要：

- .NET 8 SDK
- 与 WinUI 3 兼容的 Visual Studio 组件
- Python 3
- 安装 `core/python-renderer/requirements.txt` 以运行对齐测试

## 评审前检查

在报告 Windows 构建或运行时故障之前，请先确认评审机器确实具备验证该应用的条件：

- `dotnet --info`
- `python --version`
- `python -c "import pretty_midi, numpy, scipy"`

详细检查清单位于 `REVIEWING.md`。

## 评审包

要为外部 Windows 评审准备一个打包文件，请运行：

```bash
apps/windows/scripts/create_review_bundle.sh
```

该打包文件包含：

- `apps/windows/`
- `core/python-renderer/`
- `assets/previews/`
- `.github/workflows/windows-release.yml`
- `global.json`

## 安装程序与可移植发布

Windows 版本以两种形式发布：

- 用于手动分发和评审的自包含可移植 zip 包
- 面向普通最终用户的 Inno Setup 安装程序

两者都基于相同的已发布 `win-x64` 输出构建。
