# Web UI i18n

Language/语言: [English](./README.md) | 简体中文

旧 Flask 渲染前端使用此目录中的 JSON 目录，同时服务于 Flask 渲染的 HTML 和
`templates/index.html` 中的浏览器 UI。生产 Vue 前端的 catalog 副本位于
`frontend/src/i18n/`。修改任一 catalog 集合时，请保持 `en.json`、`fr.json`
和 `zh-CN.json` 的键集合对齐；英文仍是回退语言。

## 法语翻译覆盖范围

第一批法语翻译已覆盖共享目录中的可见浏览器 UI：语言选择器、设置对话框、主题控制、MIDI 队列控制、层和曲线控制、波形标签、处理状态和警报，以及 `/synthesise` 使用的 Flask 缺失文件校验错误。

## 暂缓处理的字符串

- 本批次中，Web Flask 文档和启动器文本仍保留在现有英文和简体中文文档中。
- 浏览器 console 警告、IndexedDB/localStorage 键和 JavaScript 函数名仍保留为内部开发者字符串。
- `Hz`、`dB`、波形载荷值、生成文件名和预览资源名称等技术值保持语言中立。
- 原生 macOS 和 Windows 应用字符串有意不纳入这个 Web 专用翻译范围。
