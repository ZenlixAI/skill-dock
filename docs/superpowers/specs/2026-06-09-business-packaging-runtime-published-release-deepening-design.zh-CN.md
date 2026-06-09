# Business Packaging Runtime / Published Release Deepening Design

## 目标

继续深化 `business/packaging`，把当前兼容 facade `Service` 背后的实现复杂度收拢到两个更深的 module：

- `Packaging Workspace Runtime`
- `Published Release Workflow`

本次调整保持以下外部 contract 不变：

- `packaging.Service` 的现有 public method 保持兼容
- `PrepareRequest` / `ValidateRequest` / `PackRequest` / `PublishRequest` 等输入输出结构保持兼容
- `app/catalogapi`、CLI、Web 不需要因为这次深化修改调用顺序

目标是提升：

- **locality**：`Packaging Workspace` 生命周期、bundle access、cache、publish retry 规则集中在少数 module
- **leverage**：caller 继续学习少量 facade interface，但内部复杂 workflow 不再散在单个文件
- **testability**：测试面从 `Service` 大而宽的 facade 下沉到真正一起变化的 workflow
- **AI-navigability**：未来修改 managed workspace 或 publish 规则时，不必在整个 `service.go` 里跳转

## 非目标

- 不改变 **Package Name**、**Skill Bundle**、**Release Record**、**Published Release** 的领域语义
- 不重写 `workspaceSourceModule` 或 `bundleRuleModule` 的核心 bundle 规则
- 不调整 HTTP path、CLI contract 或 Web adapter
- 不引入新的持久化后端
- 不把 `Service` 立即拆成新的 public seam

## 当前问题

### 1. `Service` 仍是宽而浅的 facade

当前 [business/packaging/service.go](/Users/ultraman/workspace/skill_registry/business/packaging/service.go:18) 同时持有：

- `Packaging Workspace` prepare / materialize / cleanup / prune
- managed workspace persistence
- validation cache
- bundle handle / bundle content / package metadata access
- `Published Release` orchestration
- workspace-level locking

这些行为都挂在同一个 interface 后面，导致 interface 学起来不小，而 implementation 更是被多套 concern 混在一起。

### 2. `managed` / `unmanaged` adapter 分叉散落在每个 workflow

现在 `workspaceStore != nil` 这条分支贯穿：

- `PrepareContext`
- `ValidateContext`
- `CleanupContext`
- `BundlePathContext`
- `BundleContentContext`
- `PackageMetadataContext`
- `packWorkspaceLocked`
- `PruneExpiredWorkspaces`

caller 虽然只经过一个 seam，但 implementation 内部几乎每个 workflow 都要重新判断 adapter 路径，locality 很差。

### 3. `Published Release` workflow 被埋在 `PublishContext`

当前 publish 流程混合了：

1. 通过 `Packaging Workspace` 做 pack
2. 处理 bundle validation failure
3. 先尝试写 `Release Record`
4. 如果 catalog 报 `registry_version_not_found`，则发 package repository publish
5. 处理 `registry_version_conflict`
6. 重试 `Release Record`

这是一条真实 workflow，但目前只是 facade 的一个大方法，而不是稳定 module。

### 4. 测试主要仍围绕 facade

`service_test.go` 很有价值，但当前 test surface 更接近“整个 facade 有没有坏”，而不是：

- `Packaging Workspace Runtime` 是否正确处理 managed / unmanaged adapter
- `Published Release Workflow` 是否正确处理 idempotent replay 和 publish recovery

## 方案对比

### 方案 A：继续在 `Service` 内用 helper 收敛

优点：

- 改动路径短
- public seam 不动

缺点：

- deletion test 不通过
- `Service` 仍知道所有 workflow 细节
- `managed` / `unmanaged` 分叉仍会继续散落

### 方案 B：保留 facade，新增两个深 module

优点：

- public seam 保持稳定
- locality 最好
- module-level tests 可以直接覆盖真实 workflow

缺点：

- 需要一轮委托迁移
- 短期内会存在 facade + internal module 双层结构

### 方案 C：直接拆 public seam

优点：

- interface 最干净

缺点：

- 影响 `catalogapi` 和其他 caller
- 超出本轮 internal deepening 范围

### 推荐

采用 **方案 B**。

## 推荐结构

### 1. `Packaging Workspace Runtime`

这是 `Packaging Workspace` 相关行为的主要 module。

它的 interface 负责：

- `Prepare`
- `Validate`
- `Pack`
- `Cleanup`
- `BundlePath`
- `BundleContent`
- `PackageMetadata`
- `PruneExpiredWorkspaces`

它的 implementation 负责：

- `Packaging Workspace` prepare / materialize
- managed 和 ephemeral 两个 adapter 选择
- validation cache
- bundle reuse
- bundle handle / bundle content access
- package metadata cache
- workspace lock
- expiry prune

它依赖但不重新定义：

- `workspaceSourceModule`
- `bundleRuleModule`
- `persist.WorkspaceStore`

### 2. `Published Release Workflow`

这是把一个已打包 workspace 变成 **Published Release** 的 module。

它的 interface 只负责：

- `Publish(ctx, PublishRequest)`

它的 implementation 负责：

- 调用 `Packaging Workspace Runtime` 完成 pack 前置条件
- 在 validation failed 时返回一致错误
- 先尝试 `Release Record`
- 在 `registry_version_not_found` 时发 package repository publish
- 处理 `registry_version_conflict` 下的恢复路径
- 最终返回 `PublishResponse`

它依赖：

- `Packaging Workspace Runtime`
- `releaseRecordCreator`
- `packagePublisher`

### 3. `Service` 退化为兼容 facade

`Service` 继续保留现有 public seam，但 implementation 只负责：

- 组装 `Packaging Workspace Runtime`
- 组装 `Published Release Workflow`
- 兼容旧 public method，并委托给上述 module

这让 caller 不变，但内部复杂度重新集中。

## 文件边界

建议结构：

- Modify: `business/packaging/service.go`
  - 保留 `Service`、构造 wiring、兼容委托
- Create: `business/packaging/workspace_runtime_module.go`
  - `Packaging Workspace Runtime`
- Create: `business/packaging/published_release_workflow.go`
  - `Published Release Workflow`
- Keep: `business/packaging/workspace_source_module.go`
  - 继续作为 runtime 内部依赖
- Keep: `business/packaging/bundle_rule_module.go`
  - 继续作为 runtime 内部依赖
- Keep: `foundation/persist/workspace_store.go`
  - 继续作为 persisted adapter seam

## 迁移策略

### 第一步：先抽 `Packaging Workspace Runtime`

先把以下逻辑从 `Service` 移走：

- `PrepareContext`
- `ValidateContext`
- `CleanupContext`
- `BundlePathContext`
- `BundleContentContext`
- `PackageMetadataContext`
- `packWorkspaceLocked`
- `PruneExpiredWorkspaces`

要求：

- 行为保持兼容
- 先让 `Service` 委托给 runtime
- 不在这一步调整 publish 流程

### 第二步：再抽 `Published Release Workflow`

把当前 `PublishContext` 里的 orchestration 移到独立 module。

要求：

- publish 成功 / replay / conflict recovery 行为不变
- `Service.PublishContext` 只负责调用 workflow

### 第三步：瘦身 facade

删除 `Service` 内不再承担的 helper，把 remaining helper 只留给 wiring 或 facade-level compatibility。

## 测试策略

### 保留 facade 级 smoke coverage

`business/packaging/service_test.go` 继续覆盖：

- public seam 仍可用
- 构造和兼容委托仍然成立

但它不再承担大部分细节验证。

### 新增 `Packaging Workspace Runtime` tests

覆盖：

- managed adapter 的 validation cache 命中 / 未命中
- bundle reuse
- bundle content 缺失时的 pack-on-demand
- package metadata cache
- cleanup / prune
- managed 与 ephemeral adapter 的行为对齐

### 新增 `Published Release Workflow` tests

覆盖：

- `Release Record` 已存在时直接 replay
- `registry_version_not_found` 时先 publish 再登记
- `registry_version_conflict` 时恢复到 replay / existing release
- validation failed 时不应 publish
- publish workflow 未配置时返回一致错误

## 为什么这个深化值得保留

按 deletion test：

- 删掉 `Packaging Workspace Runtime`，`Service` 会重新长回 managed / unmanaged 分叉、cache、bundle access、lock 和 prune 细节
- 删掉 `Published Release Workflow`，`PublishContext` 会重新长回 publish recovery 规则

这说明它们不是 pass-through，而是能把复杂度真正集中起来的 module。

## 风险与控制

风险：

- internal module 数量增加，短期阅读路径变多
- runtime 与 publish workflow 的职责如果切得不清，可能出现双重 orchestration

控制：

- runtime 只拥有 `Packaging Workspace` 相关规则
- publish workflow 只拥有 **Published Release** 相关规则
- `Service` 不再重新实现下沉后的 workflow

## 验证计划

本设计进入实现后，至少要运行：

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./app/catalogapi
```

第二条不是因为本轮直接改 `catalogapi`，而是为了验证 `packaging.Service` 兼容 facade 没有破坏上层 caller。
