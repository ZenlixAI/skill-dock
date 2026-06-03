# skill_registry

一个面向 agent skill 的 Catalog 服务与配套 CLI。

CLI 细节看 [apps/skills-cli/README.md](/Users/ultraman/workspace/skill_registry/apps/skills-cli/README.md)。

## 环境

- Go `1.26+`
- Node.js `18+`
- `npm`
- `uv`
- `make`

## 项目结构

```text
cmd/
  skill-registry/     Go 服务入口

app/
  auth/               认证与授权
  catalogapi/         HTTP API 与请求编排
  observability/      请求指标与日志辅助

business/
  catalog/            Catalog 业务模型与仓储实现
  packaging/          skill 打包规则与服务

foundation/
  artifact/           npm / mock registry artifact 适配器
  persist/            SQL audit / idempotency 存储适配器

apps/
  skills-cli/         独立 CLI
  skills-web/         首版 Web UI scaffold（Catalog / Import / Governance）

zarf/
  docker/             Dockerfile 与 Verdaccio 配置
  compose/            本地 / 生产 compose
  scripts/            运行 Verdaccio / Nexus 的辅助脚本

tools/
  batch-import/       批量导入工具
```

## 常用命令

```bash
make help
make verify
make verify-full
```

Web UI 骨架：

```bash
make web-install
make web-dev
```

## 快速开始

### 1. 验证

```bash
make verify
```

会执行：

- `go test ./...`
- CLI type-check
- CLI fast tests（默认排除 `apps/skills-cli/tests/dist.test.ts` 这类构建烟测）

如果你要连 CLI 的 dist build smoke test 一起跑：

```bash
make verify-full
```

如果你只在改 packaging / publish 链路，优先用更窄的目标：

```bash
make test-go-packaging
make test-cli-packaging
```

### 2. 本地联调

最短路径是先跑 Catalog，再用 CLI 对拍：

```bash
make registry-run-fresh
```

另一个终端：

```bash
make cli-find QUERY=pdf
make cli-add PACKAGE=@platform.skill/pdf-skill
```

这条路径默认使用 sample data，并让 sample skill 的 tarball 指向本地 `/mock-npm`。

如果你要连 package metadata 也一起切到内置 mock registry：

```bash
make registry-run-mock
```

如果你只是在现有 sqlite 上重启 Catalog：

```bash
make registry-run
```

### 3. 真实 npm-compatible registry 联调

本地真实 E2E 优先用 Verdaccio：

```bash
make verdaccio-run
make registry-run-verdaccio
```

然后：

```bash
make cli-publish-verdaccio SKILL_PATH=/abs/path/to/skill
make cli-find QUERY=pdf
make cli-add PACKAGE=@platform.skill/pdf-skill
```

详细步骤看 [docs/local-verdaccio-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-e2e.zh-CN.md)。

如果要验证“发布需要登录”的路径：

```bash
make verdaccio-run-auth
make verdaccio-login
make registry-run-verdaccio-auth
make cli-publish-verdaccio-auth SKILL_PATH=/abs/path/to/skill
```

完整实操见 [docs/local-verdaccio-auth-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-auth-e2e.zh-CN.md)。

如果你要补一条更接近企业仓行为的本地 Nexus 路径：

```bash
make nexus-run
make nexus-bootstrap
make registry-run-nexus
make cli-publish-nexus SKILL_PATH=/abs/path/to/skill
```

如果你要验证“tarball / metadata 读取也要求鉴权”的严格模式：

```bash
make nexus-run
make nexus-bootstrap-strict
make registry-run-nexus
make cli-publish-nexus SKILL_PATH=/abs/path/to/skill
make cli-add-nexus PACKAGE=@platform.skill/pdf-skill
```

详细步骤看 [docs/local-nexus-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-nexus-e2e.zh-CN.md)。

如果你要接别的本地或企业 npm-compatible registry
（例如 Nexus、Artifactory、GitLab Package Registry），启动 Catalog 时改这两个值即可：

```bash
make registry-reset
env \
  GOCACHE=/private/tmp/go-cache \
  SKILL_REGISTRY_ADDR=127.0.0.1:18080 \
  SKILL_REGISTRY_DATABASE_URL=file:/private/tmp/skill-registry.db \
  SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=https://npm.example.internal/repository/skills-npm \
  SKILL_REGISTRY_NPM_REGISTRY_URL=https://npm.example.internal/repository/skills-npm \
  SKILL_REGISTRY_READER_TOKEN=reader-secret \
  SKILL_REGISTRY_AUTHORING_TOKEN=authoring-secret \
  SKILL_REGISTRY_PUBLISHER_TOKEN=publisher-secret \
  go run ./cmd/skill-registry
```

说明：

- `SKILL_REGISTRY_SAMPLE_NPM_BASE_URL` 决定 sample skill 写入什么 `tarballUrl`
- `SKILL_REGISTRY_NPM_REGISTRY_URL` 决定 Catalog 从哪个 npm registry 读取 package metadata
- 上面示例是 Nexus 风格 URL；换成别的 npm registry 时，只需要替换成对应地址
- 切换这两个值前应先 `make registry-reset`，避免旧 seed 数据残留

### 4. CLI 使用方式

```bash
make cli-find QUERY=pdf
make cli-find-json QUERY=pdf
make cli-add PACKAGE=@platform.skill/pdf-skill
make cli-add-nexus PACKAGE=@platform.skill/pdf-skill
make cli-publish SKILL_PATH=/abs/path/to/skill
```

说明：

- `make cli-add` 默认做项目级安装
- 对严格鉴权的 Nexus tarball 读取，使用 `make cli-add-nexus` 让 install-side 复用 `/private/tmp/skill-registry-nexus/.npmrc`
- skill 会安装到当前工作目录下的 `./.agents/skills/`
- 项目级安装状态会写到当前工作目录下的 `skills-lock.json`
- 用户级安装请直接使用 CLI 并传 `-g`

### 4.5 Web UI Skeleton

当前仓库已经补了一个独立的首版 Web UI 骨架：

- 目录：`apps/skills-web/`
- 技术栈：`Vite + React + TypeScript`
- 当前页面：
  - `/`
  - `/authoring`
  - `/catalog`
  - `/skills/:packageName`
  - `/skills/:packageName/versions/:version`
  - `/import`
  - `/governance`
  - `/audit`

运行：

```bash
make web-install
make web-dev
```

如果要让 Web UI 直接命中本地 Catalog，而不是默认 mock client：

```bash
make registry-run-fresh
```

另一个终端：

```bash
cd apps/skills-web
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 \
VITE_SKILL_REGISTRY_TOKEN=publisher-secret \
npm run dev
```

可选环境变量：

- `VITE_SKILL_REGISTRY_URL`
  - Catalog 基地址；为空时前端自动退回 mock mode
- `VITE_SKILL_REGISTRY_TOKEN`
  - 真实 bearer token，优先级高于 `VITE_SKILL_REGISTRY_ROLE`
- `VITE_SKILL_REGISTRY_ROLE`
  - 仅本地联调用；未提供 token 时走 `X-Role`
- `VITE_SKILL_REGISTRY_ACTOR`
  - 可选，透传到 `X-Actor`
- `VITE_SKILL_REGISTRY_USE_MOCK=true`
  - 即使提供了 URL，也强制前端走 mock mode

构建验证：

```bash
make typecheck-web
make build-web
```

说明：

- 当前 UI 已支持 mock / real API 双模式
- 读接口已接 Catalog API
- 写接口已接 packaging prepare/validate/pack、release-record、skill/version governance
- `Import & Package` 的 `pack` 结果现在可以直接 handoff 到 `/skills/:packageName/versions/:version` 的 release review 表单
- 新增 `/authoring` 作为更上层 platform / authoring 入口页，用来承接 import -> release review -> governance / audit 的作者流
- UI 设计规格见 [docs/superpowers/specs/2026-06-02-registry-web-ui-design.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/superpowers/specs/2026-06-02-registry-web-ui-design.zh-CN.md)
- 前端实施计划见 [docs/superpowers/plans/2026-06-02-registry-web-ui-implementation-plan.md](/Users/ultraman/workspace/skill_registry/docs/superpowers/plans/2026-06-02-registry-web-ui-implementation-plan.md)

### 5. Docker 部署

生产镜像：

```bash
make docker-build
```

如果你要直接起一个 `Catalog + Postgres` 的最小生产部署示例：

```bash
make compose-production-up
```

关键点：

- `zarf/docker/Dockerfile` 默认关闭 sample seed 和 `/mock-npm`
- `zarf/compose/production.yml` 会同时起 `catalog` 和 `postgres`
- 生产前应显式设置 `SKILL_REGISTRY_NPM_REGISTRY_URL`

如果你要起一个本地真实联调栈 `Catalog + Postgres + Verdaccio`：

```bash
make compose-local-up
```

详细说明看 [docs/production-packaging-and-customer-usage.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/production-packaging-and-customer-usage.zh-CN.md)。

### 6. 批量导入

如果你已经有一批本地 skill 目录，或者要从 git 源批量导入，可以直接用：

```bash
uv run python tools/batch-import/batch_import_skills.py --manifest tools/batch-import/example.json --dry-run
```

正式执行：

```bash
uv run python tools/batch-import/batch_import_skills.py \
  --manifest tools/batch-import/example.json \
  --npm-registry http://127.0.0.1:4873
```

详细说明看 [docs/batch-import-skills.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/batch-import-skills.zh-CN.md)。

也可以直接走 Make 入口：

```bash
make batch-import MANIFEST=/abs/path/to/import.json
make batch-import-verdaccio MANIFEST=/abs/path/to/import.json
make batch-import-verdaccio-auth MANIFEST=/abs/path/to/import.json
```

当前导入行为：

- 对于已经满足导入请求所需 target / metadata 条件的外部 `local` / `git` skill，默认路径不再要求源目录预先手写 bundle 用的 `package.json` 或 capability 占位内容
- packaging `prepare` 默认会只在临时 workspace 内重写面向 bundle 的 `package.json` 以匹配内部 target / metadata，并在需要时合成缺失的 capability 占位内容，不会回写源目录
- 如果你想拒绝这类还不满足 publishable bundle 契约的源目录，使用批量导入脚本的 `--strict`，或在 Make 入口传 `STRICT=1`

如果想先把 Python 运行环境准备好：

```bash
make python-sync
```

### 7. 清理本地状态

```bash
make registry-reset
make clean-dev
```

## 关键配置

CLI：

- `SKILLS_REGISTRY_URL`
- 可选：`SKILLS_REGISTRY_TOKEN`
- 可选：`SKILLS_REGISTRY_ROLE`

Registry：

- `SKILL_REGISTRY_ADDR`
- `SKILL_REGISTRY_DATABASE_URL`
- `SKILL_REGISTRY_READER_TOKEN`
- `SKILL_REGISTRY_AUTHORING_TOKEN`
- `SKILL_REGISTRY_PUBLISHER_TOKEN`
- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY`

`--registry <url>` 只建议作为临时调试覆盖，不应作为常态配置方式。

补充约束：

- `make registry-run` 默认让 sample skill 的 tarball 指向本地 `/mock-npm`
- seed 只在空库时写入；如果你改过 registry 地址，先 `make registry-reset`
- `make registry-run-mock` 会把 package metadata 读取也切到本地 `/mock-npm`
- CLI 永远连 Catalog；切换后端 npm registry 时，只需要改 Catalog 的环境变量
- 批量导入脚本不依赖具体 npm 实现；只要 `npm publish` 与 Catalog `release-record` 指向同一个 registry 即可
- 本地真实 E2E 优先 Verdaccio；要模拟企业权限和仓库治理再接 Nexus 一类系统
- 当前认证版 Verdaccio 推荐“发布鉴权、读取匿名”；如果 tarball 读取也强制鉴权，`skills add` 不是当前目标
- Docker 镜像默认关闭 sample data 与 `/mock-npm`
