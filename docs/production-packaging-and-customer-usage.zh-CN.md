# 生产打包与客户使用指南

## 文档定位

本文描述 `skill_registry` 仓库如何：

- 打包成可上线的 Catalog 服务与 CLI 分发物
- 接到内部 npm-compatible 仓形成完整发布链路
- 交付给客户做搜索、安装、更新与发布

它不是新的系统设计；它是一份面向部署、发布和客户交付的运行说明。

## 读者

- 平台运维 / 发布工程师
- Catalog 服务部署方
- CLI 分发方
- 客户侧接入工程师
- 负责发布 skill 的内容提供方

## 先说结论

项目当前提供：

- Catalog API
- 本地 packaging / publish 流程
- registry-first CLI 搜索、安装、更新、移除
- 项目级 `skills-lock.json` 与 `./.agents/skills/` 安装语义

但它还不是“零前提直接公网开放”的最终成品。正式给客户使用前，至少要确认下面两类约束：

### 1. sample 数据与 mock 路由

`cmd/skill-registry/main.go` 在空库启动时会：

- 构造 `sampleData`
- 注册 `/mock-npm` mock 路由
- 在 repository 初始化时把 sample skill seed 到空数据库

这意味着：

- 直接拿默认 build 启动 production，会出现 sample 数据
- 默认 binary 也会暴露 mock 能力

仓库提供了这两个开关：

- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY`

并且 [zarf/docker/Dockerfile](/Users/ultraman/workspace/skill_registry/zarf/docker/Dockerfile) 的默认值是：

- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA=false`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY=false`

### 2. CLI 安装时的 tarball 认证

CLI 安装 skill 时，会直接对 `tarballUrl` 做 HTTP `fetch`，不会额外附加 npm registry token。

这意味着客户安装链路默认假设：

- Catalog 返回的 tarball URL 对客户可直接访问
- 或者该 URL 所在的 npm-compatible 仓已经通过别的方式允许读取

如果你的 production npm 仓要求单独 npm auth，而 tarball 又不能匿名读，那么 CLI 还需要补 tarball 读取认证能力。

## 生产交付物

建议把系统拆成三类交付物：

### 1. Catalog 服务

形态：

- 单个 Go binary
- 或你自己封装后的容器镜像

职责：

- 搜索 skill
- 解析 install target
- 提供 packaging / release-record API
- 记录治理与审计数据

### 2. `skills` CLI

形态：

- npm package
- 或 `npm pack` 后的 `.tgz`

职责：

- 面向客户提供 `find` / `add` / `list` / `update` / `remove`
- 面向发布方提供 `inspect` / `publish`

### 3. skill bundle 制品

形态：

- 发布到内部 npm-compatible 仓的每个 `package_name + version`

职责：

- 作为真正可下载、可安装的 skill bundle
- 由 `release-record` 登记进 Catalog

## 推荐生产拓扑

最小可用拓扑：

1. 客户端安装 `skills` CLI
2. CLI 访问 Catalog HTTPS 域名
3. Catalog 使用 Postgres 持久化 catalog / audit / idempotency 数据
4. Catalog 访问内部 npm-compatible 仓读取 package metadata 与 tarball
5. 客户安装 skill 时，CLI 直接读取 Catalog 返回的 tarball URL

推荐职责边界：

- Catalog：做解析、治理、审计、packaging
- npm-compatible 仓：存 tarball 与 registry metadata
- CLI：只做消费端入口与发布端工具

## 生产打包步骤

### 1. 先跑验证

在 release 前至少执行：

```bash
make verify
```

这会覆盖：

- `go test ./...`
- CLI type-check
- CLI tests

如果这里只过本地 mock 流程、不做真实 npm 仓联调，不建议直接发给客户。

### 2. 打包 Catalog 服务

仓库同时提供两种产物路径：

#### 方式 A：Go binary

```bash
mkdir -p dist
go build -o dist/skill-registry ./cmd/skill-registry
```

#### 方式 B：容器镜像

```bash
docker build -f zarf/docker/Dockerfile -t skill-registry:latest .
```

或直接：

```bash
make docker-build
```

### 3. 生产 compose 示例

如果你要一个最小的 `Catalog + Postgres` 部署示例，可直接使用：

```bash
docker compose -f zarf/compose/production.yml up --build
```

或：

```bash
make compose-production-up
```

这个 compose 文件包含：

- `catalog`
- `postgres`

它不包含 Verdaccio，因为正式生产应优先接外部真实 npm-compatible 仓，而不是把测试仓混进生产拓扑。

### 建议的运行时环境变量

必需项：

- `SKILL_REGISTRY_ADDR`
- `SKILL_REGISTRY_DATABASE_URL`
- `SKILL_REGISTRY_READER_TOKEN`
- `SKILL_REGISTRY_AUTHORING_TOKEN`
- `SKILL_REGISTRY_PUBLISHER_TOKEN`

生产常见附加项：

- `SKILL_REGISTRY_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_AUTH_TOKENS_JSON`

生产安全相关项：

- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA=false`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY=false`

不要把这些本地/样例变量直接带进生产：

- `SKILL_REGISTRY_SAMPLE_NPM_BASE_URL`
- `SKILL_REGISTRY_ARTIFACTS_PATH`（除非你明确要走静态 JSON artifact source）

### 数据库建议

服务同时支持：

- SQLite
- Postgres

生产推荐：

- 用 Postgres
- 不要用临时 SQLite 文件做正式 catalog 存储

原因：

- 审计、治理、幂等等数据都已经进入 SQL 路径
- Postgres 更符合多实例与正式备份场景

### 本地 compose 联调

如果你要一个本地真实联调栈，而不是单独手工起服务，可以直接用：

```bash
docker compose -f zarf/compose/local.yml up --build
```

或：

```bash
make compose-local-up
```

这个 compose 文件包含：

- `catalog`
- `postgres`
- `verdaccio`

本地 compose 默认也关闭 sample seed 和 `/mock-npm`，因此更接近“真实发布后再搜索/安装”的链路，而不是继续依赖内置样例数据。

### 本地联调时切到真实 local npm registry

如果你不是想继续使用内置 `/mock-npm`，而是想把本地 Catalog 接到一个真实的 local
npm-compatible registry，例如：

- Nexus
- Verdaccio
- Artifactory
- GitLab Package Registry
- 其他兼容 npm metadata / tarball 读取语义的内部仓

关键不是改 CLI，而是改 Catalog 启动时的两个环境变量：

- `SKILL_REGISTRY_SAMPLE_NPM_BASE_URL`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`

它们的职责分别是：

- `SKILL_REGISTRY_SAMPLE_NPM_BASE_URL`
  - 决定空库 seed 进去的 sample skill 写入什么 `tarballUrl`
  - 影响 `skills add` 最终去哪里下载 tarball
- `SKILL_REGISTRY_NPM_REGISTRY_URL`
  - 决定 Catalog 在 `release-record` / artifact source 中从哪个 npm registry 读取 package metadata
  - 影响 `skills publish` 后 Catalog 如何回查已发布版本

通用启动示例：

```bash
make registry-reset
env \
  GOCACHE=/private/tmp/go-cache \
  SKILL_REGISTRY_ADDR=127.0.0.1:18080 \
  SKILL_REGISTRY_DATABASE_URL=file:/private/tmp/skill-registry.db \
  SKILL_REGISTRY_SAMPLE_NPM_BASE_URL=http://127.0.0.1:8081/repository/skills-npm \
  SKILL_REGISTRY_NPM_REGISTRY_URL=http://127.0.0.1:8081/repository/skills-npm \
  SKILL_REGISTRY_READER_TOKEN=reader-secret \
  SKILL_REGISTRY_AUTHORING_TOKEN=authoring-secret \
  SKILL_REGISTRY_PUBLISHER_TOKEN=publisher-secret \
  go run ./cmd/skill-registry
```

注意：

- 这里的 registry URL 要填 npm repository 根地址，而不是产品 UI 首页
- 切换前一定要先清掉本地 sqlite；当前 seed 只在空库时写入
- CLI 仍然只连 Catalog，例如 `SKILLS_REGISTRY_URL=http://127.0.0.1:18080`
- 不需要把 CLI 直接改成指向 Nexus / Verdaccio / Artifactory

### 关于 tarball 读取认证

即使你已经把 Catalog 接到了真实 local npm registry，CLI 安装 bundle 时仍然是直接 `fetch(tarballUrl)`。

所以还要额外确认：

- tarball URL 对客户机器是否可直接访问
- 该 local npm registry 是否允许匿名读 tarball
- 如果不允许匿名读，是否已经通过网关、反向代理、预签名 URL 或后续 CLI 改造补上认证

否则会出现：

- `find` 正常
- `add` 在 tarball 下载阶段失败

### 本地真实 E2E 时优先 Verdaccio

如果你只是想在开发机上验证：

- `npm publish`
- metadata 解析
- `release-record`
- tarball 下载
- `skills add`

那么 Verdaccio 通常比 Nexus 更合适，因为它更轻、更快，也更容易脚本化。

本仓库已经提供：

- `make verdaccio-run`
- `make registry-run-verdaccio`
- `make cli-publish-verdaccio`
- `make batch-import-verdaccio`

详细步骤见 [docs/local-verdaccio-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-e2e.zh-CN.md)。

如果你还要验证“发布需要登录”的本地路径，本仓库还提供了认证版 Verdaccio。它推荐采用：

- `publish` 需要认证
- tarball 读取允许匿名

这是因为 CLI `add` 下载 tarball 时不会附带 npm registry 认证。详细边界见 [docs/local-verdaccio-auth-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-auth-e2e.zh-CN.md)。

### 3. 打包 `skills` CLI

先构建：

```bash
npm --prefix apps/skills-cli ci
npm --prefix apps/skills-cli run build
```

如果你要生成离线可分发包：

```bash
cd apps/skills-cli
npm pack
```

输出物一般会是：

- `skills-<version>.tgz`

如果你要发到内部 npm 仓：

```bash
cd apps/skills-cli
npm publish --registry https://npm.example.com
```

适合客户的两种分发方式：

### 方式 A：内部 npm 分发

优点：

- 客户安装简单
- 便于版本升级

安装示例：

```bash
npm install -g skills --registry https://npm.example.com
```

### 方式 B：离线 tgz 分发

优点：

- 更适合封闭环境
- 不要求客户先接 npm registry

安装示例：

```bash
npm install -g ./skills-1.5.9.tgz
```

### 4. 发布 skill 到生产链路

当前推荐流程：

1. skill 作者准备本地目录
2. 用 `skills inspect` 校验
3. 用 `skills publish` 走 packaging + `npm publish` + `release-record`

示例：

```bash
npx skills inspect ./my-skill --registry https://catalog.example.com
npx skills publish ./my-skill --registry https://catalog.example.com
```

当前 `skills publish` 内部顺序是：

1. 运行 packaging session
2. 调用 `POST /api/v1/skill-packaging/pack`
3. 本地 `npm publish <bundle>`
4. 调用 `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

这意味着发布方还要自己准备 npm publish 凭据，例如：

- `.npmrc`
- npm token
- 企业内部 npm 登录态

### 5. 批量初始化已有 skill

如果不是单个作者手工执行 `skills publish`，而是要把一批现成 skill 批量灌入生产 Catalog，当做初始数据，可以直接使用仓库内的 Python 脚本：

```bash
uv run python tools/batch-import/batch_import_skills.py --manifest /abs/path/to/import.json
```

它仍然走同一条生产链路：

1. prepare
2. validate
3. pack
4. `npm publish`
5. `release-record`

注意：

- `path` 模式使用的是 `source.type=local`，要求 Catalog 服务端能访问相同文件路径
- 如果 Catalog 和导入机不在同一台机器，改用 manifest 里的 `source.type=git`
- `--npm-registry` 要和 Catalog 的 `SKILL_REGISTRY_NPM_REGISTRY_URL` 保持一致

详细字段与示例见 [docs/batch-import-skills.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/batch-import-skills.zh-CN.md)。

## 给客户的使用方式

### 1. 客户先安装 CLI

如果你是通过内部 npm 分发：

```bash
npm install -g skills --registry https://npm.example.com
```

如果你是通过 tgz 分发：

```bash
npm install -g ./skills-1.5.9.tgz
```

### 2. 客户配置 Catalog 访问参数

最少需要：

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>
```

`SKILLS_REGISTRY_ROLE` 更适合本地开发或带角色头的内网环境；正式生产一般优先使用 bearer token。

### 3. 客户搜索 skill

```bash
skills find pdf
skills find pdf --json
```

用途：

- 人工搜索
- 平台侧脚本自动化查询

### 4. 客户安装 skill

### 项目级安装

在项目根目录执行：

```bash
skills add @platform.skill/pdf-skill -a codex -y
```

结果：

- skill 安装到 `./.agents/skills/`
- 项目状态写到 `./skills-lock.json`

适合：

- 团队项目
- 需要把 skill 跟 repo 一起管理的场景

### 全局安装

```bash
skills add @platform.skill/pdf-skill -g -a codex -y
```

适合：

- 个人环境
- 跨项目复用

### 关于 `-y`

当前 CLI 契约下，`-y` 的含义是：

- 跳过确认 prompt
- 不再隐式广播到所有已知 adapter
- 未显式指定 `--agent` 时，会优先使用已检测到的 agent；若一个也没检测到，则回落到 universal `.agents/skills` 那组 agent

因此，面向客户的推荐命令仍然是：

- 显式给 `--agent`
- 是否全局由 `-g` 决定

不要把“裸 `skills add <package> -y`”当成长期稳定契约来写进客户手册。

### 5. 客户查看、更新、移除

查看：

```bash
skills list
skills list -g
```

更新：

```bash
skills update
skills update @platform.skill/pdf-skill
skills update -g
```

移除：

```bash
skills remove @platform.skill/pdf-skill -y
skills remove --global @platform.skill/pdf-skill -y
```

## 面向客户的最小交付包

如果你要把这套系统给一个内部客户团队，至少应一起交付：

1. `skills` CLI 安装方式
2. Catalog 域名
3. 读 token 申请方式
4. 推荐的项目级安装命令
5. 推荐的全局安装命令
6. `skills-lock.json` 与 `./.agents/skills/` 的语义说明
7. skill 发布方的单独权限与流程说明

## 推荐写进客户文档的命令

```bash
# 1. 配置 Catalog
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

# 2. 搜索
skills find pdf

# 3. 项目级安装到指定 agent
skills add @platform.skill/pdf-skill -a codex -y

# 4. 查看项目内已安装 skill
skills list

# 5. 更新
skills update @platform.skill/pdf-skill
```

## 上生产前的检查单

正式给客户前，建议逐项确认：

- 已移除或禁用 sample seed
- 已移除或禁用 `/mock-npm` 面向客户暴露
- Catalog 已使用正式 Postgres
- `SKILL_REGISTRY_NPM_REGISTRY_URL` 指向正式 npm-compatible 仓
- `skills publish` 使用的 npm 凭据已经走正式账号体系
- CLI 安装链路所需的 tarball URL 对客户真实可访问
- 已验证客户侧 `skills add` 在真实网络与真实认证下能成功下载 bundle
- 已明确项目级安装与全局安装的推荐默认值
- 已准备 token 轮换、审计、回滚方案

## 交付前不应省略的风险提示

如果你现在就对外给客户使用，至少要提前讲清楚：

- repo 默认代码含 sample 行为，不应直接拿开发默认配置上生产
- CLI 安装 tarball 的认证模型仍然偏简化
- 仓库虽已提供容器与 compose 示例，但仍没有 Helm、secret 管理和正式 CI/CD release pipeline

这三点不说明清楚，客户很容易把“可以跑通”误解成“已经是完整生产产品”。
