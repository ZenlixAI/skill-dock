# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> 面向可复用 Agent Skill 的自托管控制平面，负责发现、打包、安装和治理。

Skill Registry 包含两个主要运行面：

- **Catalog server**：负责 catalog 事实、release 记录、可见性、治理、打包和安装解析；
- **`skills` CLI**：负责发现、安装、刷新、检查、导入和发布。

Bundle 存放在外部 npm 兼容制品仓库中。制品仓库可以替换；只有 server 接受了 release record，bundle 才算已发布版本。

## 当前工作流

| 工作流 | 命令 | 事实/执行来源 |
| --- | --- | --- |
| Catalog 发现 | `skills find` | Server catalog，可选 `skills.sh` 结果 |
| 安装 | `skills add` | Registry 包由 server 解析；本地/Git 来源由 CLI 解析 |
| 本地清单 | `skills list`、`skills remove` | CLI lock 文件和本地安装文件 |
| Registry 刷新 | `skills update` | Server release 解析 |
| Source 刷新 | `skills sync` | 已记录的本地或 Git 来源 |
| 本地创作 | `skills init`、`skills inspect` | 本地文件系统 |
| 发布 | `skills publish` | Server 打包和 release 流程 |
| 来源导入 | `skills import-local`、`skills import-git` | Server 打包和 release 流程 |
| npm 导入 | `skills import-npm` | CLI 下载 npm 包，再交给 server 导入 |

使用 `skills add` 安装本地或 Git 来源后，CLI 默认还会通过 server 的 local import 流程提交一次已安装来源。项目级 `skills.config.json` 可以关闭这个行为；它不会把 source install 变成 registry install。设置 `syncOnAdd` 为 `false` 可以保持纯本地安装。

## 边界

- **Server**：负责 catalog、release、安装、可见性、治理、审计和打包决策。
- **CLI**：负责命令协议、来源发现、本地物化、lock 文件，以及 npm 导入时的客户端下载。
- **制品仓库**：负责 npm 元数据和不可变 bundle。
- **数据库**：持久化 catalog、release、治理、审计、workspace 和幂等状态。

Skill 的身份是 scoped package name；release 的身份是 package name 加 exact version。Package identity 和 release version 不会被静默改写。

## 安装目标模式

对于 registry 安装，server 返回以下模式之一：

- `proxy`：通过 server 的 npm proxy 下载 tarball；
- `direct`：直接访问上游 npm 兼容仓库，必要时使用 npm 配置中的凭据。

这只是传输策略，不是第二套 skill 身份或 release 模型。

## 本地开发

```bash
make dev
make dev-images
make down
```

`make dev` 创建本地栈并初始化示例数据。kind 集群已经存在时，`make dev-images` 只重新加载构建好的服务镜像，不重新初始化数据。默认栈包含 catalog server、PostgreSQL 和外部 Artifact Keeper 制品仓库。

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
- [外部来源与发现](docs/10-external-sources-and-search.md)
- [外部来源实现状态](docs/11-cli-and-api-extension-plan.md)
- [CLI 命令参考](docs/12-cli-command-spec.md)
- [API 端点参考](docs/13-api-endpoint-spec.md)
- [CLI 包说明](apps/skills-cli/README.md)

## 配置速览

### Server

| 变量 | 用途 |
| --- | --- |
| `SKILL_REGISTRY_ADDR` | HTTP 监听地址 |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite 或 PostgreSQL 地址 |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | 可写的打包 workspace 根目录 |
| `SKILL_REGISTRY_NPM_REGISTRY_URL` | 私有 npm 兼容制品仓库 |
| `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` | 返回给客户端的制品地址 |
| `SKILL_REGISTRY_READER_TOKEN` | 读取 token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | 创作和打包 token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | 发布和治理 token |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | `proxy` 或 `direct` |
| `SKILL_REGISTRY_SKILLS_SH_URL` | 可选的 `skills.sh` 搜索地址，默认 `https://skills.sh` |

完整配置和优先级见[Server 架构与配置](docs/01-server-architecture.md)。

### CLI

| 变量 | 用途 |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server 基地址 |
| `SKILLS_REGISTRY_TOKEN` | Catalog 请求使用的 Bearer token |
| `SKILLS_REGISTRY_ROLE` | 可选角色头 |
| `NPM_CONFIG_USERCONFIG` | direct npm 请求使用的 npm 凭据文件 |

## 许可证

MIT
