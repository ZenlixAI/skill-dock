# 当前 Skill Registry 使用指南

本文介绍在这个仓库里如何使用当前的 Skill Registry：如何启动本地环境、如何连接 CLI、如何安装和发布 Skill，以及什么时候该用 `update`、`sync`、`publish`、`import-local` 和 `import-git`。

## 这个仓库里的“当前 Registry”是什么

当前的本地开发栈会把下面几部分一起拉起来：

- Web UI
- Catalog API
- PostgreSQL
- 独立运行的 Artifact Keeper（npm 兼容制品仓库）

在这个部署模型里：

- `SKILLS_REGISTRY_URL` 指向 Catalog / Web 的公共入口
- npm bundle 的上传和下载走单独的 npm 兼容仓库地址
- `skills publish` 的实际流程是 `local pack -> npm publish -> Catalog release-record`
- `skills import-local` 和 `skills import-git` 走 Catalog 的一次性发布流程，不是本地安装刷新

## 1. 启动本地环境

```bash
make dev
make down
```

默认本地地址如下：

| 服务 | 地址 |
| --- | --- |
| Web UI | <http://127.0.0.1:8080> |
| Catalog API | <http://127.0.0.1:18080> |
| npm proxy（通过 Catalog） | <http://127.0.0.1:18080/repository/skills-npm/> |

如果你的 Artifact Keeper 不在默认内部地址，可以覆盖下面两个变量再启动：

```bash
ARTIFACTKEEPER_INTERNAL_NPM_URL=http://host.docker.internal:30080/ \
ARTIFACTKEEPER_PUBLIC_NPM_URL=http://127.0.0.1:18080/repository/skills-npm/ \
make dev
```

## 2. 配置 CLI 连接

最常用的是把 Catalog 入口和 token 单独设置好：

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=reader-secret
```

- `reader-secret` 可以做搜索、查看和安装
- `publisher-secret` 可以做发布和治理操作
- npm 认证仍然依赖本地 `.npmrc`、`NPM_CONFIG_USERCONFIG` 或用户目录里的 npm 配置

如果你在私有 npm 仓库上安装或发布，先确认 CLI 能读到正确的 npm auth，再执行 `skills add` 或 `skills publish`。

## 3. 常见操作

### 搜索和安装

```bash
skills find pdf
skills add @company.skill/your-skill -y
skills add @company.skill/pdf-skill --version 1.2.0
```

### 发布本地 Skill

```bash
export SKILLS_REGISTRY_TOKEN=publisher-secret
skills publish /absolute/path/to/your-skill
```

### 导入外部来源

```bash
skills import-local ./legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0
skills import-git owner/repo --skill pdf
```

### 更新或同步已安装 Skill

```bash
skills update @company.skill/pdf-skill
skills sync legacy-skill
```

简单记忆：

- `update`：只管 registry 已安装的 Skill
- `sync`：只管本地路径或 Git 来源安装的 Skill
- `remove`：只做本地移除，不改 Catalog
- `publish`：本地打包后发布到 Catalog
- `import-local` / `import-git`：把外部来源通过 Catalog 一次性发布进去

## 4. 排查常见问题

- `skills find` / `skills add` 返回 401：检查 `SKILLS_REGISTRY_TOKEN`
- 下载 tarball 失败：检查 `.npmrc` 是否指向了正确的 npm 认证
- `skills publish` 失败：确认使用的是 `publisher-secret` 或等价权限
- 安装后看不到 Skill：确认 `SKILLS_REGISTRY_URL` 指向的是 Catalog 入口，而不是 npm 仓库本身

## 5. 相关文档

- 根目录 README: [`../README.md`](../README.md)
