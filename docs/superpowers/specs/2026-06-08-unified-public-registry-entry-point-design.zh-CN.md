# 统一 Public Registry Entry Point 设计

## 摘要

当前系统对外暴露两类入口：

- Catalog API，例如 `https://skills.example.com/api`
- Nexus npm repository，例如 `https://nexus.example.com/repository/skills-npm/`

这导致外部 CLI 使用者必须自己分辨两套地址：

- `SKILLS_REGISTRY_URL` / `skills ... --registry ...` 指向 Catalog
- `npx --registry ...` / `npm install -g skills --registry ...` 指向 Nexus

这个分裂和现有项目语言不完全一致。已有约束已经把 `registry` 收敛为一个 **Registry Entry Point**，由系统内部派生 package repository 细节，而不是让用户自己管理 Catalog 与 npm 仓的分工。

本设计将公开入口统一为同一个外部 host/path 空间：

- `https://skills.example.com/api/...`
- `https://skills.example.com/repository/skills-npm/...`

统一动作优先放在 ingress / gateway / reverse-proxy 层完成，而不是让 Catalog 进程直接承担 npm reverse proxy 职责。

## 目标

- 让外部用户只记住一个公共 host。
- 让 `npx --registry ...` 可以使用与 Catalog 相同的公共域名/端口。
- 让 Catalog 对外返回的 npm registry / tarball URL 也统一落在同一个公共入口下。
- 保持内部架构边界清晰：
  - Catalog 继续做 product truth / API
  - Nexus 继续做 npm-compatible artifact store
  - ingress / gateway 负责公开路由统一

## 非目标

- 不在本次设计里让 Catalog 进程成为 npm repository server 或 full reverse proxy。
- 不修改 Nexus 内部存储、publish 协议或 Catalog release-record 语义。
- 不强制老部署立即迁移；允许继续使用独立 Nexus 域名作为兼容模式。
- 不在本次设计里引入多仓路由或多 package repository 选择。

## 背景与现状

现有代码和文档里已经有两类 URL：

- `SKILL_REGISTRY_NPM_REGISTRY_URL`
  - Catalog Pod 内部访问 Nexus 的地址
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
  - Catalog 返回给外部 CLI 的 npm public URL

当前默认和示例仍普遍把 public npm URL 设成独立 Nexus 地址，例如：

- `https://nexus.example.com/repository/skills-npm/`

因此即便 Catalog 已经是统一控制面，最终对用户暴露的入口仍是“两套地址”。

这在以下场景里持续制造心智负担：

- `npx --registry ... skills --help`
- `npm install -g skills --registry ...`
- CLI publish/import 时的 npm registry 指导
- Catalog 返回 tarball public URL 给 CLI / Web

## 当前摩擦

### 1. 用户需要记两套入口

外部用户必须理解：

- `skills ... --registry` 是 Catalog
- `npm/npx --registry` 是 Nexus

这与“Registry Entry Point”概念相冲突。

### 2. 文档和部署示例传播了双入口模型

当前 runbook / k8s 示例会同时要求：

- 一个 `skills.example.com`
- 一个 `nexus.example.com`

这会把“内部后端分层”泄露成“外部用户必须知道的系统细节”。

### 3. 如果将统一逻辑塞进应用层，会污染 Catalog 边界

把 `/repository/...` 代理直接塞进 Go Catalog 服务虽然可行，但会让它额外承担：

- 流式转发
- header/caching/range 行为兼容
- npm registry proxy 故障面

这会让应用边界变浅。

## 备选方案

### 方案 A：Ingress / Gateway 统一入口

对外保持一个 host，在同一入口下分 path：

- `/api` -> Catalog
- `/repository/skills-npm` -> Nexus

优点：

- 最符合用户体验目标
- 不需要让 Go 应用承担 npm proxy 协议细节
- 与当前领域边界一致

缺点：

- 改动重心在部署层、chart、文档
- 依赖 ingress / gateway 具备 path-based routing 能力

### 方案 B：Catalog 应用层代理 `/repository/...`

由 Catalog 服务本身接收 `/repository/...` 并转发到 Nexus。

优点：

- 单服务看起来更集中
- 对没有现成 ingress 能力的简单环境更直观

缺点：

- Catalog 服务职责变脏
- 代理语义、流量、故障都压到应用层
- 长期不利于清晰架构

### 方案 C：只优化文案，不改入口

继续保留双入口，只通过 `make` / 文档隐藏复杂度。

优点：

- 变更最小

缺点：

- 不能真正消除双入口
- 用户问题本质上仍然存在

## 决策

采用 **方案 A**。

实现上将统一入口放在 ingress / gateway / reverse-proxy 层，不在本次实现里引入 Catalog 内建 npm reverse proxy。

同时将工作拆成两个阶段：

- **Phase A**：统一 CLI 包安装入口
- **Phase B**：统一 Catalog 对外暴露的 npm registry / tarball URL

## 设计

### 公开入口模型

统一后的对外入口模型：

- Catalog API：`https://skills.example.com/api/...`
- npm repository：`https://skills.example.com/repository/skills-npm/...`

内部后端模型保持不变：

- `/api` 请求进入 Catalog Pod
- `/repository/skills-npm` 请求进入 Nexus Service

这意味着“一个 Registry Entry Point”是公开模型，不是“一个进程处理所有协议”。

### 内外地址分离

保留两类配置并明确职责：

- `SKILL_REGISTRY_NPM_REGISTRY_URL`
  - 仅供 Catalog Pod 内部访问 Nexus
  - 例如：`http://nexus.skill-registry.svc.cluster.local:8081/repository/skills-npm/`
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
  - 仅供 Catalog 返回给外部 CLI / Web
  - 新推荐形态：`https://skills.example.com/repository/skills-npm/`

这样既能维持集群内可达性，也能统一外部入口。

## Phase A：统一 `npx --registry` / `npm install -g` 公共入口

### 目标

让用户能直接使用：

```bash
npx --registry https://skills.example.com/repository/skills-npm/ skills --help
```

以及：

```bash
npm install -g skills --registry https://skills.example.com/repository/skills-npm/
```

### 变更范围

- `zarf/k8s/k3s-all-in-one.yaml`
- `zarf/k8s/customer-k3s-example.yaml`
- `zarf/k8s/dev/kind-stack.yaml`
- `zarf/charts/skill-registry` 中与 ingress / ingressRoute / nexus public URL 相关的 values 和模板
- `docs/k3s-all-in-one.zh-CN.md`
- `docs/customer-k3s-deployment.zh-CN.md`
- 其他引用公开 Nexus 地址的部署文档

### 行为

- 对现有公共 host 增加 `/repository/skills-npm` 路由
- 路由目标为 Nexus Service
- 默认 / 示例配置中的 `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 改成统一入口 path，而不是独立 Nexus 域名
- 所有安装 `skills` CLI 的示例都切换到统一入口

### 结果

用户安装 CLI 包时，不再需要理解为什么必须换到另一套 host。

## Phase B：统一 Catalog 对外回传的 public npm URL

### 目标

让 Catalog 返回给 CLI / Web 的 npm registry / tarball URL 也全部使用统一入口，例如：

- `https://skills.example.com/repository/skills-npm/`

### 变更范围

- `cmd/skill-registry/main.go`
- `app/catalogapi` 中对外构造 publish/install registry 信息的逻辑
- 对应 server tests
- CLI / Web / 文档中引用 public npm URL 的示例

### 行为

- Catalog 继续从环境读取 `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
- 但推荐值改成统一入口 path
- 所有对外返回值继续复用该 public URL，不自行猜测 Nexus 外网地址

### 结果

以下能力会自然对齐到同一个公开入口：

- publish guidance
- install resolution
- tarball public URL
- Web / CLI 中展示给用户的 npm registry 地址

## 部署层设计

### K3s / 静态 YAML

静态清单需要把对外入口从“两个独立 Ingress / Host”调整为“同 Host 多 Path”或功能等价的 route 规则：

- `/api` -> skills-web / catalog 现有路径
- `/repository/skills-npm` -> Nexus

### Helm chart

Chart 需要支持两种模式：

1. 推荐模式：统一 public host
2. 兼容模式：独立 public Nexus host

建议 values 层显式表达：

- 是否启用 unified public registry path
- 统一 host/path 是什么
- 兼容模式下是否继续单独启用 Nexus ingress

### 兼容模式

如果用户已经部署：

- `skills.example.com`
- `nexus.example.com`

则允许继续保留：

- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL=https://nexus.example.com/repository/skills-npm/`

本次变更不强制迁移，只修改默认推荐方案和官方示例。

## 错误处理

- 不在 Catalog 应用层做 `/repository/...` fallback proxy
- 如果 ingress / gateway 未正确配置，外部应直接看到明确的 404 / 502 / route error
- 文档应把该失败明确归类为“public route misconfigured”，而不是 Catalog 业务错误

## 测试策略

### 配置与模板测试

验证：

- ingress / ingressRoute 渲染后存在 `/repository/skills-npm` 到 Nexus 的公开路由
- 统一入口模式下，默认 public npm URL 与路由一致

### Server 测试

验证：

- Catalog 对外返回值继续使用 `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
- 改为统一入口 path 后，现有 publish/install 语义不变

### 文档验收

至少覆盖：

```bash
npx --registry https://skills.example.com/repository/skills-npm/ skills --help
```

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com/api
skills find pdf
```

```bash
skills publish /abs/path/to/skill
```

验收标准：

- 用户只需要记一个公共 host
- 用户仍能区分 `/api` 与 `/repository/...`，但不再需要区分两套外部地址

## 风险与缓解

### 风险 1：Ingress 能力差异

不同现场可能用：

- 标准 Ingress
- Traefik IngressRoute
- 其他网关

缓解：

- 保留“兼容模式”配置
- 文档明确给出统一入口模式与独立 Nexus 模式两套配置方式

### 风险 2：把应用层和部署层边界重新做浅

如果后续为了“省事”把 `/repository` 代理写回 Catalog，边界会重新变脏。

缓解：

- 在 spec 与后续 plan 中显式写明：应用层代理仅作为备选，不是主路径

### 风险 3：已有文档和示例地址不一致

如果只改 YAML 不改文档，用户仍会沿着旧的双入口路径走。

缓解：

- Phase A 把文档作为正式变更范围的一部分

## 实施顺序

1. 更新 spec 并获批
2. 写 implementation plan
3. 实现 Phase A
4. 验证 CLI 包安装入口
5. 实现 Phase B
6. 验证 Catalog 对外回传的 public npm URL 与统一入口一致

## 开放问题

无。

这个设计已经明确选择：

- 统一外部入口
- 保留内部 Catalog / Nexus 分工
- 优先改 ingress / gateway / docs / public URL 配置
- 不把 npm reverse proxy 主逻辑塞进 Go 应用
