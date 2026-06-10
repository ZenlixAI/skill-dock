# 客户现场交付 Runbook：Helm + K3s + Skills CLI

## 1. 当前推荐部署路径

客户现场现在只推荐一条主路径：

1. PostgreSQL 复用现场已有实例，或由客户单独提供托管实例。
2. Skills Registry 用 Helm 部署：
   - `skills-web`
   - `catalog-read`
   - `catalog-write`
   - `nexus`
   - 可选内置 `postgresql`
3. `skills` CLI 部署在集群外的运维机、发布机或客户开发机。

当前 Helm chart 仓库位于：

- `/Users/ultraman/workspace/skill-registry-helm-chart`

静态 YAML 现在只保留给历史兼容或紧急排障，不再作为默认交付入口。

## 2. 交付前输入清单

开始前先确认这些值：

- `skills` 对外域名，例如 `skills.example.com`
- PostgreSQL 连接信息：
  - `host`
  - `port`
  - `database`
  - `username`
  - `password`
  - `sslMode`
- Catalog 鉴权：
  - `reader token`
  - `authoring token`
  - `publisher token`
- 浏览器登录：
  - `session secret`
  - `session users json`
- Nexus 发布凭据：
  - `catalogCommon.auth.npmPublishAuthorization`
- 镜像地址：
  - `catalogCommon.image.repository`
  - `catalogCommon.image.tag`
  - `web.image.repository`
  - `web.image.tag`
  - `nexus.image.repository`
  - `nexus.image.tag`

## 3. Helm 部署边界

这套 chart 现在的部署形态是：

- `skills-web`
  - 浏览器 UI
  - 反代 `/api`
- `catalog-read`
  - 只承担读接口
  - 支持多副本、HPA、PDB
- `catalog-write`
  - 承担 `prepare`、`validate`、`pack`、`publish`、`cleanup` 等写路径
  - 默认单副本
  - 需要共享打包工作目录时，可挂 `RWX PVC`
- `nexus`
  - npm 仓
- `postgresql`
  - 默认关闭
  - 仅在你明确要内置数据库时启用

## 4. 关键 values 结构

核心 values 现在分三块：

- `catalogCommon`
  - 共享镜像、鉴权、session、feature flags
- `catalogRead`
  - 读副本数、缓存、并发、连接池、HPA、PDB
- `catalogWrite`
  - 写副本数、并发、连接池、RWX workspace、PDB

至少要关注这些字段：

```yaml
global:
  imageRegistry: ""

postgresql:
  enabled: false
  host: postgresql.postgresql.svc.cluster.local
  port: 5432
  sslMode: disable
  auth:
    database: skill_registry
    username: postgres
    password: replace-me

catalogCommon:
  image:
    repository: harbor.example.com/infras/skill-registry
    tag: v0.1.0
  auth:
    readerToken: replace-reader-token
    authoringToken: replace-authoring-token
    publisherToken: replace-publisher-token
    npmPublishAuthorization: Bearer replace-npm-token
  session:
    secret: replace-session-secret
    usersJson: |
      {
        "reader": {
          "password": "replace-reader-password",
          "role": "reader",
          "actor": "reader"
        },
        "operator": {
          "password": "replace-operator-password",
          "role": "operator",
          "actor": "operator"
        }
      }

catalogRead:
  replicaCount: 2

catalogWrite:
  replicaCount: 1

traefik:
  routes:
    web:
      enabled: true
      host: skills.example.com
      pathPrefix: /
      tls: true
    nexus:
      enabled: true
      host: skills.example.com
      pathPrefix: /repository/skills-npm
      tls: true

nexus:
  admin:
    password: replace-nexus-admin-password
  repository:
    publicUrl: ""
```

## 5. 推荐 values 准备方式

建议在 chart 仓里从生产模板复制一份：

```bash
cd /Users/ultraman/workspace/skill-registry-helm-chart
cp values.production.example.yaml values.customer.yaml
```

然后至少修改：

- `global.imageRegistry`
- `catalogCommon.image.*`
- `web.image.*`
- `nexus.image.*`
- `postgresql.*`
- `catalogCommon.auth.*`
- `catalogCommon.session.*`
- `catalogRead.replicaCount`
- `catalogWrite.replicaCount`
- `traefik.routes.web.*`
- `traefik.routes.nexus.*`
- `nexus.admin.password`

本地已有一个现场样例可参考：

- `/Users/ultraman/workspace/skill-registry-helm-chart/values.192.168.10.124.yaml`

## 6. 外部地址与内部地址

必须区分两类 Nexus 地址：

- `nexus.repository.internalUrl`
  - 给 Catalog Pod 自己访问
  - 不填时默认走集群内 Service
- `nexus.repository.publicUrl`
  - 给 CLI 返回可下载的公网 npm 地址
  - 如果 Traefik 已经把 `/repository/skills-npm` 暴露在 `skills.example.com`，这里可以留空，让 chart 自动推导

推荐公网统一入口：

- Web：`https://skills.example.com`
- npm：`https://skills.example.com/repository/skills-npm/`

如果这两个地址配反，典型表现是：

- `skills find` 正常
- `skills add` 或 `skills publish` 拉包失败

## 7. Helm 安装命令

先做渲染检查：

```bash
cd /Users/ultraman/workspace/skill-registry-helm-chart
helm template skill-registry . -f values.customer.yaml
```

再执行安装或升级：

```bash
helm upgrade --install skill-registry /Users/ultraman/workspace/skill-registry-helm-chart \
  -n skill-registry \
  --create-namespace \
  -f /Users/ultraman/workspace/skill-registry-helm-chart/values.customer.yaml
```

如果你要启用内置 PostgreSQL，再额外设置：

```yaml
postgresql:
  enabled: true
  auth:
    database: skill_registry
    username: skill_registry
    password: replace-me
```

## 8. 高可用建议

读多写少的现场，建议起步配置：

- `catalogRead.replicaCount: 2`
- `catalogWrite.replicaCount: 1`
- `catalogRead.hpa.enabled: true`
- `catalogRead.pdb.enabled: true`
- `catalogWrite.pdb.enabled: true`

如果写路径需要多副本，再先满足这两个前提：

1. `catalogWrite.workspace.enabled=true`
2. 提供可多 Pod 共享的 `RWX PVC`

否则不要直接把 `catalogWrite` 扩成多副本。

## 9. Helm 部署后检查

先看工作负载：

```bash
kubectl -n skill-registry get pods
kubectl -n skill-registry get svc
kubectl -n skill-registry get ingressroute
kubectl -n skill-registry get pdb
kubectl -n skill-registry get hpa
```

重点确认：

- `catalog-read-*` Ready
- `catalog-write-*` Ready
- `skills-web-*` Ready
- `nexus-*` Ready

如果启用了内置 PG，再确认：

- `postgresql-*` Ready

## 10. CLI 下发与配置

`skills` CLI 不应该部署进 k3s。它应该装在：

- 发布机
- 运维机
- 客户开发机

安装方式：

### 10.1 统一公网 npm 仓安装

```bash
npm install -g skills --registry https://skills.example.com/repository/skills-npm/
```

### 10.2 离线 `.tgz` 安装

```bash
npm install -g ./skills-<version>.tgz
```

查询和安装链路：

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills add @company.skill/pdf-skill -y
```

发布链路还需要 `.npmrc`：

```ini
registry=https://skills.example.com/repository/skills-npm/
//skills.example.com/repository/skills-npm/:_authToken=REPLACE_ME
```

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/abs/path/to/.npmrc

skills publish /abs/path/to/skill
```

## 11. 回归验证命令

部署完成后，至少跑这几组检查。

### 11.1 读链路

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills find pdf --json
skills list
```

### 11.2 安装链路

```bash
skills add @company.skill/pdf-skill -a codex -y
```

### 11.3 发布链路

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/abs/path/to/.npmrc

skills publish /abs/path/to/skill
```

如果是历史 skill：

```bash
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --registry https://skills.example.com \
  --npm-registry https://skills.example.com/repository/skills-npm/ \
  --publish-registry https://skills.example.com/repository/skills-npm/
```

## 12. 故障排查

`skills find` 失败时，先查：

- `SKILLS_REGISTRY_URL`
- `SKILLS_REGISTRY_TOKEN`
- CLI 到 `skills.example.com` 的网络

`skills add` 或 `skills publish` 拉包失败时，优先查：

- `nexus.repository.publicUrl`
- Traefik `/repository/skills-npm` 路由
- CLI 机器到公网 npm 入口的可达性

浏览器能打开但列表为空时，优先查：

- `skills-web` 到 `catalog-read` 的反代
- session 配置
- 只读固定 token 模式下的 `web.auth.catalogAuthorizationHeader`

## 13. 兼容说明

如果必须走旧的静态 YAML，再参考：

- [zarf/k8s/customer-k3s-example.yaml](../zarf/k8s/customer-k3s-example.yaml)

但这条路径现在不再作为主线交付方式，也不会优先跟进新的读写拆分和 HA 能力。
