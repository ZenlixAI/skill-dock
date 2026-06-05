# Platform Authoring Web 导入升级设计

## 目标

把 `apps/skills-web/src/pages/ImportPage.tsx` 里 `platform_authoring` 来源，从“手填 `files` JSON”升级成真实可用的作者入口：

- 支持模板预设
- 支持多文件编辑
- 支持新增 / 删除文件
- 仍然复用现有 `source.type=platform_authoring` 契约

这次改动只升级前端 authoring 体验，不改变后端 packaging 协议。

## 当前问题

当前 `platform_authoring` 流程的问题不在后端，而在 Web 入口层：

- 页面只暴露一个裸 `textarea`
- 用户必须手写 `Record<string, string>` JSON
- 无法以“创建一个 skill”的心智操作
- 无法快速从常见 skill 模板开始
- 多文件 authoring 几乎不可用

结果是：

- `platform_authoring` 理论上可扩展
- 实际上不适合作为平台 authoring / operator 工作台使用

## 非目标

- 不改 `POST /api/v1/skill-packaging/prepare` 契约
- 不新增后端模板 API
- 不把 `ImportPage` 做成完整 IDE
- 不在这次加入 diff、预览、重命名、拖拽排序
- 不改变 `git` 导入流程

## 约束

### 后端现状

`business/packaging/service.go` 里：

- `platform_authoring` 接收 `files`
- 服务端会自动写根级 `SKILL.md`
- 服务端会统一写 bundle 用的 `package.json`

因此前端不应该再让用户手改根级 `SKILL.md` 或把 package identity 逻辑搬到浏览器。

### 前端现状

`ImportPage` 目前：

- `sourceType=platform_authoring` 时只渲染 `inlineFiles` JSON textarea
- `buildPrepareRequest()` 直接把 JSON parse 成 `files`

因此本次改动的核心是：

- 保持 `buildPrepareRequest()` 的输出 shape 不变
- 只替换 `platform_authoring` 的输入方式

## 用户与场景

### 目标用户

- 平台作者
- operator
- publisher

### 核心场景

1. 选择一个常见 skill 模板开始
2. 修改模板生成的文件内容
3. 补充额外文件，例如 `docs/README.md`
4. 直接执行 `prepare -> validate -> pack`

## 方案对比

### 方案 A：固定表单

只暴露少量固定字段，例如：

- 命令内容
- prompt 内容
- 可选 README

优点：

- 实现快

缺点：

- 很快又会不够用
- 最终还是会逼用户回到 JSON

### 方案 B：模板驱动的多文件编辑器

提供：

- 模板选择
- 文件列表
- 文件内容编辑区
- 新增 / 删除文件
- 可选原始 JSON 高级模式

优点：

- 保持协议不变
- 足够覆盖大多数 authoring 场景
- 不会把页面做成过重 IDE

缺点：

- 前端状态管理比固定表单复杂

### 方案 C：接近 IDE 的工作台

提供：

- 文件树
- 重命名
- diff
- 预览
- 更复杂的文件操作

优点：

- 功能最强

缺点：

- 这次明显过重
- 会把 `ImportPage` 变成新产品，而不是升级现有导入面板

### 推荐

采用 **方案 B：模板驱动的多文件编辑器**。

## 目标交互

当 `sourceType=platform_authoring` 时，页面替换为新的 authoring 面板：

1. 顶部模板栏
2. 左侧文件列表
3. 右侧当前文件编辑器
4. 下方高级模式折叠区

普通用户默认不会看到裸 JSON。

## 模板设计

首版内置 3 个模板：

### `prompt`

默认生成：

- `prompts/main.md`

默认用途：

- 以单个 prompt 能力为主的 skill

### `command`

默认生成：

- `commands/run.txt`

默认用途：

- 以单个命令型能力为主的 skill

### `workflow`

默认生成：

- `workflows/main.md`

默认用途：

- 以结构化步骤说明为主的 skill

### 模板切换规则

- 初次进入 `platform_authoring` 时默认选 `command`
- 切换模板时，如果当前文件内容还是模板初始态，则直接替换
- 如果用户已经改过内容，则要求显式点击“重置为模板”

首版为了降低交互复杂度，可以先采用更保守规则：

- 模板切换立即重建文件集
- 但在有用户修改时给出明确确认提示

## 文件模型

前端内部维护一个有序文件列表：

- `path`
- `content`
- `locked`

其中：

- `path` 是相对路径
- `content` 是文件内容
- `locked=true` 的文件不能删除

首版核心规则：

- 模板默认文件为 `locked`
- 用户新增文件为可删除
- 根级 `SKILL.md` 不进入这个列表，由服务端生成

## 文件操作

### 支持

- 选择当前文件
- 编辑内容
- 新增文件
- 删除非锁定文件

### 暂不支持

- 重命名
- 拖拽排序
- 多选操作

### 新增文件约束

- 路径不能为空
- 路径不能重复
- 路径必须是相对路径
- 不允许 `..`
- 不允许根级 `SKILL.md`

## 高级模式

保留一个折叠区：

- 名称：`高级模式 / 原始 files JSON`

用途：

- 调试
- 粘贴已有 `files` payload
- 验证底层协议

规则：

- 默认折叠
- 展开时显示当前文件模型序列化后的 JSON
- 用户改 JSON 后，需要重新同步回文件列表
- 如果 JSON 非法，则阻止 `prepare`

## 请求构造

`buildPrepareRequest()` 继续输出：

- `source.type = platform_authoring`
- `source.files = Record<string, string>`

变化仅为：

- `files` 不再主要来自手填 textarea
- 而是来自 authoring 文件列表序列化结果

高级模式开启时：

- 若用户手改了 raw JSON，则以 JSON 为准
- 成功 parse 后回写到文件列表状态

## 页面结构改动

主要修改：

- `apps/skills-web/src/pages/ImportPage.tsx`
- `apps/skills-web/src/pages/ImportPage.test.tsx`
- `apps/skills-web/src/styles.css`

建议在 `ImportPage.tsx` 内新增这些局部辅助结构：

- 模板定义常量
- 文件列表状态初始化函数
- 文件列表 <-> `Record<string, string>` 转换函数
- 路径校验函数

如果实现后状态逻辑明显变复杂，可再把 authoring 面板拆到独立组件，但首版不强制要求。

## UI 细节

### 模板栏

- 使用按钮组或 `select`
- 显示模板名和一句短说明

### 文件列表

- 每行展示路径
- 当前文件高亮
- 非锁定文件显示删除按钮

### 编辑区

- 展示当前文件路径
- `textarea` 编辑内容

### 空状态

- 当没有选中文件时，自动选中第一项

### 高级模式

- 用折叠卡片承载
- 明确标注“仅调试/高级用法”

## 错误处理

前端本地错误需要早于请求暴露：

- 重复路径
- 非法路径
- 空路径
- 原始 JSON 非法

这些错误不需要打到后端才让用户看到。

但后端仍然是最终真相源：

- `prepare` 返回的 `warningDetails`
- `prepare` / `validate` 返回的 `errors`

前端不复刻服务端规则判断。

## 测试

至少补以下测试：

1. `platform_authoring` 默认呈现模板化文件编辑器，而不是只靠 JSON
2. 切换模板后会生成对应默认文件
3. 可以新增文件并编辑内容
4. 删除非锁定文件后，请求不会再带该文件
5. `preparePackaging` 被调用时，请求体里的 `source.files` 正确
6. 现有 `git` 导入 happy path 不回归

## 验收标准

- 用户不写 JSON 也能完成一次 `platform_authoring` prepare
- 模板 `prompt / command / workflow` 都可用
- 用户可以在页面里直接增删改文件
- `prepare -> validate -> pack` 现有工作流保持可用
- `git` 来源 UI 和行为不回归

## 实现顺序

1. 在 `ImportPage` 引入 authoring 文件模型与模板常量
2. 替换 `platform_authoring` 的 JSON textarea 主界面
3. 保留折叠式 raw JSON 高级模式
4. 增补本地校验与交互提示
5. 更新测试
6. 跑 web 端相关验证

## 风险与缓解

### 风险 1：ImportPage 状态复杂度上升

缓解：

- 先用局部纯函数隔离转换逻辑
- 若状态逻辑继续膨胀，再拆组件

### 风险 2：模板切换覆盖用户输入

缓解：

- 对已修改状态做确认
- 首版至少明确提示会重建文件集

### 风险 3：raw JSON 与可视编辑双向同步容易错

缓解：

- 默认隐藏高级模式
- 保证单一真相源始终是“文件列表状态”
- raw JSON 只有在用户显式编辑并成功 parse 后才回写

## 结论

这次应该把 `platform_authoring` 的 Web 入口，从“协议调试面板”升级成“真实作者入口”，但仍然保持：

- 后端契约不变
- ImportPage 仍然是 packaging workflow 页面
- 不把它做成过重 IDE

最合适的落点是：

- **模板驱动**
- **多文件编辑**
- **保留高级 JSON 通道**
