# 客户现场交付 Runbook：Helm + K3s + Skills CLI

## 1. 当前推荐部署路径

客户现场现在推荐的主路径只有一条：

1. PostgreSQL 使用现场已有实例，或由客户单独提供托管实例。
2. `skill-registry` 通过 Helm chart 部署以下组件：
   - `web`
   - `catalog-read`
   - `catalog-write`
   - `nexus`
3. `skills` CLI 运行在集群外的发布机、运维机或客户开发机。

当前 Helm chart 仓库位于：

- `/Users/ultraman/workspace/skill-registry-helm-chart`

这份 runbook 以该 chart 当前实现为准，不再沿用旧版静态 YAML 的组件拆分和 values 结构。

## 2. 当前 chart 的真实边界

现在的 chart 和旧文档相比，有几条边界必须先讲清楚：

- PostgreSQL 始终是外部实例，chart 不负责部署内置 PostgreSQL
- npm 仓当前只支持内置 Nexus，不再支持 `nexus.mode=external`
- 对外主入口只保留 `web`
- `web` 统一代理浏览器页面、Catalog API 和 npm 仓库路径
- `nexus` 管理台可以单独通过 `nexus.traefik.*` 暴露，但这不是 CLI/npm 的主入口

`web` 当前代理的关键路径：

- `/` -> `web`
- `/api/` -> `catalog-read`
- `/api/v1/skill-packaging/` -> `catalog-write`
- `/api/v1/skills/.*/(release-record|governance-changes)` -> `catalog-write`
- `/api/v1/audit-events` -> `catalog-write`
- `/repository/skills-npm/` -> `nexus`

这意味着：

- 外部 CLI 和 CI 使用的 npm registry 地址应指向 `web` 的对外地址
- `catalog-write` 在集群内直接访问 Nexus Service
- 不再需要单独给 npm 仓库准备第二套公网入口

## 3. 交付前输入清单

开始前先确认这些值：

- 外部访问地址
  - `traefik.entrypoint`
  - `traefik.host`
  - `nexus.repository.publicUrl`
- PostgreSQL 连接信息
  - `postgresql.host`
  - `postgresql.port`
  - `postgresql.database`
  - `postgresql.username`
  - `postgresql.password`
  - `postgresql.sslMode`
- Catalog 鉴权
  - `catalogCommon.auth.readerToken`
  - `catalogCommon.auth.authoringToken`
  - `catalogCommon.auth.publisherToken`
  - 或 `catalogCommon.auth.authTokensJson`
- Web 鉴权模式
  - 浏览器 session 登录
    - `catalogCommon.session.secret`
    - `catalogCommon.session.usersJson`
  - 或固定 Header 透传
    - `web.auth.catalogAuthorizationHeader`
- Catalog 服务端发布 npm 包时使用的凭据
  - `catalogCommon.auth.npmPublishAuthorization`
- 镜像地址
  - `global.imageRegistry` 或 `global.registry`
  - `catalogCommon.image.repository`
  - `catalogCommon.image.tag`
  - `web.image.repository`
  - `web.image.tag`
  - `nexus.image.repository`
  - `nexus.image.tag`
- 持久化与资源
  - `nexus.persistence.*`
  - `catalogWrite.workspace.*`，仅在需要持久化工作目录时
  - `catalogRead.hpa.enabled=true` 时，集群应已安装 `metrics-server`

## 4. 当前 values 结构

当前 chart 的核心结构如下：

```yaml
global:
  registry: ""
  imageRegistry: ""
  imagePullSecrets: []

traefik:
  enabled: true
  entrypoint: web
  host: "skills.local"

postgresql:
  host: postgres.example.internal
  port: 5432
  database: skill_registry
  username: postgres
  password: replace-with-database-password
  sslMode: require

catalogCommon:
  image:
    repository: infras/skill-registry
    tag: v0.1.0
  auth:
    readerToken: replace-reader-token
    authoringToken: replace-authoring-token
    publisherToken: replace-publisher-token
    npmPublishAuthorization: Bearer replace-npm-token
    authTokensJson: ""
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
  replicaCount: 3
  hpa:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
  pdb:
    enabled: true
    minAvailable: 2

catalogWrite:
  replicaCount: 1
  workspace:
    enabled: false
    existingClaim: ""
    size: 10Gi
    storageClass: ""
    accessModes:
      - ReadWriteOnce

web:
  service:
    type: ClusterIP
    port: 8080
    nodePort: 0
  auth:
    catalogAuthorizationHeader: ""

nexus:
  admin:
    username: admin
    password: replace-nexus-admin-password
  repository:
    name: skills-npm
    publicUrl: https://skills.example.com/repository/skills-npm/
    anonymousRead: true
  traefik:
    enabled: true
    entrypoint: ""
    host: nexus.skills.local
  persistence:
    size: 50Gi
    storageClass: ""
    accessModes:
      - ReadWriteOnce
```

需要特别注意，下面这些旧字段已经不适用于当前 chart：

- `postgresql.enabled`
- `postgresql.auth.*`
- `traefik.routes.*`
- `nexus.mode`
- `nexus.repository.internalUrl`
- `nexus.externalService.*`

## 5. 当前 chart 的校验规则

Helm 模板当前会校验这些条件：

- `catalogCommon.image.repository` 必填
- `web.image.repository` 必填
- `nexus.image.repository` 必填
- `postgresql.host` 必填
- `postgresql.password` 必填
- `nexus.admin.password` 必填
- `nexus.repository.publicUrl` 必填

条件必填项：

- `catalogCommon.auth.authTokensJson` 为空时，`readerToken` / `authoringToken` / `publisherToken` 都必须填
- `web.auth.catalogAuthorizationHeader` 为空时，`catalogCommon.session.secret` 和 `catalogCommon.session.usersJson` 都必须填

Service 规则：

- `service.type=ClusterIP` 时，`service.nodePort` 必须是 `0`
- 只有 `service.type=NodePort` 时，才允许填写 `service.nodePort`

Traefik 规则：

- `nexus.traefik.enabled=true` 时，要求 `traefik.enabled=true`
- `nexus.traefik.enabled=true` 时，要求 `nexus.traefik.host` 非空
- 如果 `nexus` 管理台路由和 `web` 路由共用同一个 entrypoint，`nexus.traefik.host` 必须和 `traefik.host` 不同

## 6. 推荐 values 准备方式

建议直接从 chart 默认值复制一份：

```bash
cd /Users/ultraman/workspace/skill-registry-helm-chart
cp values.yaml values.customer.yaml
```

然后至少修改这些字段：

- `global.imageRegistry` 或 `global.registry`
- `catalogCommon.image.*`
- `web.image.*`
- `nexus.image.*`
- `postgresql.*`
- `catalogCommon.auth.*`
- `catalogCommon.session.*`
- `web.auth.catalogAuthorizationHeader`
- `traefik.*`
- `nexus.admin.password`
- `nexus.repository.publicUrl`
- `nexus.persistence.*`

按需再改：

- `catalogRead.replicaCount`
- `catalogRead.hpa.*`
- `catalogRead.pdb.*`
- `catalogWrite.replicaCount`
- `catalogWrite.workspace.*`
- `nexus.traefik.*`

本地已有一个现场样例可参考：

- `/Users/ultraman/workspace/skill-registry-helm-chart/values.192.168.10.124.yaml`

## 7. 地址和鉴权的几个关键点

### 7.1 `nexus.repository.publicUrl`

这个值是给外部 CLI/CI 使用的 npm 下载与发布地址，通常应该写成：

```text
https://skills.example.com/repository/skills-npm/
```

或者在没有域名时写成统一入口的 IP:Port 形式：

```text
https://192.168.10.102:31482/repository/skills-npm/
```

如果这里填错，常见现象是：

- `skills find` 正常
- `skills add` 或 `skills publish` 拉包失败

### 7.2 `catalogCommon.auth.npmPublishAuthorization`

这个值是给 `catalog-write` 服务端自己用来发 `npm publish` 的，不是浏览器登录凭据，也不是开发机本地 CLI 的 `.npmrc`。

常见形式：

```yaml
catalogCommon:
  auth:
    npmPublishAuthorization: Bearer NpmToken.xxxxx
```

或：

```yaml
catalogCommon:
  auth:
    npmPublishAuthorization: Basic base64xxxx
```

如果你的 `.npmrc` 里是：

```ini
//skills.example.com/repository/skills-npm/:_authToken=NpmToken.xxxxx
```

那 values 里应填：

```text
Bearer NpmToken.xxxxx
```

如果你的 `.npmrc` 里是：

```ini
//skills.example.com/repository/skills-npm/:_auth=base64xxxx
```

那 values 里应填：

```text
Basic base64xxxx
```

### 7.3 Web 登录模式

两种模式二选一：

- `web.auth.catalogAuthorizationHeader=""`
  - 浏览器走 session 登录
  - 需要 `catalogCommon.session.secret` 和 `catalogCommon.session.usersJson`
- `web.auth.catalogAuthorizationHeader="Bearer ..."`
  - Web 固定透传 Header
  - 浏览器登录会被关闭

## 8. Helm 安装命令

先做渲染检查：

```bash
cd /Users/ultraman/workspace/skill-registry-helm-chart
helm template smoke . -f values.customer.yaml
```

再执行安装或升级：

```bash
helm upgrade --install skill-registry /Users/ultraman/workspace/skill-registry-helm-chart \
  -n skill-registry \
  --create-namespace \
  -f /Users/ultraman/workspace/skill-registry-helm-chart/values.customer.yaml
```

常见模板报错包括：

- `postgresql.host must be set`
- `postgresql.password must be set`
- `nexus.admin.password must be set`
- `catalogCommon.auth.readerToken must be set when catalogCommon.auth.authTokensJson is empty`
- `catalogCommon.session.secret must be set when browser login is enabled (web.auth.catalogAuthorizationHeader is empty)`
- `nexus.repository.publicUrl must be set so Catalog can publish a reachable npm registry URL`

## 9. 高可用与存储建议

读多写少的现场，建议起步配置：

- `catalogRead.replicaCount: 2` 或 3
- `catalogRead.hpa.enabled: true`
- `catalogRead.pdb.enabled: true`
- `catalogWrite.replicaCount: 1`
- `catalogWrite.pdb.enabled: false`

说明：

- `catalog-write` 单副本时，保持 `pdb.enabled=false` 更符合当前默认行为
- 写路径如果暂时没有共享工作目录需求，不要急着开多副本

只有在写路径确实需要持久化工作目录时，才启用：

```yaml
catalogWrite:
  workspace:
    enabled: true
```

补充：

- `existingClaim` 不为空时，chart 会复用已有 PVC
- 挂载路径固定为 `/srv/packaging-workspaces`
- 如果要把 `catalog-write` 扩成多副本，需要先确认底层存储和工作目录语义都支持

## 10. Helm 部署后检查

先看工作负载和服务：

```bash
kubectl -n skill-registry get pods
kubectl -n skill-registry get svc
kubectl -n skill-registry get ingressroute
kubectl -n skill-registry get hpa
kubectl -n skill-registry get pdb
kubectl -n skill-registry get configmap | grep postgresql
helm -n skill-registry get notes skill-registry
```

通常会看到这些资源 Ready：

- `<release>-web`
- `<release>-catalog`
- `<release>-catalog-write`
- `<release>-nexus`

重点确认：

- `web` 和 `nexus` Service 默认应保持 `ClusterIP`
- `IngressRoute` 的 backend 指向 `web`
- `nexus.repository.publicUrl` 和实际外部入口一致

如果要看 Traefik 路由细节：

```bash
kubectl -n skill-registry get ingressroute entrypoint -o yaml
kubectl -n skill-registry get ingressroute nexus-entrypoint -o yaml
```

## 11. CLI 下发与配置

`skills` CLI 不应该部署进 k3s。它应安装在：

- 发布机
- 运维机
- 客户开发机

### 11.1 从统一 npm 仓安装

```bash
npm install -g skills --registry https://skills.example.com/repository/skills-npm/
```

### 11.2 离线 `.tgz` 安装

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

## 12. 回归验证命令

部署完成后，至少跑这几组检查。

### 12.1 读链路

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills find pdf --json
skills list
```

### 12.2 安装链路

```bash
skills add @company.skill/pdf-skill -a codex -y
```

### 12.3 发布链路

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

## 13. 故障排查

`skills find` 失败时，优先检查：

- `SKILLS_REGISTRY_URL`
- `SKILLS_REGISTRY_TOKEN`
- CLI 到 `skills.example.com` 的网络

`skills add` 或 `skills publish` 拉包失败时，优先检查：

- `nexus.repository.publicUrl`
- `web` 对 `/repository/skills-npm/` 的代理是否正常
- CLI 机器到公网 npm 入口的可达性

Web 可以打开但发布失败时，优先检查：

- `catalogCommon.auth.npmPublishAuthorization`
- `nexus.repository.publicUrl`
- `catalog-write` 是否能访问 `nexus`

排查命令：

```bash
kubectl -n skill-registry logs deploy/skill-registry-catalog-write
kubectl -n skill-registry logs statefulset/skill-registry-nexus
kubectl -n skill-registry logs deploy/skill-registry-web
```

浏览器登录异常时，优先检查：

- `catalogCommon.session.secret`
- `catalogCommon.session.usersJson`
- `web.auth.catalogAuthorizationHeader`

如果设置了 `web.auth.catalogAuthorizationHeader`，浏览器登录会被禁用，这是当前预期行为。
