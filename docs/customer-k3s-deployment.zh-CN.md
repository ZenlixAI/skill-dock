# 客户现场交付 Runbook：K3s + 外部 Nexus + Skills CLI

## 1. 适用范围

这份 Runbook 只覆盖下面这条交付形态：

- 集群内部署：
  - `skills-web`
  - `catalog`
- 集群外已有：
  - `Postgres`
  - `Nexus` npm 仓
- 集群外下发：
  - `skills` CLI

如果客户没有外部 Nexus，这份文档不够，需要单独补方案。

如果客户现场是“**复用集群内 Postgres，但 Nexus 跟业务一起部署**”，现在可以直接用 Helm chart：

- [zarf/charts/skill-registry](../zarf/charts/skill-registry)

它当前默认就是这条边界：

1. 外部 / 集群内已有 Postgres
2. Chart 内部署 `nexus + catalog + skills-web`
3. `skills` CLI 仍然部署在集群外机器

## 2. 最小生产边界

现场最小生产边界不是“web + catalog”，而是：

1. `skills-web`
2. `catalog`
3. `Postgres`
4. 外部 `Nexus`
5. `skills` CLI 下发到发布机 / 运维机 / 客户开发机

## 3. 交付前输入清单

开始前先拿齐这些信息：

### 3.1 外部依赖

- Postgres DSN
- Nexus npm hosted 仓地址
- Catalog Pod 内访问 Nexus 的地址
- CLI 机器访问 Nexus 的地址

### 3.2 域名与入口

- Web 域名
- Catalog 是否通过 web 同域 `/api` 暴露
- Ingress 类型
  - Traefik
  - Nginx
  - 其他

### 3.3 认证与权限

- `reader token`
- `authoring token`
- `publisher token`
- Nexus 发布凭据
  - `.npmrc`
  - token
  - 或用户名/密码

### 3.4 镜像与版本

- `catalog` 镜像 tag
- `skills-web` 镜像 tag
- `skills` CLI 版本

## 4. 交付物清单

这次交付至少要包含：

### 4.1 集群交付物

- `catalog` 镜像
- `skills-web` 镜像
- k3s 清单
  - 参考 [zarf/k8s/customer-k3s-example.yaml](../zarf/k8s/customer-k3s-example.yaml)

### 4.2 CLI 交付物

- `skills` npm 包
  - 或 `.tgz`
- Node.js 版本要求
  - `>=18`
- CLI 安装说明
- CLI 使用说明

### 4.3 运维资料

- Secret 键名清单
- 发布验证命令
- 回滚命令或回滚策略

## 5. 关键配置表

### 5.1 Catalog 运行时环境变量

必填：

- `SKILL_REGISTRY_ADDR`
- `SKILL_REGISTRY_DATABASE_URL`
- `SKILL_REGISTRY_READER_TOKEN`
- `SKILL_REGISTRY_AUTHORING_TOKEN`
- `SKILL_REGISTRY_PUBLISHER_TOKEN`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`

建议同时设置：

- `SKILL_REGISTRY_ENABLE_SAMPLE_DATA=false`
- `SKILL_REGISTRY_ENABLE_MOCK_REGISTRY=false`

如果 Web 端要支持浏览器登录、切换账号、退出登录，这两项必须补：

- `SKILL_REGISTRY_SESSION_SECRET`
- `SKILL_REGISTRY_SESSION_USERS_JSON`

### 5.1.1 Catalog 副本数约束

当前 packaging `prepare / validate / pack / bundle download` 依赖 Catalog Pod 本地临时 workspace。

这意味着：

- `catalog` 现在必须先保持 **1 个副本**
- 不要把 `catalog` 默认部署成 `2+ replicas`
- 否则一次导入里的 `prepare` 和后续 `validate / pack / bundle` 可能落到不同 Pod，最终报：
  - `invalid_request: workspacePath must reference an existing directory`

如果你需要 `catalog` 横向扩容，先别扩这一块；要等后面把 packaging workspace 改成共享存储或异步 job。

`SKILL_REGISTRY_SESSION_USERS_JSON` 最小示例：

```json
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
```

### 5.2 两个 Nexus 地址的区别

- `SKILL_REGISTRY_NPM_REGISTRY_URL`
  - 给 Catalog Pod 自己访问 Nexus metadata / tarball
  - 必须是 **Pod 内可达地址**

- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
  - 给 Catalog 返回给外部 CLI 的 `npmRegistry` / `tarballUrl`
  - 必须是 **发布机 / 运维机 / 客户开发机可达地址**

常见正确示例：

- Pod 内地址：
  - `http://nexus.repo.svc.cluster.local/repository/npm-hosted/`
- CLI 外部地址：
  - `https://nexus.customer.com/repository/npm-hosted/`

如果这两个值配反，最常见表现是：

- `skills find` 正常
- `skills add` 报 `fetch failed`

### 5.3 Web 侧建议

推荐做法：

- 浏览器访问 `https://skills.example.com`
- `skills-web` 用 Nginx 提供静态资源
- `Nginx` 反代 `/api/...` 到 `catalog`
- 登录模式下，不注入固定 `Authorization`
- 让浏览器通过 session cookie 调同域 `/api/...`

只有在旧的只读固定 token 部署里，才继续注入 `reader token`。

不要把 `publisher token` 编进前端静态资源。

## 6. 部署资产位置

### 6.1 Catalog 镜像

- Dockerfile：
  - [zarf/docker/Dockerfile](../zarf/docker/Dockerfile)

构建示例：

```bash
docker build -f zarf/docker/Dockerfile -t registry.example.com/skill-registry-catalog:1.0.0 .
```

### 6.2 Web 镜像

- Dockerfile：
  - [zarf/docker/skills-web.Dockerfile](../zarf/docker/skills-web.Dockerfile)
- Nginx 模板：
  - [zarf/docker/nginx/skills-web.conf.template](../zarf/docker/nginx/skills-web.conf.template)

构建示例：

```bash
docker build -f zarf/docker/skills-web.Dockerfile -t registry.example.com/skill-registry-web:1.0.0 .
```

### 6.3 K3s 示例清单

- [zarf/k8s/customer-k3s-example.yaml](../zarf/k8s/customer-k3s-example.yaml)

这份示例假设：

- Postgres 已存在
- Nexus 已存在
- 你只需要替换：
  - image
  - 域名
  - Secret 值
  - ingressClass / annotation

### 6.4 Helm chart

- [zarf/charts/skill-registry](../zarf/charts/skill-registry)

默认边界：

1. 复用集群内已有 Postgres
2. Chart 内部署 `nexus + catalog + skills-web`
3. 如需 full all-in-one，再显式设置 `postgres.enabled=true`

当前 chart 还额外支持这些现场常用交付字段：

- `global.registry`
- `global.imagePullSecrets`
- `catalog/web/nexus/postgres` 的：
  - `podAnnotations`
  - `podLabels`
  - `nodeSelector`
  - `tolerations`
  - `affinity`
- `catalog.service` / `web.service` / `nexus.service` 的：
  - `type`
  - `nodePort`
- `nexus.persistence.storageClass`
- `postgres.persistence.storageClass`

按你当前集群口径的最小安装示例：

```bash
helm upgrade --install skill-registry ./zarf/charts/skill-registry \
  -n skill-registry \
  --create-namespace \
  --set database.host=postgresql.postgresql.svc.cluster.local \
  --set database.port=5432 \
  --set postgres.auth.user=postgres \
  --set postgres.auth.password=password
```

## 7. Skills CLI 下发方式

`skills` CLI 不应该部署进 k3s。它的角色是：

- 客户开发机安装工具
- 发布机 / 运维机发布工具

### 7.1 方式 A：发到客户已有 npm 仓

这是默认推荐路径。

发布：

```bash
cd apps/skills-cli
corepack enable
pnpm install --frozen-lockfile
pnpm run build
npm publish --registry https://nexus.example.com/repository/npm-hosted
```

客户安装：

```bash
npm install -g skills --registry https://nexus.example.com/repository/npm-hosted
```

### 7.2 方式 B：离线下发 `.tgz`

这是封闭环境路径。

构建：

```bash
cd apps/skills-cli
corepack enable
pnpm install --frozen-lockfile
pnpm run build
npm pack
```

客户安装：

```bash
npm install -g ./skills-1.5.9.tgz
```

### 7.3 方式 C：预装到运维机或发布机

如果 CLI 只给少数人使用，可以直接预装：

```bash
npm install -g skills --registry https://nexus.example.com/repository/npm-hosted
```

## 8. CLI 需要改哪些配置

结论：

- **不改源码**
- 只改运行配置

### 8.1 查询 / 安装链路

给 CLI 配 Catalog 地址和 token：

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>
```

或显式传参：

```bash
skills find pdf --registry https://catalog.example.com
skills add @company.skill/pdf-skill --registry https://catalog.example.com
```

适用命令：

- `skills find`
- `skills add`
- `skills list`
- `skills update`
- `skills remove`

### 8.2 发布 / 导入链路

除了 Catalog 地址外，还要让 `npm publish` 能发到 Nexus。

准备 `.npmrc`：

```ini
registry=https://nexus.example.com/repository/npm-hosted/
//nexus.example.com/repository/npm-hosted/:_authToken=REPLACE_ME
```

典型环境变量：

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/abs/path/to/.npmrc
```

关键参数：

- `--registry`
  - 指向 Catalog
- `--npm-registry`
  - 指向 Nexus npm hosted 仓
- `--publish-registry`
  - 写入 package metadata / release-record 的 registry 地址

## 9. 现场部署步骤

按这个顺序落：

### Step 1. 确认客户现场依赖

确认：

- Postgres 已可连通
- Nexus npm hosted 仓已创建
- CLI 外部访问 Nexus 地址已确定
- k3s Ingress 路径已确定

### Step 2. 构建并推送镜像

构建：

```bash
docker build -f zarf/docker/Dockerfile -t registry.example.com/skill-registry-catalog:1.0.0 .
docker build -f zarf/docker/skills-web.Dockerfile -t registry.example.com/skill-registry-web:1.0.0 .
```

然后推送到客户可拉取的镜像仓。

### Step 3. 准备 Secret

至少要准备：

- Postgres DSN
- `reader token`
- `authoring token`
- `publisher token`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
- web 侧代理用 `reader token`

### Step 4. 改 k3s 清单

基于 [zarf/k8s/customer-k3s-example.yaml](../zarf/k8s/customer-k3s-example.yaml) 替换：

- image
- namespace
- host / domain
- Secret 引用
- ingressClass / annotation
- 保持 `catalog.spec.replicas: 1`

### Step 5. 应用 k3s 清单

先创建：

- Namespace
- Secret

再创建：

- `catalog` Deployment / Service
- `skills-web` Deployment / Service
- Ingress

### Step 6. 验证浏览器入口

验证：

- 能打开 web 域名
- 搜索接口正常
- skill 详情页正常

如果这里失败，先查：

- Ingress
- `/api` 代理
- reader token 注入

### Step 7. 安装 CLI 到运维机

任选一种方式：

- npm 仓安装
- `.tgz` 安装

安装后先跑：

```bash
skills --help
```

### Step 8. 验证只读链路

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills list
```

### Step 9. 验证安装链路

```bash
skills add @company.skill/pdf-skill -a codex -y
```

验证结果：

- 安装目录写入 `./.agents/skills/`
- 项目状态写入 `./skills-lock.json`

### Step 10. 验证发布链路

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/abs/path/to/.npmrc

skills publish /abs/path/to/skill
```

如果是历史 skill：

```bash
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --registry https://catalog.example.com \
  --npm-registry https://nexus.example.com/repository/npm-hosted/ \
  --publish-registry https://nexus.example.com/repository/npm-hosted/
```

## 10. 现场标准验证命令

### 10.1 只读验证

```bash
export SKILLS_REGISTRY_URL=https://catalog.example.com
export SKILLS_REGISTRY_TOKEN=<reader-token>

skills find pdf
skills find pdf --json
skills list
```

### 10.2 安装验证

```bash
skills add @company.skill/pdf-skill -a codex -y
skills update @company.skill/pdf-skill
skills remove @company.skill/pdf-skill -y
```

### 10.3 发布验证

```bash
export SKILLS_REGISTRY_TOKEN=<publisher-token>
export NPM_CONFIG_USERCONFIG=/abs/path/to/.npmrc

skills publish /abs/path/to/skill
```

## 11. 故障排查表

### 11.1 `skills find` 失败

先查：

- `SKILLS_REGISTRY_URL` 是否正确
- CLI 到 Catalog 网络是否通
- token 是否正确
- web 同域代理是否把 `/api` 指到了 Catalog

### 11.2 `skills add` 时报 `fetch failed`

优先查：

- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 是否没配
- 它是否仍是集群内地址
- CLI 机器是否能直接访问 Nexus tarball URL
- Nexus tarball 是否允许当前认证方式读取

### 11.3 `skills publish` 失败

优先查：

- `.npmrc` 是否生效
- `NPM_CONFIG_USERCONFIG` 是否指向正确文件
- `--npm-registry` 是否与目标 Nexus 一致
- 发布账号是否对该 npm hosted 仓有写权限

### 11.4 浏览器能打开但列表为空

优先查：

- web 代理是否注入了 `reader token`
- Catalog 是否已成功连接 Postgres
- Catalog 是否能访问 Nexus metadata

## 12. 上生产前检查单

- 已关闭 sample seed
- 已关闭 mock registry
- Catalog 已使用正式 Postgres
- `SKILL_REGISTRY_NPM_REGISTRY_URL` 指向正式 Nexus
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 对真实 CLI 可访问
- `skills publish` 使用正式发布凭据
- 已验证客户侧 `skills add` 能真实下载 bundle
- 已明确项目级安装与全局安装默认策略
- 已准备 token 轮换、审计、回滚方案

## 13. 风险提示

交付前必须明确告诉客户：

1. Web 不是 CLI 的替代
2. CLI 不是集群内常驻服务
3. 这套仓库已有可用交付资产，但还没有完整 Helm、secret 管理和正式 CI/CD release pipeline

## 14. 附录：组件边界

### 14.1 Web

职责：

- 提供搜索、浏览、会话信息和面向运营的 UI
- 通过同域 `/api/...` 调 Catalog

### 14.2 Catalog

职责：

- 搜索 skill
- 返回 install target
- 处理 packaging / release-record / audit / governance
- 从 Nexus 回查 package metadata 和 tarball

### 14.3 Postgres

职责：

- 持久化 catalog、audit、幂等、治理状态

### 14.4 Nexus

职责：

- 存 skill bundle tarball
- 提供 npm metadata

## 15. 关联文档

- [README.md](../README.md)
- [docs/local-nexus-e2e.zh-CN.md](local-nexus-e2e.zh-CN.md)
- [docs/batch-import-skills.zh-CN.md](batch-import-skills.zh-CN.md)
