# Python 渲染器

Language/语言: [English](./README.md) | 简体中文

此目录包含规范 Python MIDI 转 WAV 渲染器。Flask 应用会直接使用它，保留的 macOS 辅助构建会直接使用它，保留的 Windows 对齐测试会间接使用它。

## 公共接口

- 模块：`midi_to_wave.py`
- 主要函数：`midi_to_audio(midi_path, output_path, sample_rate=48000, layers=None)`
- CLI 位置参数：输入 MIDI 路径、输出 WAV 路径
- CLI 选项：`--type`、`--duty`、`--rate`、`--layers-json`

## 约定

- 输入是平台无关的文件路径和波形配置
- 输出是写入磁盘的 WAV 文件
- 无效配置应明确报错，而不是静默回退；唯一例外是未提供可听层时使用文档约定的默认单个脉冲层

## 层结构

每个层包含：

- `type`：`pulse`、`sine`、`sawtooth`、`triangle` 之一
- `duty`：脉冲宽度，校验范围为 `0.01` 到 `0.99`
- `volume`：线性基础增益，校验为 `>= 0`
- `frequency_curve`：可选的 `{frequency_hz, gain_db}` 点数组

频率曲线会根据每个被渲染音符的基频进行计算。计算出的曲线增益会乘以该层针对该音符的基础 `volume`。

曲线规则：

- 缺失、`null` 或空的 `frequency_curve` 表示全频段 `0 dB`
- 支持的音符频率范围：`8.1757989156 Hz` 到 `12543.8539514 Hz`
- 支持的增益范围：`-36 dB` 到 `+12 dB`
- 每层最多 `8` 个曲线点
- 点会按 `frequency_hz` 升序排序
- 插值方式是在对数频率轴上对 `gain_db` 做线性插值
- 低于第一个点的值会钳制到第一个点的增益
- 高于最后一个点的值会钳制到最后一个点的增益

UI 代码、打包代码和平台专用启动行为应保留在此目录之外。

## 依赖范围

- `requirements.txt` 只包含渲染器/运行时依赖。
- Web 专用包位于 `legacy/web-flask/requirements.txt`。
- 保留的 macOS 辅助构建包位于 `legacy/native/macos/requirements-build.txt`。
