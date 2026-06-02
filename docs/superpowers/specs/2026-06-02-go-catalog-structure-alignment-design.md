# Go Catalog Structure Alignment Design

## 文档定位

本文定义 `skill_registry` 仓库中 Go Catalog 服务的目录重组方案，目标是吸收
`ardanlabs/service` 的分层思想，但不机械复制其目录名字，也不在本次重组中调整
TypeScript CLI 的内部结构。

本文只覆盖：

- Go Catalog 服务代码结构
- 部署资产与运维脚本归位
- 业务辅助工具归位
- 分阶段迁移顺序

本文不覆盖：

- CLI 内部重组
- 领域模型重写
- 新功能开发

## 设计目标

- 让 Go Catalog 服务形成更清晰的 seam：入口、应用编排、业务、基础设施、部署资产
- 降低当前 `internal/*` 目录里业务逻辑、HTTP 适配器、SQL 适配器、部署脚本混放带来的理解成本
- 保持现有 ADR 不变：
  - **Package Name** 仍是唯一公开身份
  - Catalog 仍治理 discovery / resolution / governance
  - npm-compatible 仓仍只负责 artifact storage
- 让部署资产与操作工具的职责边界清楚：
  - `zarf/` 放“怎么运行系统”
  - `tools/` 放“怎么操作系统中的内容和数据”

## 非目标

- 不追求一次性“大爆炸”重命名
- 不把 TypeScript CLI 迁入 Go 服务分层
- 不在本轮拆出过多细粒度业务模块，例如单独的 `release` / `governance` / `install`
- 不为了模仿 `ardanlabs/service` 而引入当前项目不需要的抽象

## 当前问题

当前仓库顶层有这些主要区域：

```text
cmd/
internal/
apps/skills-cli/
docs/
scripts/
docker/
```

其中主要 friction 在 Go Catalog 服务部分：

1. `internal/catalog` 同时承载领域类型、聚合持久化、artifact source、mock registry、SQL state store。
2. `internal/httpapi` 同时承载 HTTP handler、认证、审计、SQL idempotency store、运行时观测。
3. `internal/packaging` 是单独模块，但与 Catalog 的业务关系已经足够紧密，仍放在纯技术分组下。
4. 根 `scripts/` 同时包含：
   - 部署/联调脚本
   - Verdaccio 辅助脚本
   - 批量导入工具
5. Docker / compose / Verdaccio 配置已经出现，但仍未收束到一个稳定的“部署资产”位置。

这些问题的结果是：

- 读者必须在多个技术包之间来回跳转才能理解一个完整用例
- 模块 interface 接近实现细节，深度不够
- 部署资产与业务工具混杂，不利于长期维护

## 参考原则

参考 `ardanlabs/service`，本次只吸收它的核心原则，不直接复制其全部目录：

- `cmd/` 是程序入口
- `app/` 放应用层编排和外部接口适配
- `business/` 放领域与核心用例
- `foundation/` 放数据库、外部系统、基础设施适配器
- `zarf/` 放部署与运行资产

对当前项目，这套原则比单纯保留 `internal/catalog` / `internal/httpapi` 更适合，因为：

- Catalog 业务已经不只是简单 CRUD
- HTTP、SQL、artifact source、packaging 都是明显不同的 seam
- 还有 Docker、Compose、Verdaccio、本地导入工具等 repo-level 运行资产需要落位

## 方案比较

### 方案 A：轻量整理

- 保留 `internal/catalog`、`internal/httpapi`、`internal/packaging`
- 只移动根脚本和 Docker 资产

优点：

- 风险最低
- import path 改动最少

缺点：

- Go Catalog 服务仍然主要按技术包划分
- 不能明显改善“跨目录理解一个用例”的问题

### 方案 B：重组 Go Catalog 服务，CLI 保持独立

- Go 服务按 `app / business / foundation / zarf` 分层
- CLI 保留在 `apps/skills-cli/`

优点：

- 最贴近当前问题
- 不触碰 CLI，范围可控
- 适合分阶段迁移

缺点：

- 会有一轮 import path 调整
- 需要短期适应新的包位置

### 方案 C：Go 服务和 CLI 一起统一重排

优点：

- 顶层结构最统一

缺点：

- 迁移面过大
- 风险高
- 当前收益不成比例

## 选定方案

选择 **方案 B：只重组 Go Catalog 服务，CLI 保持独立**。

原因：

- 这是当前仓库最需要整理的部分
- 能最大程度吸收 `ardanlabs/service` 的结构优势
- 避免把一次代码结构整理扩大成多语言、多运行时的大迁移

## 目标结构

目标顶层结构：

```text
cmd/
  skill-registry/

app/
  catalogapi/
  auth/
  observability/

business/
  catalog/
  packaging/

foundation/
  artifact/
    mock/
    npm/
  persist/
    postgres/
    sqlite/

apps/
  skills-cli/

docs/

zarf/
  docker/
  compose/
  scripts/

tools/
  batch-import/
```

## 模块职责

### `cmd/skill-registry`

职责：

- 进程启动入口
- 读取环境变量
- 组装应用模块
- 启动 HTTP server

不负责：

- 业务规则
- SQL 细节
- artifact 解析细节

### `app/catalogapi`

来源：

- 当前 `internal/httpapi/server.go`

职责：

- HTTP 路由
- 请求解码
- 响应编码
- 调用业务模块

它是外部接口 seam，不应继续承载过多 SQL 细节实现。

### `app/auth`

来源：

- 当前 `internal/httpapi/authorizer.go`

职责：

- 认证与角色判定
- request principal 相关逻辑

### `app/observability`

来源：

- 当前 `internal/httpapi/runtime_observability.go`
- 与 request metrics、request logging 相关的辅助逻辑

职责：

- 运行时指标
- 请求日志
- 观测辅助

### `business/catalog`

来源：

- 当前 `internal/catalog` 中与 Agent Skill Aggregate、governance、install resolution、release record 直接相关部分

职责：

- Catalog 核心领域模型
- discovery / resolution / governance 规则
- Release Record 语义
- Agent Skill Aggregate 持久化接口

它是当前系统的核心业务模块。

### `business/packaging`

来源：

- 当前 `internal/packaging`

职责：

- prepare / validate / pack
- single-skill bundle 规则
- package.json 和 SKILL.md 规范化相关逻辑

### `foundation/persist`

来源：

- 当前 SQL state store
- SQL audit store
- SQL idempotency store

职责：

- 数据库存储适配器
- Postgres / SQLite 的具体实现

这里是典型的 adapter，而不是业务模块。

### `foundation/artifact`

来源：

- 当前 `artifact_source.go`
- 当前 `mock_registry.go`

职责：

- npm registry metadata / tarball 读取
- mock registry 适配
- bundle scan 与 artifact-level 校验辅助

### `zarf`

职责：

- 部署与运行资产
- 容器相关配置
- compose 文件
- 本地联调壳脚本

### `tools`

职责：

- 业务辅助工具
- 初始化、导入、批量操作类脚本

## 脚本归位规则

### `zarf/`

放“怎么运行系统”的资产：

- `Dockerfile`
- compose 文件
- Verdaccio 运行脚本
- 本地登录脚本
- 运行时配置模板

建议结构：

```text
zarf/
  docker/
    Dockerfile
    verdaccio/
      config.yaml
  compose/
    production.yml
    local.yml
  scripts/
    run-verdaccio.sh
    verdaccio-login.sh
```

### `tools/`

放“怎么操作系统数据或内容”的工具：

- `batch_import_skills.py`
- 示例 manifest

建议结构：

```text
tools/
  batch-import/
    batch_import_skills.py
    example.json
```

### 禁止继续保留杂糅根 `scripts/`

本轮迁移完成后，不再保留一个同时混放：

- 部署脚本
- registry 脚本
- 数据导入工具

的根 `scripts/` 目录。

## 分阶段迁移

### Phase 1：部署与工具资产归位

先移动这些不影响 Go import path 的内容：

- `Dockerfile` -> `zarf/docker/Dockerfile`
- `docker-compose.production.yml` -> `zarf/compose/production.yml`
- `docker-compose.local.yml` -> `zarf/compose/local.yml`
- `docker/verdaccio/config.yaml` -> `zarf/docker/verdaccio/config.yaml`
- `scripts/run_verdaccio.sh` -> `zarf/scripts/run-verdaccio.sh`
- `scripts/verdaccio_login.sh` -> `zarf/scripts/verdaccio-login.sh`
- `scripts/batch_import_skills.py` -> `tools/batch-import/batch_import_skills.py`
- `scripts/batch-import.example.json` -> `tools/batch-import/example.json`

这是最低风险的第一步。

### Phase 2：Go 包按职责切开

先做包移动，但不改变业务行为：

- `internal/httpapi` 拆到 `app/catalogapi`、`app/auth`、`app/observability`
- `internal/packaging` -> `business/packaging`
- `internal/catalog` 中业务逻辑 -> `business/catalog`
- SQL / artifact 适配器下沉到 `foundation/`

原则：

- 每次移动保持测试通过
- 避免同时重写逻辑和移动位置

### Phase 3：入口重新 wiring

最后调整：

- `cmd/skill-registry/main.go`
- import path
- Docker / compose / Makefile / 文档

这样可以把运行时 wiring 放在最后统一对齐。

## 命名约束

- 业务概念继续使用 `CONTEXT.md` 里的语言：
  - Agent Skill
  - Package Name
  - Skill Version
  - Release Record
  - Install Target
  - Governance Change
- 架构讨论使用：
  - module
  - seam
  - adapter
  - leverage
  - locality

避免重新引入模糊命名，例如：

- `service`
- `manager`
- `helper`
- `util`

除非其职责真的不可再具体化。

## 测试策略

目录重组本身不应改变行为，因此测试目标是“行为对拍”：

- `go test ./...` 仍应通过
- Docker build 仍应成功
- compose 配置仍应可解析
- 本地 Verdaccio E2E 文档命令仍应成立

迁移时优先保留现有测试，再按新包位置重定向。

## 风险与控制

### 风险 1：import path 大范围抖动

控制：

- 分 phase 迁移
- 先资产，后代码
- 先移动，后小修

### 风险 2：业务和适配器拆分过度

控制：

- 本轮只做 `catalog` / `packaging` 两个业务模块
- 不额外拆 `release` / `governance` / `install`

### 风险 3：CLI 被牵连

控制：

- CLI 顶层保持 `apps/skills-cli/`
- 只在路径引用层面对齐文档和脚本

## 完成定义

本轮结构整理完成时，应满足：

1. Go Catalog 服务不再主要以 `internal/catalog` / `internal/httpapi` 技术分组暴露
2. 部署资产集中到 `zarf/`
3. 操作工具集中到 `tools/`
4. CLI 仍保持独立
5. README 和生产文档的路径说明与实际目录一致
6. 测试、Docker、compose 入口仍然可验证
