# Backend Architecture Deepening Summary

## 目标

本次后端深化只调整 module 和 seam，不改变外部 contract：

- `app/catalogapi.NewServer(ServerConfig)` 保持不变
- 现有 HTTP path、请求体、响应体保持兼容
- `business/catalog.Repository` 继续作为兼容 facade
- `business/packaging.Service` 继续作为兼容 facade

目标是把复杂度从宽而浅的 adapter / facade 收拢到更深的内部 module，提升 **locality**、**leverage**、testability 和 AI-navigability。

## 已落地结构

### 1. `app/catalogapi`

`Server` 现在主要负责 route wiring、middleware、request metadata 和 shared observability。

真实工作流分别下沉到 4 个内部 module：

- `catalogReadModule`
  - 负责 Catalog read use case 和按 principal 过滤 read projection
- `governanceModule`
  - 负责 governance change listing 和 package filter enforcement
- `packagingModule`
  - 负责 packaging workflow、idempotency replay 和 packaging audit orchestration
- `sessionModule`
  - 负责 current principal projection、login、logout cookie 和 session enablement

结果：

- `Server` 这个 HTTP Adapter 的 interface 没变，但 implementation 不再混合所有 workflow 细节
- module-level tests 可以直接覆盖 packaging/session/read seam，而不需要总是通过全量 HTTP handler 触发

### 2. `business/catalog`

`CatalogRepository` 现在是显式 facade：

- `ReadModule`
  - 持有 `QueryStore`
  - 负责 listing、detail reads、governance reads、install resolution policy
- `WriteModule`
  - 持有 `AggregateStateStore` 和 `ArtifactSource`
  - 负责 `Release Record`、`Governance Change`、seed 和 aggregate mutation retry

`Repository` interface 被拆成：

- `ReadRepository`
- `WriteRepository`
- `Repository` 继续组合两者，保持兼容

结果：

- install resolution policy 不再散落在 memory / SQL adapter
- read seam 不再依赖 aggregate mutation implementation
- aggregate write rules 的 locality 集中在 `WriteModule`

### 3. `business/packaging`

`Service` 现在是兼容 facade，内部拆成两个更深的 module：

- `workspaceSourceModule`
  - 负责 source intake、git/archive/inline materialization、`Packaging Workspace` 生命周期、bundle path ownership 和过期清理
- `bundleRuleModule`
  - 负责 target validation、workspace validation、import normalization、`package.json` / import report 写入，以及 bundle 产出规则

结果：

- prepare 阶段的 source/workspace concerns 与 validate/pack 阶段的 bundle rules 被分开
- `Service` 保留现有 public seam，但不再同时拥有所有 packaging 细节

## 为什么这次深化是值得保留的

按 deletion test 看，这些 module 不是 pass-through：

- 删掉 `catalogReadModule` / `governanceModule` / `packagingModule` / `sessionModule`，HTTP handler 会重新长回 workflow 细节
- 删掉 `ReadModule` / `WriteModule`，Catalog policy 会重新散回 repository facade 和 adapter
- 删掉 `workspaceSourceModule` / `bundleRuleModule`，`packaging.Service` 会重新变回一个混合 source intake、workspace lifecycle、validation 和 bundle rules 的宽 module

因此这次不是“把代码拆成更多文件”，而是把真正一起变化的行为重新集中起来。

## 测试面变化

新增或强化的测试面：

- `app/catalogapi`
  - route wiring 保持在 adapter 级测试
  - packaging/session/read workflow 下沉到 module 级测试
- `business/catalog`
  - `ReadModule` 独立验证 install resolution policy
  - `WriteModule` 独立验证 aggregate mutation behavior
- `business/packaging`
  - `workspaceSourceModule` 独立验证 workspace materialization
  - `bundleRuleModule` 独立验证 normalization、validation 和 bundle output

这让测试更接近 interface，而不是被迫穿过更大的 facade。

## 受影响文件

核心落地点：

- `app/catalogapi/server.go`
- `app/catalogapi/catalog_read_module.go`
- `app/catalogapi/governance_module.go`
- `app/catalogapi/packaging_module.go`
- `app/catalogapi/session_module.go`
- `business/catalog/repository.go`
- `business/catalog/catalog_repository.go`
- `business/catalog/query_store.go`
- `business/catalog/read_module.go`
- `business/catalog/write_module.go`
- `business/packaging/service.go`
- `business/packaging/workspace_source_module.go`
- `business/packaging/bundle_rule_module.go`

## 验证

已通过的定向验证：

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./app/catalogapi
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging
```

## 非目标与未做事项

这次没有做：

- 新的公共抽象层去统一 `catalogapi` 与其他 adapter 的 audit / error mapping 模式
- 全仓测试或发布流程验证
- ADR 级别的新公开架构承诺，因为本次主要是 internal seam deepening，未改变 public identity、Catalog truth 或 artifact ownership 的既有决策
