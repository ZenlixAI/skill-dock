# 本地 Nexus 真实 E2E

这条路径给“需要更接近企业仓库行为”的本地联调准备。和 Verdaccio 相比，Nexus 更重，但更接近很多客户现场会出现的 npm-compatible 制品仓。

目标链路仍然是同一条：

1. `skills publish` 或批量导入脚本生成 bundle
2. `npm publish` 把 bundle 推到 Nexus npm hosted repository
3. Catalog `release-record` 从 Nexus 读取 package metadata
4. `skills add` 根据 install target 下载 tarball 并安装

## 这套方案的边界

这份资产默认采用：

- `publish` 需要认证
- tarball / metadata 允许匿名读取
- `publish` 必须直接指向 npm hosted repository；如果后面要用 group 做读流量，Community 版也仍然要把写流量打到 hosted

原因和认证版 Verdaccio 一样：

- CLI `publish` 通过本机 `npm publish`
- CLI `add` 下载 tarball 时当前不会附带 npm registry token
- Nexus npm group 的 `writableMember` 是 Pro-only 能力，Community 版不能把 group 当 publish 入口

所以如果你把 Nexus 调成“读取也必须认证”，当前链路里：

- `publish` 仍可能成功
- `release-record` 仍可能成功
- `find` 仍可能成功
- `add` 只有在 install-side 也能拿到匹配的 `.npmrc` / `NPM_CONFIG_USERCONFIG` 时才会成功

这不是 Nexus 问题，而是“本地调用面有没有把 npm auth context 真的传进 install-side”。

## 仓库里新增了什么

- `make nexus-run`
  - 本机直接启动一个 Nexus 容器
- `make nexus-bootstrap`
  - 默认启用 anonymous read
  - 等待 Nexus 就绪
  - 把初始 admin 密码改成固定测试密码
  - 按模式打开或关闭匿名读取
  - 创建 `skills-npm` hosted repository
  - 写出隔离的 `.npmrc`
- `make nexus-bootstrap STRICT=1`
  - 关闭 anonymous read，用来验证 tarball / metadata 读取也要鉴权的场景
- `make registry-run REGISTRY_BACKEND=nexus`
  - 把 Catalog 指向 Nexus npm repository
- `make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus`
  - 让 CLI 用隔离 `.npmrc` 发布到 Nexus
- `make cli-add PACKAGE=@platform.skill/your-skill-name NPM_BACKEND=nexus`
  - 让 `skills add` 也显式复用 Nexus `.npmrc`
- `make batch-import MANIFEST=/abs/path/to/import.json NPM_BACKEND=nexus`
  - 让批量导入工具用同一个 Nexus 仓
- `make compose-up COMPOSE_STACK=nexus`
  - 一次拉起 `catalog + postgres + nexus`

默认参数：

- Nexus UI / API：`http://127.0.0.1:8081`
- npm repository：`http://127.0.0.1:8081/repository/skills-npm`
- Catalog：`http://127.0.0.1:18080`
- 本地状态目录：`/private/tmp/skill-registry-nexus`
- 测试 admin 密码：`admin123456`

## 当前验证状态

这套资产我已经在 2026-06-03 本机验证了下面这些步骤：

- `make nexus-run`
- `make nexus-bootstrap`
- `make registry-run REGISTRY_BACKEND=nexus`
- `npm publish` 到 Nexus hosted repo
- `make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus`
- `make cli-find QUERY=nexus-cli-smoke`
- `make cli-add PACKAGE=@platform.skill/nexus-cli-smoke`
- `make nexus-bootstrap STRICT=1`
- 严格模式下：
  - `make cli-add PACKAGE=@platform.skill/nexus-cli-smoke` 会在 tarball 下载阶段 `401 Unauthorized`
  - `make cli-add PACKAGE=@platform.skill/nexus-cli-smoke NPM_BACKEND=nexus` 会成功

也确认了：

- Nexus hosted npm repository 会被自动创建
- `nexus-bootstrap` 现在会自动通过 REST API 接受 Community EULA
- `npm adduser --auth-type=legacy` 的登录态能自动写入 `.npmrc`
- Catalog 能正常启动并指向 Nexus repository URL
- CLI 现在已经有一层 npm auth adapter，会从匹配的 `.npmrc` / `NPM_CONFIG_USERCONFIG` 解析 tarball 下载认证头
- 触发过的 `403 Forbidden` 根因已经定位：不是 package metadata，也不是 npm 11 专属兼容性，而是 Nexus Community 首次启动后还没接受 EULA
- 严格读鉴权模式下，adapter 本身是够的；真正缺口是调用面是否把生成出来的 Nexus `.npmrc` 传给 `skills add`

这次用于收敛 403 的关键对照是：

- 同一个 hosted repo 上，`npm whoami=admin` 但 `npm publish` 最小裸包仍然 `403`
- 同样的最小包在 `npm 11.12.1`、容器里的 `npm 10.8.2`、`npm 8.19.4` 上都能复现
- 直接 `curl PUT` 同一 package 路径时，Nexus 返回体明确是：
  - `You must accept the End User License Agreement (EULA) through the onboarding wizard or REST API before proceeding.`
- 对 `/service/rest/v1/system/eula` 做 `POST {"accepted":true,...}` 后，最小包 `npm publish` 立即恢复成功
- 从干净状态重新验证后，`bootstrap -> cli publish -> cli find -> cli add` 已经可以完整闭环
- 把 anonymous read 关闭后：
  - 不带 `NPM_CONFIG_USERCONFIG` 的 `cli-add` 会稳定复现 `Failed to download tarball: 401 Unauthorized`
  - 显式带 `/private/tmp/skill-registry-nexus/.npmrc` 的 `cli-add` 会恢复成功

所以当前这份交付现在更准确的定位是：

- Docker 化 Nexus 测试资产：已补齐
- Nexus 自动初始化：已跑通
- Catalog 对接 Nexus：已跑通
- Nexus Community onboarding / EULA 接受：已纳入 bootstrap
- `npm publish` 最后一跳：根因已定位并可通过 bootstrap 自动消除
- 严格 tarball 鉴权模式：CLI adapter 已具备能力；repo 里的本地 E2E 现在通过 `make cli-add PACKAGE=@platform.skill/your-skill-name NPM_BACKEND=nexus` 提供标准接线

## 路径 A：分终端启动，最容易排查

### 终端 1：启动 Nexus

第一次建议先清理旧状态：

```bash
make nexus-reset
```

启动：

```bash
make nexus-run
```

保持这个终端不要退出。

### 终端 2：初始化 Nexus

```bash
make nexus-bootstrap
```

如果你要验证严格读鉴权模式，改成：

```bash
make nexus-bootstrap STRICT=1
```

预期结果：

- admin 密码被固定成 `admin123456`
- `skills-npm` 仓库创建成功
- 写出：

```bash
/private/tmp/skill-registry-nexus/.npmrc
```

### 终端 3：启动指向 Nexus 的 Catalog

```bash
make registry-reset
make registry-run REGISTRY_BACKEND=nexus
```

保持这个终端不要退出。

### 终端 4：发布 / 搜索 / 安装

发布一个本地 skill：

```bash
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus
```

然后继续验证：

```bash
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name
```

## 路径 B：先用 compose 拉起 Catalog + Postgres + Nexus

如果你不想分开手工起 Catalog / Postgres，可以直接：

```bash
make compose-up COMPOSE_STACK=nexus
```

然后在另一个终端做一次初始化：

```bash
make nexus-bootstrap
```

之后继续：

```bash
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name
```

如果当前是严格模式，最后一步改成：

```bash
make cli-add PACKAGE=@platform.skill/your-skill-name NPM_BACKEND=nexus
```

结束时：

```bash
make compose-down COMPOSE_STACK=nexus
```

## 批量导入

如果你要验证批量导入和 Nexus 的组合：

```bash
make batch-import MANIFEST=/abs/path/to/import.json NPM_BACKEND=nexus
```

它会复用：

```bash
/private/tmp/skill-registry-nexus/.npmrc
```

## 一套最短命令

```bash
# terminal 1
make nexus-reset
make nexus-run
```

```bash
# terminal 2
make nexus-bootstrap
```

```bash
# terminal 3
make registry-reset
make registry-run REGISTRY_BACKEND=nexus
```

```bash
# terminal 4
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name
```

严格模式最短链路：

```bash
# terminal 2
make nexus-bootstrap STRICT=1
```

```bash
# terminal 4
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name NPM_BACKEND=nexus
```

## 失败时优先检查

如果 `make nexus-bootstrap` 失败，优先检查：

1. Nexus 容器是否仍在运行
2. `/private/tmp/skill-registry-nexus/admin.password` 是否已经生成
3. `http://127.0.0.1:8081/service/rest/v1/status` 是否可访问
4. 当前机器是否安装了 `expect`

如果 `make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=nexus` 失败，优先检查：

1. `make nexus-bootstrap` 是否已经成功跑过
2. `/private/tmp/skill-registry-nexus/.npmrc` 是否存在
3. `SKILL_PATH` 下的 `SKILL.md` / `package.json` 是否满足当前发布契约
4. Catalog 是否通过 `make registry-run REGISTRY_BACKEND=nexus` 启动
5. 同版本是否已经发布过
6. 如果仍然是 `403`，先直接看响应体是不是 EULA / onboarding 问题，而不是先怀疑 skill metadata

如果 `find` 正常但 `add` 失败，优先检查：

1. Nexus anonymous access 是否被关掉了
2. `dist.tarball` URL 是否仍然指向 `skills-npm`
3. 是否把 Nexus 改成了 tarball 读取强鉴权模式
4. 如果是严格模式，当前命令是否显式带了 `NPM_CONFIG_USERCONFIG=/private/tmp/skill-registry-nexus/.npmrc`，或者直接使用 `make cli-add PACKAGE=@platform.skill/your-skill-name NPM_BACKEND=nexus`
