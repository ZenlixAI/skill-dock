# Skills Web 架构深化设计

## 目标

把 `apps/skills-web` 从“页面直连 adapter + 页面内自管状态机”的结构，深化成几个有清晰 interface 的 module，让前端更容易测试、演进和导航。

这次改动的目标不是重做产品交互，而是提升：

- **locality**：会话、权限、打包工作流、Catalog 写操作的复杂度集中在少数 module
- **leverage**：页面通过更小的 interface 获取行为，而不是重新拼装底层步骤
- **testability**：测试面从大页面组件下沉到真正的 interface
- **AI-navigability**：未来修改 `skills-web` 时，不必在多个页面里来回追同一种行为

## 非目标

- 不改后端 API contract
- 不改 `Catalog Database`、`State Store`、`Release Record`、`Governance Change` 的领域语义
- 不改 `Package Name` 是唯一公开身份的 ADR
- 不改变页面的主要用户流程、路由结构和视觉方向
- 不把所有页面都重写成完全新的 UI 结构

## 当前问题

### 1. Session seam 半实半虚

仓库已经有 `SessionProvider`，但多数页面仍然自己调用 `getSession()`：

- `ImportPage`
- `SkillDetailPage`
- `VersionDetailPage`
- `GovernancePage`
- `AuthoringFlowPage`

结果是：

- authenticated session 的加载逻辑分散
- capability posture 的解释分散
- package-scoped capability 判断分散
- 页面既依赖 `SessionProvider`，又旁路 `SessionProvider`

这个 seam 只在 `RouteGuards`、`AppShell`、`LoginPage` 上成立，在页面层并不是真正的 seam。

### 2. `ImportPage` 是超大浅 module

`ImportPage.tsx` 同时承担：

- 作者输入编辑
- `prepare -> validate -> pack -> cleanup` 工作流
- mutation result 持久化到 URL
- handoff 生成
- 错误解释
- permission gating 所需的部分 posture 计算

如果删除这个 module，复杂度不会消失，只会散回更多页面或子组件。这说明真正该存在的是更深的 module，而不是一个超大页面。

### 3. Catalog 写操作 ceremony 大量重复

`SkillDetailPage` 与 `VersionDetailPage` 都重复了同一类行为：

- submit state
- error normalize
- mutation result 写入 URL
- activity log
- refresh detail data

这些行为围绕的其实是同一个领域概念：对 Catalog 进行写操作并暴露审计线索。现在却作为页面内 implementation 重复存在。

### 4. API adapter seam 过宽

`api/client.ts` 与 `api/mockClient.ts` 都把多类行为揉在一起：

- session
- Catalog read
- Skill Bundle packaging
- Published Release 写入
- Governance Change 写入

其中 `mockClient.ts` 还同时承担 adapter 和 in-memory 状态拥有者。它没有稳定 seam，只有一个大对象。

## 约束

### 领域约束

根据 `CONTEXT.md` 与 ADR：

- **Package Name** 仍然是唯一公开身份
- **Catalog** 继续治理 discovery、install resolution、governance state
- npm-compatible repository 仍只是 artifact storage
- `Release Record`、`Governance Change`、`Install Target` 等概念保持不变

因此这次改动只能调整前端 module 和 seam，不能把领域判断搬出既有 contract。

### 迁移约束

- 允许重构期间短时间存在新旧双轨
- 最终行为要保持等价
- 优先建立新 seam，再迁移页面，最后删除旧重复逻辑
- 测试允许先红后绿，但每一步都必须有明确验证点

## 方案对比

### 方案 A：只拆页面文件

把大页面拆成更多组件和局部工具函数。

优点：

- 改动快
- 视觉层文件更小

缺点：

- 复杂度只是搬家
- deletion test 不通过
- 同一种工作流仍会跨多个文件跳转

### 方案 B：按领域建立深 module，再迁移页面

先建立：

- `registry session module`
- `skill packaging workflow module`
- `catalog mutation module`
- `catalog adapter modules`

然后让页面只消费这些 interface。

优点：

- locality 最好
- leverage 最好
- 测试面清晰
- 后续扩展新的页面或 mutation 时可以复用

缺点：

- 一次性改动比纯拆文件更大
- 需要先补 hook / module 测试

### 方案 C：保持页面结构，只收敛到更大的公共 helper

把重复逻辑抽成 helper，但不形成明确 seam。

优点：

- 改动中等

缺点：

- interface 仍然浅
- helper 会逐渐退化成“共享杂物间”

### 推荐

采用 **方案 B**。

## 目标结构

### 1. Registry Session Module

这个 module 是 `skills-web` 唯一的 session seam。

它的 interface 负责：

- 提供当前 `RegistrySession`
- 暴露 loading / unavailable / authenticated posture
- 解释当前 session 的 write posture
- 统一判断 package-scoped capability
- 提供 refresh 行为

页面不再直接 `getSession()`。

保留：

- `SessionProvider`
- `useRegistrySession()`

新增或重组：

- 让 provider 暴露更高层的 capability / posture 结果
- `RouteGuards` 和 `PermissionGate` 只消费 session module 的 interface
- 页面通过同一 seam 获取 session 和 capability 结果

### 2. Skill Packaging Workflow Module

这个 module 承担 `Skill Bundle` 的前端工作流 interface。

它的 interface 负责：

- 维护 `prepare -> validate -> pack -> cleanup` 的阶段状态
- 维护最新 review 数据
- 解释能否执行下一步
- 统一生成 handoff
- 统一持久化 mutation result
- 统一解释 packaging 相关错误

页面不再知道每一步的内部 ceremony，只知道：

- 当前 stage
- 当前 review
- 当前 actions
- 当前 handoff

### 3. Authoring Draft Module

把 `platform_authoring` 来源的作者输入单独做成一个 module。

它的 interface 负责：

- 模板切换
- 文件列表
- 当前选中文件
- 新增 / 删除文件约束
- 原始 JSON 与文件模型同步
- 产出 `platform_authoring.files`

这样 `ImportPage` 不再同时持有“作者输入模型”和“打包工作流状态机”两套复杂度。

### 4. Catalog Mutation Module

这个 module 负责 `Release Record` 和 `Governance Change` 相关写操作。

它的 interface 负责：

- 执行 mutation
- 维护 submit / action state
- normalize error
- 生成 mutation result
- 将 mutation result 持久化到 URL
- 触发页面所需 refresh contract

目标不是把所有表单字段都抽走，而是把“写 Catalog 的通用行为”变成真正的 seam。

### 5. Catalog Adapter Modules

把当前过宽的 adapter 按领域拆开：

- session adapter
- catalog read adapter
- packaging adapter
- catalog mutation adapter

同时把 mock 侧的 in-memory 状态从 adapter object 中拆出来，形成一个小型前端 `State Store`：

- fixtures
- pagination helpers
- state mutation helpers
- audit recording helpers

这样 mock adapter 只是 concrete adapter，而不是状态、行为、fixture 的混合体。

## 页面迁移设计

### `ImportPage`

迁移后保留的职责：

- 表单布局
- 工作流展示
- 调用 `PermissionGate`
- 组合作者输入和打包工作流的 UI

迁移走的职责：

- `platform_authoring` 文件编辑规则
- workflow 阶段迁移
- mutation result URL 持久化
- handoff 生成
- packaging 错误解释
- 自行拉取 session

目标是把它从“业务状态拥有者”降成“组合层 module”。

### `SkillDetailPage`

迁移后：

- 不再直接 `getSession()`
- retire skill 的 submit / persist / refresh ceremony 迁到 catalog mutation module
- 页面只保留 retire 表单和详情渲染

### `VersionDetailPage`

迁移后：

- 不再直接 `getSession()`
- `release-record` 与 `governance` 两类 mutation 统一走 catalog mutation module
- handoff 读取仍可保留在页面层，但 mutation result 持久化不再由页面自管

### `GovernancePage`

迁移后：

- 不再自己拉 session
- 只消费 session posture
- 自己仍然保留筛选和治理时间线展示

### `AuthoringFlowPage`

迁移后：

- 不再自己拉 session
- 只消费 session posture
- 继续作为平台作者台的聚合入口

## 目录建议

目标不是机械地增加目录层级，而是让 seam 清楚。

建议形成如下结构：

- `src/session/`
  - provider / hooks / posture helpers
- `src/features/packaging/`
  - `useAuthoringDraft`
  - `usePackagingWorkflow`
  - packaging-specific URL state helpers
- `src/features/catalog-mutations/`
  - release record mutation hook
  - governance mutation hook
  - shared mutation URL helpers
- `src/api/`
  - session adapter
  - catalog read adapter
  - packaging adapter
  - catalog mutation adapter
  - mock state store

如果迁移过程中发现某个 feature 目录只剩单文件薄包装，则允许回退成更少的文件。目标是深 module，不是目录数量。

## 测试设计

### 原则

这次重构的核心原则是：

- **the interface is the test surface**
- 页面测试变薄
- 行为状态机和 capability 判断下沉到 hook / module 测试

### Session Module 测试

验证：

- loading / unavailable / authenticated posture
- mutation posture
- package-scoped capability 判断
- refresh 行为

### Authoring Draft Module 测试

验证：

- 模板初始化
- 选中文件切换
- 新增 / 删除文件约束
- 原始 JSON 与文件模型同步
- 产出 `platform_authoring.files`

### Packaging Workflow Module 测试

验证：

- stage transition
- `prepare` 成功后清空后续结果
- `validate` / `pack` guard rails
- cleanup 重置状态
- handoff 生成
- mutation result URL 持久化
- packaging 错误解释

### Catalog Mutation Module 测试

验证：

- release record 成功 / 失败
- skill governance 成功 / 失败
- version governance 成功 / 失败
- mutation result URL 持久化
- request id 与错误信息传递
- refresh contract 是否触发

### 页面测试

页面测试保留，但改成更薄的集成验证：

- 页面是否正确消费 hook state
- 页面在不同 posture 下是否显示正确 CTA / fallback
- 页面在 mutation 完成后是否显示正确结果

不再在多个页面里重复测试同一种内部状态迁移细节。

## 迁移顺序

### 第 1 步：Session seam 做实

- 让页面停止直接 `getSession()`
- provider 暴露统一 posture / capability interface
- 调整 `RouteGuards` 与 `PermissionGate`

这是后续迁移的基础。

### 第 2 步：抽离 Authoring Draft 与 Packaging Workflow

- 先补新 hook 测试
- 再迁移 `ImportPage`
- 最后删除页面中的旧状态机和作者输入逻辑

这是收益最大的 deepening opportunity，应优先完成。

### 第 3 步：抽离 Catalog Mutation Module

- 先补 mutation hook 测试
- 迁移 `SkillDetailPage`
- 迁移 `VersionDetailPage`
- 清理重复的 mutation persistence 逻辑

### 第 4 步：拆 Adapter 与 Mock State Store

- 保持对外导出不立即剧烈变化
- 先在内部按领域拆分
- 再让页面与 hook 指向新的 adapter seam

这样可以降低迁移风险。

## 风险与缓解

### 风险 1：重构期间新旧 seam 并存

缓解：

- 每迁完一个页面就删除对应旧逻辑
- 避免长期双轨

### 风险 2：hook 过度抽象

缓解：

- hook 只隐藏真实重复的 workflow / mutation ceremony
- 不抽离只被一个页面使用且没有复杂不变量的纯展示逻辑

### 风险 3：mock 行为与 real adapter 偏离

缓解：

- 保持 adapter contract 不变
- 把 mock 变化集中在 state store helpers
- 用现有 adapter tests 锁定响应 shape

## 验收标准

完成后应满足：

- `ImportPage` 不再自持完整 packaging 状态机
- 页面层不再直接 `getSession()`
- `SkillDetailPage` 和 `VersionDetailPage` 不再重复 mutation persistence ceremony
- `api/client.ts` 不再是唯一宽 adapter 入口
- `mockClient.ts` 不再同时承担 fixture、state mutation、adapter orchestration 的全部职责
- 现有用户流程与 contract 保持等价
- 单元测试重点落在新 interface，而不是继续堆积到大页面组件

## 实施建议

这次重构适合按一条实现计划执行，但内部应拆成几个可独立验证的小步：

1. session seam
2. authoring draft
3. packaging workflow
4. catalog mutations
5. adapter / mock store cleanup

每一步都应该先补失败测试，再迁移实现，再删旧代码。
