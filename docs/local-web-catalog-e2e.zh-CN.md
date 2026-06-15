# 本地 skills-web + Catalog Playwright 冒烟流程

这份文档只覆盖 `apps/skills-web` 在真实 Catalog API 模式下的本地联调和 Playwright 冒烟。

## 目标

- 用本地 `kind` Catalog 验证 `skills-web` 连接的是实时后端
- 运行浏览器冒烟，覆盖 catalog、import、governance、audit 主链路

注意：

- Playwright 用例默认依赖已有 fixture package，例如 `@platform.skill/pdf-skill`
- 如果本地 Nexus / Catalog 还是空库，先参考 [local-nexus-e2e.zh-CN.md](./local-nexus-e2e.zh-CN.md) 用独立 `skills` CLI 准备测试数据

## 操作步骤

1. 启动本地 kind 栈：

```bash
make dev
```

2. 在另一个终端启动 web 开发服务器，并显式指向本地 Catalog：

```bash
cd apps/skills-web
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 \
VITE_SKILL_REGISTRY_TOKEN=publisher-secret \
npm run dev
```

3. 打开 <http://127.0.0.1:4174>，确认页面显示真实 catalog 数据。

4. 构建前端：

```bash
npm run build
```

5. 运行 Playwright 冒烟：

```bash
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 \
VITE_SKILL_REGISTRY_TOKEN=publisher-secret \
npm run test:e2e
```

如果第 2 步的 dev server 还在运行，Playwright 会复用现有 `127.0.0.1:4174` 服务；否则会自动拉起 `preview:e2e`。

## 浏览器登录流 E2E

如果你要验证 session-cookie 登录流，直接显式传入 session-enabled Catalog：

```bash
cd apps/skills-web
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18081 \
VITE_SKILL_REGISTRY_TOKEN= \
VITE_SKILL_REGISTRY_ROLE= \
VITE_SKILL_REGISTRY_ACTOR= \
PLAYWRIGHT_CATALOG_COMMAND="$(cd ../.. && pwd)/zarf/tools/start-session-e2e-catalog.sh" \
PLAYWRIGHT_CATALOG_PORT=18081 \
npm run build

VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18081 \
VITE_SKILL_REGISTRY_TOKEN= \
VITE_SKILL_REGISTRY_ROLE= \
VITE_SKILL_REGISTRY_ACTOR= \
PLAYWRIGHT_CATALOG_COMMAND="$(cd ../.. && pwd)/zarf/tools/start-session-e2e-catalog.sh" \
PLAYWRIGHT_CATALOG_PORT=18081 \
npm run test:e2e -- tests/e2e/session-login.spec.ts
```

默认登录凭据：

- `reader / reader-secret`
- `operator / op-secret`

## 推荐做法

- 想验证构建产物 + preview 时，先停掉第 2 步的 dev server，再运行 Playwright
- 想快速排 UI 问题时，可以保留 dev server，让 Playwright 直接复用它
- 每次重跑治理写链路前，最好确保 fixture 状态可重复；必要时完整重建本地环境

## 常见问题

`Real mode data missing`

- 检查 `VITE_SKILL_REGISTRY_URL` 是否指向本地 Catalog
- 检查本地 Catalog 是否已有 fixture package

`package_forbidden` / `forbidden`

- 这套冒烟默认使用 `publisher-secret`
- 如果换成 reader token，治理和 packaging 写操作会失败

`Fixture already mutated`

- 当前库里的治理 fixture 可能已经被之前的测试推进到最终状态
- 需要时执行：

```bash
kind delete cluster --name skill-registry
make clean-dev
make dev
```
