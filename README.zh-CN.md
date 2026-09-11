# Skill Registry

[English](README.md) | [简体中文](README.zh-CN.md)

> 用于发现、发布、安装和治理可复用 Agent Skill 的自托管控制平面。

Skill Registry 包含两个运行面：

- **Catalog server**：负责 catalog 数据、托管 release、安装解析、打包、可见性、治理和审计记录；
- **`skills` CLI**：负责发现来源、本地安装、刷新本地状态，以及调用 catalog 工作流。

Bundle 存放在外部 npm 兼容制品仓库中。只有 catalog server 接受 release record 后，bundle 才是已发布 release。

## 工作流

| 工作流 | 命令 | 事实来源 |
| --- | --- | --- |
| Catalog 发现 | `skills find` | Catalog server；可选 `skills.sh` 结果 |
| 安装 | `skills add` | Registry 包由 server 解析；本地和 Git 来源由 CLI 解析 |
| 本地清单 | `skills list`、`skills remove` | 本地文件和 lock 文件 |
| Registry 刷新 | `skills update` | Server release 解析 |
| Source 刷新 | `skills sync` | 已记录的本地或 Git 来源 |
| 本地创作 | `skills init`、`skills inspect` | 本地文件系统 |
| 发布 | `skills publish`、`skills import-*` | Server 打包和 release 工作流 |

安装本地或 Git 来源后，CLI 默认尝试向 server 导入。该操作为尽力而为，不会将 source install 变成 registry install。在合并后的全局或项目 `skills.config.json` 中设置 `syncOnAdd: false` 可禁用该操作。

## Docker Compose

本地开发使用根目录下的 Compose 文件。它会启动 Catalog、PostgreSQL 和本地 npm 兼容制品仓库，不部署网站：

```bash
cp .env.example .env
# 编辑 .env，替换所有示例密码和 token。
docker compose up --build -d
```

Catalog API 地址是 <http://localhost:8080/api/v1>，npm proxy 地址是 <http://localhost:8080/repository/skills-npm/>。使用 `docker compose down` 停止服务；只有需要删除本地数据库和制品数据时才使用 `-v`。

生产环境请使用[生产部署说明](docs/03-deployment.md)。该方案只运行 Catalog 和 PostgreSQL；Catalog 连接外部 npm 兼容制品仓库，并在 Catalog 前配置 TLS 反向代理。

## Dockerfile

只使用 SQLite 运行 Catalog：

```bash
docker build -f zarf/docker/local/Dockerfile -t skill-registry-catalog .
docker run --rm -p 8080:8080 -v skill-registry-data:/var/lib/skill-registry \
  -e SKILL_REGISTRY_READER_TOKEN=reader-secret \
  -e SKILL_REGISTRY_AUTHORING_TOKEN=authoring-secret \
  -e SKILL_REGISTRY_PUBLISHER_TOKEN=publisher-secret \
  skill-registry-catalog
```

生产环境请使用真实 secret，并配置 PostgreSQL 和 npm 兼容制品仓库。详细配置见[部署说明](docs/03-deployment.md)和[服务端配置](docs/01-server-architecture.md)。


## 文档

- [产品方向与范围](docs/00-product-direction.md)
- [Server 架构与配置](docs/01-server-architecture.md)
- [CLI 协议与命令模型](docs/02-cli-contract.md)
- [生产部署](docs/03-deployment.md)
- [认证与访问](docs/04-auth-and-access.md)
- [包与发布模型](docs/05-package-and-release-model.md)
- [制品解析](docs/06-artifact-resolution.md)
- [发布与导入工作流](docs/07-publish-and-import-workflows.md)
- [安装解析与刷新](docs/08-install-resolution-and-refresh.md)
- [治理与可见性](docs/09-governance-and-visibility.md)
- [外部来源与发现](docs/10-external-sources-and-search.md)
- [CLI 命令参考](docs/12-cli-command-spec.md)
- [API 端点参考](docs/13-api-endpoint-spec.md)
- [CLI 包说明](apps/skills-cli/README.md)

## 配置

### Server

| 变量 | 用途 |
| --- | --- |
| `SKILL_REGISTRY_DATABASE_URL` | SQLite 或 PostgreSQL 地址 |
| `SKILL_REGISTRY_SHARED_WORKSPACE_ROOT` | 可写的打包 workspace 根目录 |
| `SKILL_REGISTRY_ARTIFACT_READ_URL` | Server 读取制品的仓库地址 |
| `SKILL_REGISTRY_PUBLIC_INSTALL_URL` | 返回给客户端的制品地址 |
| `SKILL_REGISTRY_READER_TOKEN` | 读取 token |
| `SKILL_REGISTRY_AUTHORING_TOKEN` | 创作 token |
| `SKILL_REGISTRY_PUBLISHER_TOKEN` | 发布 token |
| `SKILL_REGISTRY_INSTALL_TARGET_MODE` | `proxy` 或 `direct` |

### CLI

| 变量 | 用途 |
| --- | --- |
| `SKILLS_REGISTRY_URL` | Catalog server 基地址 |
| `SKILLS_REGISTRY_TOKEN` | Catalog 请求使用的 Bearer token |
| `SKILLS_REGISTRY_ROLE` | 开发环境角色头 |
| `NPM_CONFIG_USERCONFIG` | direct npm 流量使用的 npm 凭据 |

## 许可证

MIT
