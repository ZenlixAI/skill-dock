# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> 面向可复用 Agent Skill 的自托管注册中心控制平面。

Skill Registry 文档只聚焦两个部分：**skills server** 和 **`skills` CLI**。系统保持 Catalog 作为事实来源，Bundle 存在外部 npm 兼容制品仓库中，CLI 负责安装、刷新、发布和导入。

## 文档范围

本文档集只讨论：

- **skills server**：发现、元数据、发布记录、可见性、治理
- **`skills` CLI**：安装、刷新、发布、导入
- **外部制品仓库**：可替换的 bundle 存储
- 运行这些组件所需的部署和配置

本组文档不展开 UI 细节。

## 产品形态

- **Catalog / server**：负责搜索、元数据、发布可见性和生命周期状态
- **CLI**：负责安装、刷新、发布和导入
- **制品仓库**：负责存放不可变 bundle
- **数据库**：负责持久化 catalog 状态、发布历史和审计数据

一个 skill 以 package name 和 versioned release 作为身份标识。Server 决定什么可以发布、可见和可安装；制品仓库只负责存储 bundle。

## 核心工作流

- 通过 `skills find` 发现技能
- 通过 `skills add` 安装技能
- 通过 `skills update` 刷新 registry 安装
- 通过 `skills sync` 刷新 source 安装
- 通过 `skills publish` 发布本地技能
- 通过 `skills import-local` 和 `skills import-git` 导入本地或 Git 来源
- 通过 `skills inspect` 和 `skills init` 检查和初始化技能目录

## 安装行为

registry 安装可以分成两种访问模式：

- `proxy`：server 会把 tarball URL 改写成自己的 npm proxy mount
- `direct`：server 会原样返回上游 npm 兼容 registry URL，因此 CLI 可以使用用户自己的 npm 配置和凭据

这只是安装解析的传输选择，不是新的 skill 身份模型。当 server 返回 `direct` 时，CLI 可以读取用户 npm 配置文件（`NPM_CONFIG_USERCONFIG` 或工作目录下的 `.npmrc`）来下载 tarball。对于 `proxy` 目标，CLI 不会使用 npm 配置。

## 本地开发

推荐使用仓库提供的开发脚本。

```bash
make dev
make dev-images
make down
```

`make dev` 会创建本地环境并初始化示例数据。`make dev-images` 适用于 kind 集群已经存在、但你只想重新加载已构建的服务镜像而不重新执行数据初始化的场景。

默认本地栈使用 catalog server、PostgreSQL 和外部 Artifact Keeper 制品仓库。

## 文档

- [产品方向与范围](docs/00-product-direction.md)
- [Server 架构与配置](docs/01-server-architecture.md)
- [CLI 协议与命令模型](docs/02-cli-contract.md)
- [部署与本地运行](docs/03-deployment.md)
- [认证与访问](docs/04-auth-and-access.md)
- [包与发布模型](docs/05-package-and-release-model.md)
- [制品解析](docs/06-artifact-resolution.md)
- [发布与导入工作流](docs/07-publish-and-import-workflows.md)
- [安装解析与刷新](docs/08-install-resolution-and-refresh.md)
- [治理与可见性](docs/09-governance-and-visibility.md)
- [外部来源、迁移与搜索](docs/10-external-sources-and-search.md)
- [外部来源的 CLI 与 API 扩展方案](docs/11-cli-and-api-extension-plan.md)
- [CLI 命令与参数规范](docs/12-cli-command-spec.md)
- [外部来源的 API 端点规范](docs/13-api-endpoint-spec.md)
- [CLI 参考](apps/skills-cli/README.md)

## 配置速览

### 服务端

| 变量 | 用途 |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | 服务监听地址 |
| `SKILL_REGISTRY_DATABASE_URL` | 数据库地址 |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Server 使用的制品仓库地址 |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | 客户端可访问的制品仓库地址 |
| `SKILL_REGISTRY_READER_TOKEN` | 只读 token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | 创作 token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | 发布 token |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | 访问私有仓库时的 Authorization header |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | Server 侧发布时使用的 Authorization header |

### CLI

| 变量 | 用途 |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog 基地址 |
| `SKILLS_REGISTRY_TOKEN` | CLI 请求使用的 Bearer token |
| `SKILLS_REGISTRY_ROLE` | 可选角色头，用于基于角色的本地认证 |

## 许可证

MIT
