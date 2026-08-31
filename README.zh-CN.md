# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> 面向可复用 Agent Skill 的开源注册、分发与治理平台。

Skill Registry 帮助团队发布、发现、安装和治理可复用的 Agent Skill。它将 Catalog API、Web 界面、命令行客户端以及兼容 npm 的制品仓库组合成一套可自托管的平台。

**Agent Skill** 是围绕 `SKILL.md` 及其配套资源组织的可复用能力。Skill Registry 为每个已发布的 Skill 提供稳定的 Package Name、版本化发布、可搜索元数据和受控安装路径。

## 为什么需要 Skill Registry？

Agent Skill 以目录形式分享很容易，但要形成可靠的生态还需要额外的运行治理能力。Skill Registry 提供这层基础设施：

- **可发现**：通过 Catalog 或 Web UI 搜索和浏览 Skill。
- **可复现安装**：解析精确的 Package Name 和版本，再通过 `skills` CLI 安装。
- **创作与发布**：校验、打包并发布本地 Skill，或导入外部 Skill。
- **生命周期治理**：弃用、撤回或退休 Skill 及其版本，而不静默删除历史。
- **可自托管**：使用部署自己的数据库、鉴权和兼容 npm 的制品仓库。
- **兼容多种 Agent**：将 Skill 安装到受支持编码 Agent 使用的目录中。

## 项目状态

Skill Registry 正在积极开发中。核心工作流已经实现，但部署契约、API 和用户侧功能仍可能持续演进。当前 API 和包格式应视为带版本管理的项目接口；对于预发布版本，不承诺所有行为都向后兼容。

## 架构

项目有意将产品事实与制品存储分离：

| 组件 | 职责 |
| --- | --- |
| **Catalog API** | 负责发现、元数据、版本可见性、安装解析和治理状态。 |
| **Skills CLI** | 负责安装、搜索、列出、更新、同步、校验、导入和发布 Skill。 |
| **Web UI** | 提供 Catalog 浏览和创作/导入工作流。 |
| **制品仓库** | 在 Nexus、Artifact Keeper 等兼容 npm 的仓库中存储不可变 Skill Bundle。 |
| **数据库** | 持久化 Catalog 状态、发布记录、治理变更、审计数据和工作流状态。 |

Catalog 是用户能够发现和安装哪些 Skill 的事实来源。兼容 npm 的仓库只负责存储 Bundle；只有在 Catalog 中成功创建对应的 release record 后，制品才被视为已发布版本。

## 设计原则

- **单一公共身份**：Skill 使用兼容 npm 的 **Package Name** 作为唯一标识。
- **不可变发布**：每个已发布版本都是可单独解析和校验的版本。
- **显式治理**：生命周期变更会记录操作和原因。
- **职责分离**：发现和策略由 Catalog 负责，Bundle 存储由制品仓库负责。
- **可运输的源输入**：支持本地创作，也支持从本地归档或 Git 源导入。
- **可观测的运行边界**：鉴权、就绪检查、幂等发布和审计记录属于平台能力。

## 快速开始

### 前置条件

启动本地开发栈需要安装：

- Docker
- [kind](https://kind.sigs.k8s.io/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Go 1.26 或更高版本
- Node.js 18 或更高版本及 npm
- [uv](https://docs.astral.sh/uv/)

### 启动本地环境

本仓库支持的本地开发栈会一起拉起：

```bash
make dev
make down
```

默认会把 Catalog 指向 `http://host.docker.internal:30080/`；CLI / 浏览器安装统一通过 Catalog proxy `http://127.0.0.1:18080/repository/skills-npm/`。如果你的 Artifact Keeper 不在默认内部地址，先用环境变量覆盖：

```bash
ARTIFACTKEEPER_INTERNAL_NPM_URL=http://host.docker.internal:30080/ \
ARTIFACTKEEPER_PUBLIC_NPM_URL=http://127.0.0.1:18080/repository/skills-npm/ \
make dev
```

默认本地入口：

| 服务 | 地址 |
| --- | --- |
| Web UI | <http://127.0.0.1:8080> |
| Catalog API | <http://127.0.0.1:18080> |
| Catalog npm proxy | <http://127.0.0.1:18080/repository/skills-npm/> |

### 使用 CLI 连接本地环境

Catalog 入口和 token 需要单独设置：

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=reader-secret

skills find pdf
skills add @company.skill/your-skill -y
```

发布需要具备发布权限的 Catalog token：

```bash
export SKILLS_REGISTRY_TOKEN=publisher-secret
skills publish /absolute/path/to/your-skill
```

CLI 同时支持导入本地和 Git 源、更新 registry 安装的 Skill，以及同步源安装的 Skill。完整命令请运行 `skills --help`。

## 常用开发命令

```bash
make help
make test-go
make test-web
make test-cli
make typecheck-cli
make build-web
make dev
make down
```

根目录 `Makefile` 负责顶层环境工作流和校验入口；交互式 Skill 操作仍由独立的 `skills` CLI 提供。

## 文档

- [当前使用指南](docs/README.md)
- [skills CLI 参考](apps/skills-cli/README.md)

根目录 README 是默认的中文入口。

## 配置

一个部署通常需要配置以下环境变量：

| 变量 | 用途 |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | Catalog 监听地址。 |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite 或 PostgreSQL 数据库地址。 |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | Catalog 使用的制品仓库地址，用于读取元数据和 Bundle。 |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | CLI 和浏览器客户端可访问的制品仓库地址。 |
| `SKILL_REGISTRY_READER_TOKEN` | Catalog 只读操作 token。 |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | 创作和打包工作流 token。 |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | 发布和治理操作 token。 |
| `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` | 访问私有制品仓库时使用的 Authorization header。 |
| `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` | 服务端发布时使用的 Authorization header。 |

完整示例、私有仓库鉴权和部署相关设置请参阅对应操作手册。

## 参与贡献

欢迎贡献代码、文档和 Skill。提交修改前请：

1. 阅读相关架构决策记录和包契约。
2. 保持 Catalog、CLI、Web UI 与制品仓库之间的边界清晰。
3. 为行为变更新增或更新测试。
4. 运行 `make verify`，并在 Pull Request 中说明环境限制。
5. 让文档和示例与实际命令或 API 保持同步。

如果修改公共包模型、安装解析、发布流程或治理语义，请在 `docs/adr/` 或对应设计文档中记录决策。

## 许可证

本项目采用 [MIT License](LICENSE) 许可。
