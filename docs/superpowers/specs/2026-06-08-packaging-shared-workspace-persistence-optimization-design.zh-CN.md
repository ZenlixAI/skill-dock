# Packaging Shared Workspace Persistence Optimization

## 背景

当前 shared workspace 持久化链路采用“整份 workspace snapshot blob + 可选 bundle blob”的模型：

- `Prepare` 会把整个 workspace 打成 tar.gz snapshot 后写入 DB
- `Validate`、`Pack`、`PackageMetadata` 会反复把 snapshot 读回、解压到临时目录，再在临时目录上工作
- `Pack` 之后还会把整份 bundle 内容再读回内存并写入 DB

这个实现解决了多副本共享正确性，但代价是：

- 重复解压、重复扫描导致请求时延和 CPU 成本偏高
- 临时磁盘和内存放大明显
- DB 被迫承载大 blob 读写，不适合作为长期的 workspace 主存储

本次设计的目标是：优先降低重复解压带来的时延/CPU 成本，同时顺手减少 DB snapshot/blob 的大对象 I/O。

## 范围与非目标

### 范围

- 优化 `business/packaging` 的 shared workspace 持久化模型
- 让同一个 `workspaceId` 在 `prepare -> validate -> pack -> metadata -> cleanup` 生命周期内复用同一份 managed workspace
- 将 bundle 产物改为共享目录内的文件引用，而非 DB blob 主存储
- 为部署增加可配置的 shared workspace root

### 非目标

- 不把这次改造扩展成对象存储、远端 artifact service 或通用文件服务
- 不做历史 workspace 的在线迁移或自动升级
- 不改动对外 API 字段名或 Published Release orchestration

## 方案概览

把 persisted workspace 从“DB 存 snapshot/blob”改成“共享目录存真实 workspace 与 bundle，DB 仅存 metadata/路径引用”。

核心变化：

1. `Prepare`
   - source 导入和 normalize 之后，不再 archive workspace
   - 直接把准备好的 workspace 移入 managed shared root，例如：
     - `<workspaceRoot>/<workspaceId>/workspace/`
   - `WorkspaceStore` 只保存 `workspaceId -> managed workspace path`

2. `Validate` / `PackageMetadata`
   - 不再从 snapshot materialize 临时目录
   - 直接在已持久化的 managed workspace 上运行

3. `Pack`
   - bundle 写入固定目录，例如：
     - `<workspaceRoot>/<workspaceId>/bundle/<package>-<version>.tgz`
   - `WorkspaceStore` 保存 `bundle_path`
   - `BundlePath` / `BundleContent` 直接读取这个共享目录里的 bundle 文件

4. `Cleanup`
   - 删除 store record
   - 删除 `<workspaceRoot>/<workspaceId>/` 整个目录树

## 数据模型

### WorkspaceStore 接口

`persist.WorkspaceStore` 从 blob 仓库语义改成 metadata registry 语义。

建议接口形态：

- `SaveWorkspace(ctx, workspaceID, workspacePath, createdAt)`
- `LoadWorkspace(ctx, workspaceID) -> WorkspaceRecord`
- `SaveBundle(ctx, workspaceID, bundlePath, updatedAt)`
- `LoadBundle(ctx, workspaceID) -> BundleRecord / bundlePath`
- `DeleteWorkspace(ctx, workspaceID)`
- `DeleteExpired(ctx, cutoff)`

### WorkspaceRecord

建议保留并强化这些字段：

- `WorkspaceID`
- `WorkspacePath`
- `BundlePath`
- `CreatedAt`
- `UpdatedAt`

`Snapshot` 和 `Bundle []byte` 不再作为主字段参与正常读写。

### SQL 表

`packaging_workspaces` 保留：

- `workspace_id`
- `created_at`
- `updated_at`

新增或切换使用：

- `workspace_path TEXT NOT NULL`
- `bundle_path TEXT`

旧列：

- `workspace_snapshot`
- `bundle_content`

本次先停止使用，不要求 destructive migration；后续可单独清理。

## 业务流细节

### Prepare

- 生成 `workspaceId`
- 将导入后的 workspace 移动到 managed shared root
- 写入 `workspace_path`
- 返回给调用方的仍然是 opaque `workspaceId`

要求：

- `workspace_path` 必须始终落在配置的 shared root 下
- 目录创建与移动必须是原子地“要么成功落盘，要么失败回滚”

### Validate

- 通过 `workspaceId` 读取 `workspace_path`
- 直接在 managed workspace 上执行 validate
- 不创建临时 materialize 目录

### Pack

- 通过 `workspaceId` 读取 `workspace_path`
- 在固定 bundle 目录输出 tgz
- 若 bundle 已存在且 workspace 未变化，可直接复用
- 把 `bundle_path` 保存到 store

### BundlePath / BundleContent

- `BundlePath` 返回 handle 风格地址，例如现有 `workspace://...`
- 内部解析到 `bundle_path`
- `BundleContent` 直接从共享目录读文件
- 不再因为 `bundle_content` 缺失而重新走 DB blob 逻辑

### Cleanup

- 先删除 DB record，再删除 managed workspace 目录，或反之做一致性保护
- 删除失败时返回明确的 internal error，并避免留下半删除状态

## 配置

新增配置：

- `SKILL_REGISTRY_WORKSPACE_ROOT`

默认值建议：

- `data/packaging-workspaces`

要求：

- 本地开发默认可用
- k8s / 多副本部署时可直接挂持久卷
- `main.go` 启动时负责把这个 root 传入 packaging service

## 兼容与迁移策略

本次采用“向前切换，不迁移旧记录”的策略。

对旧格式 workspace（只有 snapshot/blob，没有 `workspace_path`）：

- 直接返回明确错误
- 提示调用方重新 `prepare`

原因：

- 这样能把这次改造聚焦在性能收益最大的主路径
- 避免为了兼容旧数据重新引入 materialize fallback，稀释本次优化收益

## 错误处理

- `workspaceId` 不存在：维持现有 `invalid_request`
- `workspace_path` 或 `bundle_path` 指向的文件系统对象缺失：返回可诊断的 internal error 或 bundle_not_found
- 若发现 `workspace_path` 不在 configured root 下：视为数据损坏并拒绝访问
- 所有文件系统操作继续透传 `ctx`，避免请求取消后后台继续长时间 I/O

## 测试策略

### Store 测试

- 保存 workspace 时记录的是 `workspace_path`，不是 snapshot bytes
- 保存 bundle 时记录的是 `bundle_path`，不是 bundle bytes
- 加载不存在记录时仍返回现有 not found 语义

### Service 测试

- `Prepare` 后 workspace 被移动到 managed root
- `Validate` / `PackageMetadata` / `Pack` 复用同一 managed workspace
- `Pack` 产出的 bundle 可重复读取，不触发二次打包
- `Cleanup` 后 workspace 目录与 bundle 文件一并删除
- 旧格式 record 会被拒绝，并提示重新 prepare

### 回归测试

- 现有 `workspace://...` bundle 下载路径仍可工作
- Web / CLI 无需改动契约即可继续使用 `workspaceId`

## 实施顺序

1. 先写 store 层失败测试，定义 path-based metadata 契约
2. 再改 `workspace_store.go` 与 migration
3. 改 `service.go` / `workspace_persistence.go`，移除 snapshot materialize 主路径
4. 接上 `SKILL_REGISTRY_WORKSPACE_ROOT`
5. 补 service/store 回归测试

## 风险

- 这次改造把 shared workspace 的可用性从“DB 大 blob”转成了“shared filesystem root”，部署上必须确保多副本看到的是同一持久卷
- 旧 workspace 记录不会自动升级，切换后旧 handle 可能需要重新 prepare
- 如果 cleanup 处理不严谨，可能留下孤儿目录；需要在测试里覆盖

## 预期收益

- `validate` / `pack` / `metadata` 不再重复整包解压
- DB 不再承载 workspace snapshot 与 bundle blob 的主路径流量
- 请求时延、CPU、临时磁盘占用都会显著下降
- shared workspace seam 更接近长期可扩展的控制面设计
