# 本地 skills-web + Catalog Playwright 冒烟流程

这份文档只覆盖 `apps/skills-web` 在真实 Catalog API 模式下的本地联调和 Playwright 冒烟，不覆盖单独的发布脚本实现细节。

## 目标

- 用本地 `kind` Catalog 验证 `skills-web` 不是 mock-only 页面
- 再运行浏览器冒烟，覆盖 catalog、import、governance、audit 这些主链路

注意：

- 这些 Playwright 用例默认依赖已经存在的 fixture package，例如 `@platform.skill/pdf-skill`、`@platform.skill/code-review`
- 如果你的本地 Nexus / Catalog 还是空库，先通过 `make cli ARGS="publish ..."`、`make cli ARGS="import-local ..."` 或 `batch-import` 准备一份测试数据

## 操作步骤

1. 启动本地 `kind` 栈：

```bash
make dev-run
```

2. 在另一个终端启动 web 开发服务器，并显式指向本地 Catalog：

```bash
cd apps/skills-web && VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 VITE_SKILL_REGISTRY_TOKEN=publisher-secret npm run dev
```

3. 打开 <http://127.0.0.1:4174>，确认能看到真实 catalog 数据，而不是 mock 页面。

4. 在 `apps/skills-web` 目录执行构建。
说明：如果你要让随后由 Playwright 拉起的 preview 也保持 real mode，请复用同样的 `VITE_SKILL_REGISTRY_URL` / `VITE_SKILL_REGISTRY_TOKEN` 环境变量。

```bash
npm run build
```

5. 运行 Playwright 冒烟。

如果你还保留着第 3 步的 `npm run dev`，Playwright 会复用现有的 `127.0.0.1:4174` 服务。
如果你已经停掉 dev server，先带上相同的环境变量再跑测试，让 Playwright 自动拉起 `preview:e2e`：

```bash
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 VITE_SKILL_REGISTRY_TOKEN=publisher-secret npm run test:e2e
```

## 浏览器登录流 E2E

如果你要验证 session-cookie 登录而不是固定 bearer token，直接跑：

```bash
make test-web-e2e-login
```

这条命令会：

1. 用 `VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18081` 构建 `skills-web`
2. 自动拉起一个本地 session-enabled Catalog
3. 运行 `tests/e2e/session-login.spec.ts`

默认登录凭据：

- `reader / reader-secret`
- `operator / op-secret`

## 推荐做法

- 想验证“构建产物 + preview”时，先停掉第 2 步的 dev server，再执行第 5 步
- 想快速排 UI 问题时，可以保留 dev server，让 Playwright 直接复用它
- 每次重跑 `governance-audit.spec.ts` 前，最好先执行 `make dev-reset`，避免上一次治理动作已经把 fixture 版本推进到不可再次变更的状态

## 常见问题

`Mock mode`

- 说明构建或 dev 启动时没有带 `VITE_SKILL_REGISTRY_URL`
- 回看第 3 步和第 6 步的环境变量是否完整

`package_forbidden` / `forbidden`

- 这套冒烟默认使用 `publisher-secret`
- 如果换成 reader token，治理和 packaging 写操作会失败

`No governable fixture version remained`

- 说明当前库里的治理 fixture 已经被你之前的测试推进到了最终状态
- 先执行 `make dev-reset`
