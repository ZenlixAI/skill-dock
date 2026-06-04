# K3s All-in-One Runbook

这份清单对应一套“全部进 k3s”的交付形态：

- `postgres`
- `nexus`
- `catalog`
- `skills-web`

`skills` CLI 仍然部署在集群外机器上，通过 Ingress 暴露的 `skills-web/api` 与 `nexus` 工作。

主清单：

- [zarf/k8s/k3s-all-in-one.yaml](/Users/ultraman/workspace/skill_registry/zarf/k8s/k3s-all-in-one.yaml)

## 1. 部署前要改的值

至少替换这些字段：

### 1.1 镜像

- `registry.example.com/skill-registry-catalog:1.0.0`
- `registry.example.com/skill-registry-web:1.0.0`

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

```bash
kubectl apply -f zarf/k8s/k3s-all-in-one.yaml
```

检查资源：

```bash
kubectl -n skill-registry get pods
kubectl -n skill-registry get svc
kubectl -n skill-registry get ingress
```

建议等待这几个组件 Ready：

- `postgres-0`
- `nexus-0`
- `catalog-*`
- `skills-web-*`

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
2. 浏览器检查 `https://skills.example.com/api/v1/session`
3. 浏览器打开 `https://nexus.example.com`
4. CLI 执行 `skills find`
5. CLI 执行 `skills publish`
6. CLI 执行 `skills add`
7. CLI 执行 `skills list`

## 6. 当前版本的现实边界

这份清单已经够你做“一体化 k3s 交付”的第一版，但还有几个明确边界：

- `nexus-bootstrap` sidecar 运行时会在容器内安装 `curl` / `jq`
- `postgres` / `nexus` 目前都是单副本
- 还没有 Helm chart
- 还没有把 Nexus 运维流程完全收敛成独立 image

如果你下一步要把它再推进成正式客户交付，我建议继续做两件事：

1. 把 `nexus-bootstrap` 收口成一个独立 bootstrap image
2. 把这套 YAML 再拆成 `base + secrets + ingress` 三层
