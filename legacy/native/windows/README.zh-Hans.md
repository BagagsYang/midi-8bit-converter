# Windows 应用

Language/语言: [English](./README.md) | 简体中文

此目录包含 OctaBit 保留的原生 Windows 桌面重写版本。

## Deprecated/paused 状态

此原生 Windows 应用已 deprecated/paused。它不是主要开发目标；项目当前聚焦于 Web 服务。代码保留用于参考。

## 保留的职责

- 面向 Windows 的 WinUI 3 桌面界面
- 原生队列、层编辑、预览与导出工作流
- 通过 Python 参考渲染器校验的 C# 原生渲染器

## 项目结构

- `src/Midi8BitSynthesiser.Core/`：渲染引擎、波形模型、输出命名
- `src/Midi8BitSynthesiser.App/`：WinUI 3 外壳、文件对话框集成、预览播放
- `tests/Midi8BitSynthesiser.Tests/`：单元测试、工作流测试、Python 对齐测试

## 在 Windows 上构建

在仓库根目录执行：

1. 安装 .NET 8 SDK，以及 WinUI 3 桌面开发所需的 Visual Studio 组件。
2. 安装 Python 3，并安装参考渲染器依赖：`python -m pip install -r legacy/python-renderer/requirements.txt`
3. 还原、构建并测试：
   - `dotnet restore legacy/native/windows/Midi8BitSynthesiser.sln`
   - `dotnet build legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64`
   - `dotnet test legacy/native/windows/Midi8BitSynthesiser.sln -c Release -p:Platform=x64 --no-build`
此仓库中已不再保留受维护的 Windows 发布工作流或 CI 发布流水线。若未来恢复原生打包工作，应基于当前工程文件和安装程序文件重新建立发布步骤，而不是依赖已经移除的工作流。

## 面向开发者和评审者的构建要求

构建和测试仍然需要：

- .NET 8 SDK
- 与 WinUI 3 兼容的 Visual Studio 组件
- Python 3
- 安装 `legacy/python-renderer/requirements.txt` 以运行对齐测试

## 评审前检查

在报告 Windows 构建或运行时故障之前，请先确认评审机器确实具备验证该应用的条件：

- `dotnet --info`
- `python --version`
- `python -c "import pretty_midi, numpy, scipy"`

详细检查清单位于 `REVIEWING.md`。

## 评审包

要为外部 Windows 评审准备一个打包文件，请运行：

```bash
legacy/native/windows/scripts/create_review_bundle.sh
```

该打包文件包含：

- `legacy/native/windows/`
- `legacy/python-renderer/`
- `assets/previews/`
- `global.json`

历史安装程序文件仍保留在 `installer/` 下，但当前仓库不存在受维护的仓库级 Windows 发布工作流。
