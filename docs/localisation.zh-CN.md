# 本地化流程

Language/语言: [English](./localisation.md) | 简体中文

本文档是 OctaBit 添加或修改面向用户本地化内容的标准流程。

## 范围

- 生产 UI 本地化位于 `frontend/src/i18n/`。
- 仓库文档维护英文和简体中文两个版本。
- 旧 Flask UI catalog 位于 `legacy/web-flask/i18n/`，只有任务明确面向旧回退路径时才应修改。
- 已暂停的原生 macOS 和 Windows 应用本地化不在范围内，除非任务明确恢复或指定这些应用。

## 添加生产前端 locale

1. 选择稳定的 locale code。除非请求的语言需要区域区分（例如 `zh-CN`），否则优先使用 `es`、`fr` 这类通用语言代码。
2. 添加 `frontend/src/i18n/<locale>.json`，并保持与 `en.json` 相同的键集合。
3. 在所有生产前端 catalog 中加入 `toolbar.language_option.<locale>`。显示值使用该语言的本地名称，例如 `Español`。
4. 更新 `frontend/src/composables/useLocale.ts`，导入新 locale，将它加入 `Locale` 类型、`translationsByLocale` 和 `supportedLocales`。
5. 如果选择逻辑发生变化，扩展 `frontend/src/composables/__tests__/useLocale.test.ts` 中的 URL/cookie 选择测试，并保持 catalog parity 测试覆盖所有生产前端 catalog。

不要在 Vue components、composables 或 templates 中硬编码新的面向用户 Web 文案。应添加 catalog key 并调用 `t(...)`。

## 文档更新

当生产 UI 语言集合变化时，更新已有英文和简体中文文档中列出前端 catalog 覆盖范围的位置。至少检查：

- `README.md`
- `README.zh-CN.md`
- `docs/repository-layout.md`
- `docs/repository-layout.zh-CN.md`
- `AGENTS.md`
- `CLAUDE.md`
- `legacy/web-flask/i18n/README.md`
- `legacy/web-flask/i18n/README.zh-CN.md`

请区分文档语言覆盖范围和 UI 语言覆盖范围。即使生产 UI 支持更多语言，仓库文档仍保持英文和简体中文。

## 验证

修改生产 UI 本地化后运行前端检查：

```bash
cd frontend
npm run test
npm run build
```

部署前，确认构建产物包含所有 locale option 标记：

```bash
for catalog in frontend/src/i18n/*.json; do
  locale="$(basename "$catalog" .json)"
  grep -R -q "\"toolbar.language_option.$locale\"" frontend/dist/assets
done
```

生产辅助脚本 `deploy/production/deploy-vue-production.sh` 会执行同样的标记检查：先检查本地构建产物；当设置了 `PUBLIC_URL` 时，还会在 Caddy reload 后检查公开 JavaScript bundle。

