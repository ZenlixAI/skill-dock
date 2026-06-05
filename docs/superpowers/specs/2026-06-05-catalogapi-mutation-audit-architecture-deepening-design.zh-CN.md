# CatalogAPI Mutation / Audit Workflow 架构深化设计

## 目标

继续深化 `app/catalogapi`，把 `Server` 中剩余的 mutation 与 audit workflow 从 HTTP Adapter 后面移走，让 `Server` 主要负责：

- route wiring
- shared middleware
- request metadata
- shared observability

本次设计保持以下外部 contract 不变：

- `catalogapi.NewServer(ServerConfig)`
- 现有 HTTP path
- request / response body
- 现有角色授权语义

## 非目标

- 不改变 **Package Name** 是唯一公开身份
- 不改变 **Release Record**、**Governance Change**、**Packaging Workspace**、`AuditEventRecord` 的领域语义
- 不重写 `business/catalog` 或 `business/packaging` 的 public seam
- 不调整 CLI、前端或 OpenAPI 契约
- 不引入新的部署依赖

## 当前问题

### 1. `Server` 仍然是宽而浅的 Adapter

虽然 `catalogReadModule`、`sessionModule`、`packagingModule` 已经落地，但 [server.go](/Users/ultraman/workspace/skill_registry/app/catalogapi/server.go:369) 仍直接持有多类 workflow：

- `AuditEventRecord` 查询
- `Release Record` 创建
- skill / version 级 `Governance Change`
- `Packaging Workspace` cleanup
- bundle download
- shared authorization denial + audit ceremony

结果是：

- Adapter 继续知道 decode、policy、status mapping、audit details
- mutation 行为没有一个稳定的 module seam
- 相关测试仍大量停留在大而宽的 `server_test.go`

### 2. 写侧 workflow 重复拼装同一种 ceremony

`release record`、`skill governance`、`version governance`、`packaging cleanup`、bundle download 都重复类似步骤：

1. decode / validate request
2. 调用 repo 或 packaging facade
3. 计算状态码
4. 记录成功 / 失败 audit
5. 写出 JSON 或二进制响应

这些步骤属于一类真实 workflow，但当前散落在多个 handler 中。

### 3. 授权拒绝路径是 shared behavior，却仍嵌在 Adapter

`authorizeActionAccess`、`authorizePackageAccess`、`denyAuthorization` 位于 [server.go](/Users/ultraman/workspace/skill_registry/app/catalogapi/server.go:1101)，负责：

- action / package 授权
- observability decision
- auth denied audit
- 统一错误写出

这本质上是一个 shared access workflow，而不是 route wiring 细节。

## 方案对比

### 方案 A：继续在 `server.go` 内用 helper 函数收敛

做法：

- 保持单文件
- 增加更多 `handleXxx` helper

优点：

- 改动路径短
- 文件数更少

缺点：

- deletion test 不通过
- `Server` 仍然拥有 workflow 细节
- 测试面不会明显改善

### 方案 B：按 workflow 深化成内部 module

做法：

- `catalogMutationModule`
- `auditQueryModule`
- `packagingAccessModule`
- `accessWorkflow`

优点：

- locality 最好
- Adapter 的 interface 更小
- module-level tests 可以直接覆盖成功 / 失败 / denied / audit ceremony

缺点：

- 需要先补 module test seam
- `server.go` 需要一轮委托迁移

### 方案 C：把逻辑直接下沉到 repo 或 packaging facade

做法：

- 把 audit/status/HTTP concerns 一并压到 `business/catalog` 或 `business/packaging`

优点：

- `server.go` 会更薄

缺点：

- 违反当前分层
- 会把 HTTP concern 泄漏到 business module
- 与前 3 轮深化方向不一致

### 推荐

采用 **方案 B**。

## 推荐结构

### 1. `catalogMutationModule`

负责：

- `Release Record`
- skill 级 `Governance Change`
- version 级 `Governance Change`

这个 module 持有：

- request decode / path version decode
- repo 调用
- success / failure audit orchestration
- `201` / `200` / error status 选择

它不负责：

- route matching
- global authentication
- generic JSON encoding

### 2. `auditQueryModule`

负责：

- `AuditEventRecord` 查询 workflow
- query param parse
- package filter enforcement
- package-scoped principal 约束
- audit store query execution

这个 module 的价值在于把 `audit.read` 从单个大 handler 提炼成稳定 seam，使 audit filter 行为可以独立测试。

### 3. `packagingAccessModule`

负责：

- `Packaging Workspace` cleanup
- bundle download

这个 module 持有：

- request decode / query param validation
- managed workspace access checks
- bundle file read path
- success / failure audit details

它与现有 `packagingModule` 区分开：

- `packagingModule` 负责 prepare / validate / pack workflow
- `packagingAccessModule` 负责已生成 workspace / bundle 的访问与清理

### 4. `accessWorkflow`

负责 shared authorization ceremony：

- action authorization
- package authorization
- denied response mapping
- denied audit
- auth decision observability 写入

这样 `Server` 不再直接知道 denied path 的完整 ceremony，只负责在 route 上声明“这里需要 package access / action access”。

## 文件边界

建议文件结构：

- Modify: `app/catalogapi/server.go`
  - 保留 route wiring、middleware、shared helpers、module wiring
- Create: `app/catalogapi/catalog_mutation_module.go`
  - `Release Record` + `Governance Change` workflow
- Create: `app/catalogapi/audit_query_module.go`
  - `audit.read` workflow
- Create: `app/catalogapi/packaging_access_module.go`
  - cleanup + bundle download
- Create: `app/catalogapi/access_workflow.go`
  - shared authz denial workflow
- Modify: `app/catalogapi/module_test_helpers_test.go`
  - 提供新 module seam 的 test stub
- Modify: `app/catalogapi/server_test.go`
  - 保留 route wiring / adapter coverage
- Create: `app/catalogapi/catalog_mutation_module_test.go`
- Create: `app/catalogapi/audit_query_module_test.go`
- Create: `app/catalogapi/packaging_access_module_test.go`

## 测试策略

### Adapter 级测试保留

`server_test.go` 继续覆盖：

- route wiring
- auth middleware integration
- request metadata / observability 的最小 smoke coverage

### Module 级测试新增

#### `catalogMutationModule`

覆盖：

- `Release Record` 成功创建
- `Release Record` idempotent replay
- `Release Record` immutable conflict
- skill governance success / invalid transition
- version governance success / invalid transition
- 对应 audit event 的 success / failure details

#### `auditQueryModule`

覆盖：

- package-scoped principal 必须带 `package_name`
- 非法 `auth_decision` / `version` filter
- audit store 缺失 / 错误传播
- 成功 query 的 filter 组合

#### `packagingAccessModule`

覆盖：

- cleanup success / missing workspacePath / cleanup failure
- bundle download success / bundle missing / invalid query
- package access denied 时的拒绝路径
- 对应 audit event 的 success / failure details

## 兼容性要求

实现时必须保持：

- 所有 HTTP path 不变
- 所有 JSON schema 不变
- `AuditEventRecord` 字段含义不变
- `Release Record` 的 `201/200` 语义不变
- denied path 的错误码和审计语义不变

## 风险

### 1. denied path 回归

因为 authz denial 既影响 response，也影响 audit / observability，所以最容易在迁移时漏掉。

缓解：

- 先把 denied workflow 抽成独立 seam
- 用现有 package-level authz 测试锁住行为

### 2. mutation audit detail 丢失

`release record`、`governance`、`packaging cleanup` 的 success / failure details 不同，提炼时容易被“通用化过度”。

缓解：

- 通用只做到 workflow 骨架
- 具体 detail payload 继续由具体 module 方法决定

### 3. `Server` 再次变成薄 facade 但 module 只是 pass-through

如果 module 只是包一层 repo 调用，这次深化就没有意义。

缓解：

- module 必须真正持有 decode / audit / status / policy ceremony
- deletion test 不通过的拆分不接受

## 验证标准

本轮设计完成后，下一步 implementation plan 需要证明：

- `Server` 不再直接持有 `release record`、`governance change`、`audit.read`、`packaging cleanup/download` 的完整 workflow
- route wiring 仍保持在 `Server`
- 相关行为的主要测试面从 `server_test.go` 下沉到新 module tests
- `go test ./app/catalogapi` 通过
- 最好补一次 `go test ./...` 确认无回归
