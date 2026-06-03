# 本地 Verdaccio 认证 E2E

这条路径是给“发布需要登录”的本地真实联调准备的。

这份文档分成两种口径：

## 1. 推荐路径：发布需要登录，读取允许匿名

这是当前系统能够完整对拍的认证 E2E：

1. `npm publish` 需要 Verdaccio 登录态
2. Catalog `release-record` 从 Verdaccio 读取 metadata
3. `skills add` 仍可匿名拉取 tarball

原因很直接：

- CLI `publish` 走本地 `npm publish`
- CLI `add` 下载 tarball 时只是直接 `fetch(tarballUrl)`
- `add` 不会附带 npm registry token

因此，只有“发布鉴权、读取匿名”的配置才能完整覆盖：

- `publish`
- `release-record`
- `find`
- `add`

## 2. 严格路径：发布和读取都需要登录

这条路径当前只能部分验证，不是完整 E2E 目标。

它可以验证：

- `npm publish`
- `release-record`

但不能把 `skills add` 当成应当成功，因为 tarball fetch 目前没有 npm 鉴权能力。

## 仓库内置配置

- [zarf/docker/verdaccio/auth.yaml](/Users/ultraman/workspace/skill_registry/zarf/docker/verdaccio/auth.yaml)
  - `publish: $authenticated`
  - `access: $all`
  - 推荐使用
- [zarf/docker/verdaccio/strict-auth.yaml](/Users/ultraman/workspace/skill_registry/zarf/docker/verdaccio/strict-auth.yaml)
  - `publish: $authenticated`
  - `access: $authenticated`
  - 仅用于验证系统当前边界

## 从零启动到发布成功

下面这组命令按终端拆开，目标是跑通“发布需要登录、读取允许匿名”的完整本地链路。

前提：

- 当前机器能执行 `npx verdaccio@6`
- 当前机器能执行 `npm adduser`
- 你已经有一个可发布的本地 skill 目录，例如 `/abs/path/to/skill`

### 终端 1：启动认证版 Verdaccio

第一次启动前先清理旧状态：

```bash
make verdaccio-auth-reset
```

启动 Verdaccio：

```bash
make verdaccio-run VERDACCIO_MODE=auth
```

保持这个终端不要退出。

### 终端 2：登录 Verdaccio

执行：

```bash
make verdaccio-login
```

按提示输入：

1. `Username`
2. `Password`
3. `Email`

成功后，登录态会写到：

```bash
/private/tmp/skill-registry-verdaccio-auth/.npmrc
```

你可以检查一下：

```bash
cat /private/tmp/skill-registry-verdaccio-auth/.npmrc
```

### 终端 3：启动指向认证版 Verdaccio 的 Catalog

第一次联调建议先清空 Catalog 本地库，避免旧 seed 还指向别的 npm registry：

```bash
make registry-reset
make registry-run REGISTRY_BACKEND=verdaccio-auth
```

保持这个终端不要退出。

### 终端 4：发布 skill

执行：

```bash
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=verdaccio-auth
```

预期结果：

- `npm publish` 成功
- Catalog `release-record` 成功
- CLI 最终输出成功发布的 `packageName` 和 `version`

### 终端 4：继续验证发现和安装

发布成功后，在同一个终端继续执行：

```bash
make cli-find QUERY=pdf
```

如果你发布的不是 `pdf`，把查询词换成你的 skill 名称关键字。

再执行：

```bash
make cli-add PACKAGE=@platform.skill/your-skill-name
```

在推荐路径下，这一步也应成功，因为 tarball 读取仍允许匿名。

### 发布失败时优先检查

如果 `make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=verdaccio-auth` 失败，优先检查这几项：

1. Verdaccio 是否仍在运行
2. `/private/tmp/skill-registry-verdaccio-auth/.npmrc` 是否存在
3. `SKILL_PATH` 下的 `SKILL.md` 和 `package.json` 是否满足当前发布契约
4. Catalog 是否是通过 `make registry-run REGISTRY_BACKEND=verdaccio-auth` 启动的
5. 同版本是否已经发布过

### 一套最短命令清单

如果你只想复制最短路径，可以按这个顺序执行：

```bash
# terminal 1
make verdaccio-auth-reset
make verdaccio-run VERDACCIO_MODE=auth
```

```bash
# terminal 2
make verdaccio-login
```

```bash
# terminal 3
make registry-reset
make registry-run REGISTRY_BACKEND=verdaccio-auth
```

```bash
# terminal 4
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=verdaccio-auth
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name
```

## 简版步骤

```bash
make verdaccio-run VERDACCIO_MODE=auth
make verdaccio-login
make registry-run REGISTRY_BACKEND=verdaccio-auth
make cli-publish SKILL_PATH=/abs/path/to/skill NPM_BACKEND=verdaccio-auth
make cli-find QUERY=your-skill
make cli-add PACKAGE=@platform.skill/your-skill-name
```

## 批量导入

认证版 Verdaccio 下批量导入用：

```bash
make batch-import MANIFEST=/abs/path/to/import.json NPM_BACKEND=verdaccio-auth
```

它会复用同一个 `NPM_CONFIG_USERCONFIG`，因此前提也是先 `make verdaccio-login`。

## 严格读取模式怎么用

如果你要验证“CLI `add` 还不支持私有 tarball 下载”，可手动启严格模式：

```bash
make verdaccio-run VERDACCIO_MODE=strict
```

然后重复：

1. `make verdaccio-login`
2. `make registry-run REGISTRY_BACKEND=verdaccio-auth`
3. `make cli-publish SKILL_PATH=... NPM_BACKEND=verdaccio-auth`
4. `make cli-add PACKAGE=...`

此时预期是：

- `publish` 可成功
- `release-record` 可成功
- `find` 可成功
- `add` 会在 tarball 下载阶段失败

这不是环境问题，而是当前能力边界。

## 什么时候再做严格私有 tarball 支持

只有当 CLI 安装链路补上以下能力时，严格模式才应升级为“已支持”：

- 从 npm auth context 解析 token
- 对 `tarballUrl` 下载附带认证
- 或让 Catalog / 下载网关返回另一类已认证下载地址
