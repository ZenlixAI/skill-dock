# K3s All-in-One Runbook

这份静态 YAML 清单对应一套“全部进 k3s”的交付形态：

- `postgres`
- `nexus`
- `catalog`
- `skills-web`

`skills` CLI 仍然部署在集群外机器上，通过 Ingress 暴露的 `skills-web/api` 与 `nexus` 工作。

主清单：

- [zarf/k8s/k3s-all-in-one.yaml](../zarf/k8s/k3s-all-in-one.yaml)
- Helm chart：
  - [zarf/charts/skill-registry](../zarf/charts/skill-registry)
  - 当前默认更贴近客户现场：`外部 Postgres + 内置 Nexus + catalog + skills-web`
  - 如果你仍要用 Helm 跑 full all-in-one，再显式设置 `postgres.enabled=true`

## 1. 部署前要改的值

至少替换这些字段：

### 1.1 镜像

- `registry.example.com/skill-registry-catalog:1.0.0`
- `registry.example.com/skill-registry-web:1.0.0`
- 如果你们现场统一走镜像前缀，也可以直接设置：
  - `global.registry`

### 1.2 Secret

- `postgres-secrets.postgres_password`
- `nexus-secrets.nexus_admin_password`
- `catalog-secrets.skill_registry_reader_token`
- `catalog-secrets.skill_registry_authoring_token`
- `catalog-secrets.skill_registry_publisher_token`

### 1.3 域名

- `skills.example.com`
- `nexus.example.com`

### 1.4 Nexus 地址

- `skill_registry_npm_registry_url`
  - 这是 Catalog Pod 内访问 Nexus 的地址
  - 默认值应保持集群内 service 地址
- `skill_registry_public_npm_registry_url`
  - 这是返回给外部 CLI 的地址
  - 必须是集群外机器可访问的 Nexus 地址

## 2. 部署命令

### 2.1 直接 apply 静态 YAML

```bash
kubectl apply -f zarf/k8s/k3s-all-in-one.yaml
```

### 2.2 用 Helm 安装

当前 chart 默认 **不创建 Postgres**，而是直接复用集群里已有的 PG。

如果你的集群和当前客户现场一样，至少覆盖：

```yaml
database:
  host: postgresql.postgresql.svc.cluster.local
  port: 5432

postgres:
  enabled: false
  auth:
    database: skill_registry
    user: postgres
    password: password

nexus:
  persistence:
    storageClass: local-path
```

推荐先复制一份 values 再改：

```bash
cp zarf/charts/skill-registry/values.yaml /tmp/skill-registry-values.yaml
```

安装示例：

```bash
helm upgrade --install skill-registry ./zarf/charts/skill-registry \
  -n skill-registry \
  --create-namespace \
  -f /tmp/skill-registry-values.yaml
```

如果你不想给 Nexus 单独开 Ingress，而是已经有现成外部地址，至少覆盖：

```yaml
nexus:
  publicRepositoryUrl: https://nexus.example.com/repository/skills-npm/

ingress:
  nexus:
    enabled: false
```

如果你的 k3s 现场不是走标准 `Ingress`，而是走 Traefik `IngressRoute + entrypoint`，可以改成：

```yaml
ingress:
  kind: ingressRoute
  web:
    scheme: http
  nexus:
    scheme: http

traefik:
  web:
    entrypoint: web
  nexus:
    entrypoint: amber

nexus:
  publicRepositoryUrl: http://<cluster-ip>:31081/repository/skills-npm/
```

补充：

- `traefik.web.match` / `traefik.nexus.match` 现在默认会自动按 `Host(...) && PathPrefix(...)` 生成
- 只有你想自定义更复杂规则时，才需要手动覆盖这两个 `match`
- 如果你的入口不是 TLS，记得把 `ingress.web.scheme` / `ingress.nexus.scheme` 改成 `http`

说明：

- 这版仓库已经把 Helm chart 收到 `zarf/charts/`，和 Dockerfile、K8s 清单、脚本放在一起管理
- `catalog` / `web` / `nexus` / `postgres` 现在都支持：
  - `podAnnotations`
  - `podLabels`
  - `nodeSelector`
  - `tolerations`
  - `affinity`
- `web.service` / `nexus.service` / `catalog.service` 现在支持 `type` 和 `nodePort`
- 这里的 `Nginx` 目前仍然需要保留在 `skills-web` 容器内
  - Traefik 负责集群入口
  - `Nginx` 负责 Pod 内静态资源和同域 `/api` 反代

如果你确实要把 Postgres 也一起打进 Helm，再补：

```yaml
postgres:
  enabled: true
  auth:
    database: skill_registry
    user: skill_registry
    password: change-me
```

检查资源：

```bash
kubectl -n skill-registry get pods
kubectl -n skill-registry get svc
kubectl -n skill-registry get ingress
kubectl -n skill-registry get ingressroute
```

建议等待这几个组件 Ready：

- `nexus-0`
- `catalog-*`
- `skills-web-*`
- 如果 `postgres.enabled=true`，再额外等 `postgres-0`

## 3. Nexus 自动初始化说明

这套清单里，`nexus` Pod 内多带了一个 `nexus-bootstrap` sidecar。

它会在 Nexus Ready 后自动做这些事：

1. 用 `admin.password` 首次改管理员密码
2. 启用 `NpmToken` realm
3. 接受社区版 EULA
4. 按 Secret 中的名字创建 npm hosted 仓
5. 配置 anonymous read 开关

默认仓名：

- `skills-npm`

注意：

- 这套自动化只负责 **Nexus 服务端初始化**
- 不负责给集群外机器生成 `.npmrc`
- 发布机 / 运维机如果要 `npm publish`，仍要自己执行一次 `npm login` 或提供 `.npmrc`

## 4. CLI 如何接入

### 4.1 查询 / 安装

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com/api
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills add @company.skill/pdf-skill -y
```

### 4.2 发布 / 导入

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com/api
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/path/to/.npmrc

skills publish /abs/path/to/skill
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --npm-registry https://nexus.example.com/repository/skills-npm/
```

## 5. 验收顺序

建议按这个顺序验收：

1. 浏览器打开 `https://skills.example.com`
2. 用浏览器账号登录一次，并确认切换账号 / 退出登录生效
3. 浏览器打开 `https://nexus.example.com`
4. CLI 执行 `skills find`
5. CLI 执行 `skills publish`
6. CLI 执行 `skills add`
7. CLI 执行 `skills list`

## 6. 当前版本的现实边界

这份清单已经够你做“一体化 k3s 交付”的第一版，但还有几个明确边界：

- `nexus-bootstrap` sidecar 运行时会在容器内安装 `curl` / `jq`
- `nexus` 目前是单副本
- 如果启用内置 Postgres，`postgres` 也是单副本
- Helm chart 现在已经提供，但还没有拆成独立子 chart
- 还没有把 Nexus 运维流程完全收敛成独立 image

如果你下一步要把它再推进成正式客户交付，我建议继续做两件事：

1. 把 `nexus-bootstrap` 收口成一个独立 bootstrap image
2. 把这套 YAML 再拆成 `base + secrets + ingress` 三层
