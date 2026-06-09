# Unified Backend Architecture Deepening Design

## 目标

继续深化当前后端实现，在保持外部 contract 兼容的前提下，把剩余的浅 seam 收拢成更深的内部 module。

本次统一设计覆盖 5 个 workstream，并且按固定顺序推进：

1. `Catalog Query Store`
2. `Packaging Workspace Runtime`
3. `Published Release Workflow`
4. `catalogapi Server`
5. `Catalog Aggregate persistence`

本次设计保持以下外部 contract 不变：

- HTTP API contract 不变
- CLI contract 不变
- `catalog.Repository` public facade 保持兼容
- `packaging.Service` public facade 保持兼容

目标是提升：

- **locality**：把真正一起变化的查询、workspace lifecycle、publish recovery、HTTP workflow、aggregate persistence 规则集中起来
- **leverage**：caller 继续学习少量 facade interface，但内部复杂度不再散在 adapter 和 helper 中
- **testability**：测试从宽而浅的 facade / server 测试，下沉到 module seam 的 focused tests
- **AI-navigability**：未来继续修改后端时，不必跨大文件和多层 helper 才能看懂一个行为

## 非目标

- 不改变 **Package Name** 是唯一公开身份
- 不改变 **Catalog governs discovery and resolution; npm stores artifacts** 的 ADR
- 不修改现有 HTTP path、request body、response body
- 不修改 CLI command language 或 install / publish flow contract
- 不新增新的公共 Go package seam 取代 `catalog.Repository` 或 `packaging.Service`
- 不引入新的持久化后端或部署依赖

## 当前问题

### 1. `Catalog Query Store` 仍然太浅

当前 [business/catalog/sql_state_store.go](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:213) 中的多个读接口仍然采用：

1. 从 SQL 拉取一批记录
2. 在 Go 内存里排序
3. 在 Go 内存里分页

典型位置包括：

- [ListSkills](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:213)
- [ListCuratedSkills](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:259)
- [ListVersions](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:352)
- [ListGovernanceChanges](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:456)

结果是：

- `Query Store` 没真正拥有 query 语义
- 数据库 seam 只是在“供行”
- package-scoped read filtering 主要靠上层收尾，而不是靠 query seam 提供 leverage

### 2. `Packaging Workspace Runtime` 混合太多 concern

当前 [business/packaging/workspace_runtime_module.go](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:13) 同时承担：

- `Packaging Workspace` prepare / validate / pack / cleanup / prune
- managed workspace persistence
- ephemeral workspace handling
- validation cache
- bundle access
- package metadata cache
- publish 前置 materialization
- workspace-level locking

并且 `workspaceStore != nil` 的 adapter 分叉贯穿多个 workflow：

- [Prepare](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:36)
- [Validate](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:75)
- [Cleanup](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:158)
- [BundlePath](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:191)
- [BundleContent](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:230)
- [PackageMetadata](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:292)
- [PruneExpiredWorkspaces](/Users/ultraman/workspace/skill_registry/business/packaging/workspace_runtime_module.go:395)

结果是 runtime 虽然已经独立成 module，但接口深度仍不够。

### 3. `Published Release Workflow` 仍依赖错误码协议

当前 [business/packaging/published_release_workflow.go](/Users/ultraman/workspace/skill_registry/business/packaging/published_release_workflow.go:33) 通过：

- `registry_version_not_found`
- `registry_version_conflict`

这类 `catalog.Error.Code` 字符串来决定 publish recovery path。

结果是：

- 真实 workflow state 被编码进错误码
- module 之间通过隐式字符串协议协作
- 测试必须知道错误码常量，而不是知道领域状态

### 4. `catalogapi.Server` 仍然过宽

虽然前一轮深化已经抽出多个 workflow module，但 [app/catalogapi/server.go](/Users/ultraman/workspace/skill_registry/app/catalogapi/server.go:166) 仍同时持有：

- route wiring
- middleware
- request metadata
- subtree dispatch
- read query param parsing
- readiness handling
- mock registry adapter
- packaging endpoint shared helper
- registry URL rewrite

结果是 `Server` 还没有真正退化成“薄 adapter + workflow wiring”。

### 5. `Catalog Aggregate persistence` 仍是整包重写

当前 [business/catalog/sql_state_store.go](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:180) 的 `PutAggregate` 仍是：

1. 读取 revision
2. 删除 skill / versions / governance rows
3. 重新插入整个 `Agent Skill Aggregate`

关键位置：

- [PutAggregate](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:180)
- [deleteAggregateRowsTx](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:607)
- [insertAggregateTx](/Users/ultraman/workspace/skill_registry/business/catalog/sql_state_store.go:620)

这对当前规模是简单实现，但对后续的写侧 locality 和 immutable history 保真都不理想。

## 方案对比

### 方案 A：继续做局部 helper 收敛

做法：

- 不改变现有 module 结构
- 主要增加 helper 和小范围委托

优点：

- 改动路径最短
- 短期内最容易保持测试绿

缺点：

- deletion test 不通过
- 继续保留浅 seam
- 只是搬代码，不是真正 deepening

### 方案 B：保持外部 facade 不变，继续内部深挖

做法：

- `catalog.Repository` 和 `packaging.Service` 继续存在
- `Server` 继续存在
- 在它们背后继续提炼更深的 internal module / adapter seam

优点：

- 与现有仓库方向一致
- 能持续提升 locality / leverage
- 不引入 breaking change

缺点：

- 短期内会有 facade + internal module 双层结构
- 需要较强测试 discipline，防止只是换文件位置

### 方案 C：直接重塑公共 seam

做法：

- 拆掉当前 facade
- 让 caller 直接依赖新的细粒度接口

优点：

- 从接口形状看最干净

缺点：

- 会破坏 CLI / Web / HTTP adapter 的兼容面
- 超出这次“内部深化、外部兼容”的范围

### 推荐

采用 **方案 B**。

## 统一架构设计

### 1. 深化 `Catalog Query Store`

目标：

- 让 `Query Store` 真正拥有 filter / sort / page 的 query 语义
- 让 `ReadModule` 保留 read policy，而不是参与数据搬运

推荐结构：

- `ReadModule`
  - 负责 `ResolveInstallTarget`
  - 负责 read-side policy validation
  - 负责需要跨 query 结果拼装的少量规则
- `SQLStateStore`
  - 直接在 SQL 层完成排序、过滤、分页
  - 尽可能把 package-scoped 过滤前移到 query seam
  - 不再把“大列表取回内存再排序/分页”作为默认实现

不是目标：

- 不引入新公共 read repository type
- 不修改 `ReadModule` public method shape

### 2. 深化 `Packaging Workspace Runtime`

目标：

- 把 managed / ephemeral workspace 变成显式 adapter seam
- 让 runtime 不再在每个 workflow 里判断 adapter 类型

推荐结构：

- `packagingWorkspaceRuntime`
  - 负责统一 orchestration
  - 面向 workspace adapter interface 工作
- `managedWorkspaceAdapter`
  - 负责 `persist.WorkspaceStore`、managed root、bundle path persistence、cache persistence
- `ephemeralWorkspaceAdapter`
  - 负责纯文件系统临时 workspace

runtime 继续负责：

- prepare / validate / pack / cleanup / bundle access / metadata / prune
- lock lifecycle
- `bundleRuleModule` 编排

adapter 负责：

- workspace existence / load / persist
- cache persistence / lookup
- bundle path ownership
- root cleanup strategy

### 3. 深化 `Published Release Workflow`

目标：

- 让发布路径基于显式 workflow state，而不是基于错误码猜测

推荐结构：

- `publishedReleaseWorkflow`
  - 仍作为 `packaging.Service` 背后的 internal workflow
- internal release registration result
  - 例如：
    - artifact missing in registry
    - release already registered idempotently
    - immutable conflict
    - artifact already exists in package repository

具体要求：

- 可以继续复用 `catalog.Error` 对外返回
- 但 workflow 内部不再把 `Error.Code` 当成主要控制流
- 如果需要新的 sentinel error 或 typed result，只在 internal seam 内传播

### 4. 继续瘦身 `catalogapi.Server`

目标：

- 让 `Server` 更接近 route wiring + middleware + workflow wiring

推荐结构：

- 保留 `Server`
  - route registration
  - shared middleware
  - request metadata / observability
- 继续下沉：
  - read subtree query parsing / delegation
  - packaging access workflow
  - readiness workflow
  - mock registry HTTP adapter

限制：

- 不把 HTTP concern 下沉到 `business/catalog` 或 `business/packaging`
- 不改变 `NewServer(ServerConfig)` 对外构造方式

### 5. 深化 `Catalog Aggregate persistence`

目标：

- 保留 `Agent Skill Aggregate` revision contract
- 减少“每次 mutation 都整包 delete + reinsert”的实现方式

推荐结构：

- skill snapshot 写法与 immutable history 写法适度分离
- `governance_records` 尽量 append / replace targeted rows，而不是整包重建
- `skill_versions` 保留 immutable release 特征，不因无关 mutation 被反复重写

限制：

- `WriteModule` 仍然只知道 aggregate mutation rule，不知道具体 SQL 写法
- 不在这轮引入 event sourcing 或新的 persistence backend

## 文件边界

### Workstream 1: `Catalog Query Store`

- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/query_store.go`
- Test: `business/catalog/sql_repository_test.go`
- Test: `business/catalog/read_module_test.go`

### Workstream 2: `Packaging Workspace Runtime`

- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/service.go`
- Create or Modify: workspace adapter-specific file(s) under `business/packaging/`
- Test: `business/packaging/workspace_runtime_module_test.go`
- Test: `business/packaging/service_test.go`

### Workstream 3: `Published Release Workflow`

- Modify: `business/packaging/published_release_workflow.go`
- Modify if needed: internal release registration helpers in `business/packaging/`
- Test: `business/packaging/published_release_workflow_test.go`

### Workstream 4: `catalogapi Server`

- Modify: `app/catalogapi/server.go`
- Modify or Create: workflow module files under `app/catalogapi/`
- Test: `app/catalogapi/server_test.go`
- Test: module-level tests under `app/catalogapi/`

### Workstream 5: `Catalog Aggregate persistence`

- Modify: `business/catalog/sql_state_store.go`
- Modify if needed: `business/catalog/write_module.go`
- Test: `business/catalog/write_module_test.go`
- Test: `business/catalog/sql_repository_test.go`

## 迁移顺序

### 第一阶段：`Catalog Query Store`

先处理读侧，因为它最浅，并且会影响：

- Web catalog reads
- CLI discovery / install resolution reads
- 后续 read-side package filtering 收敛

这一阶段结束时，应该做到：

- 主要列表查询在 SQL 层完成排序 / 过滤 / 分页
- `ReadModule` 只保留 policy 层行为

### 第二阶段：`Packaging Workspace Runtime`

在读侧稳定后，继续收敛 runtime seam。

这一阶段结束时，应该做到：

- runtime 不再在每个方法里做 `workspaceStore != nil` 分叉
- managed / ephemeral adapter 成为显式内部 seam

### 第三阶段：`Published Release Workflow`

在 runtime 稳定后，处理 publish orchestration。

这一阶段结束时，应该做到：

- workflow 基于显式内部状态分支
- 对外返回保持兼容

### 第四阶段：`catalogapi Server`

在 `catalog` / `packaging` 内部 seam 稳定后，再瘦身 adapter。

这一阶段结束时，应该做到：

- `server.go` 主要剩 route wiring、middleware、shared metadata、shared observability

### 第五阶段：`Catalog Aggregate persistence`

最后处理写侧 persistence 深化。

这样安排的原因是：

- 前 4 个阶段先稳定调用语义
- 最后一阶段再调整底层 persistence 实现，降低中途返工概率

## 测试策略

### 1. 每个 workstream 都先补 focused tests

优先使用 package-level tests，而不是一上来依赖全仓回归。

### 2. 重点包验证

每个阶段至少覆盖：

- `go test ./business/catalog`
- `go test ./business/packaging`
- `go test ./app/catalogapi`

按实际 workstream 选择对应包做 focused 运行。

### 3. 最终回归

全部 workstream 完成后执行：

```bash
go test ./...
```

如果某阶段只触及单包，也要先跑单包 focused tests，再跑更大范围回归。

### 4. 兼容性验证

必须显式验证：

- 现有 HTTP path 仍可工作
- 现有 CLI flow 对应的服务端 read / publish contract 未变
- `catalog.Repository` / `packaging.Service` caller 不需要改调用顺序

## 风险与约束

### 1. “只是搬代码”的假深化

如果只是把代码从一个文件搬到另一个文件，但 query / adapter / workflow 语义没有集中，视为失败。

### 2. facade 兼容层过早膨胀

虽然本次允许保留 facade，但不应继续往 facade 塞新的业务细节。

### 3. 读写两侧同时大改的耦合风险

因此采用固定顺序，不并行改主 seam。

### 4. persistence 深化不能破坏 aggregate contract

即使底层 SQL 写法变化，也必须保留：

- aggregate revision conflict behavior
- immutable release conflict behavior
- install resolution 与 governance 现有语义

## 成功标准

满足以下条件时，本次统一重构才算完成：

1. 五个 workstream 都完成，且按既定顺序落地
2. HTTP API / CLI / public Go facade 无 breaking change
3. `Catalog Query Store` 不再以“全量取回 + 内存排序/分页”作为主实现
4. `Packaging Workspace Runtime` 不再让 managed / ephemeral 分支散落在每个 workflow
5. `Published Release Workflow` 不再依赖错误码字符串作为主要内部控制流
6. `catalogapi.Server` 明显缩小到 adapter 角色
7. `Catalog Aggregate persistence` 不再以整包 delete + reinsert 作为主要 mutation 写法
8. focused package tests 与最终 `go test ./...` 全部通过
