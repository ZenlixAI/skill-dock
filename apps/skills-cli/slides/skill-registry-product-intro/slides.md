---
theme: default
title: Skill Registry 产品介绍
titleTemplate: '%s'
info: 面向 AI Agent 技能的标准化生产与分发基础设施
class: text-left
drawings:
  persist: false
transition: slide-left
mdc: true
layout: cover
---

<style src="./styles.css"></style>

<div class="eyebrow">Internal Product Introduction</div>
# Skill Registry 产品介绍

<div class="hero-panel">
  <div class="quote-band">面向 AI Agent 技能的标准化生产与分发基础设施</div>
  <p class="bullets">目标：让内部产品团队在 10 分钟内统一认知它是什么、解决什么问题、边界在哪里。</p>
</div>

<!--
开场先定性：这不是单一工具演示，而是对 skill registry 产品边界的统一说明。
-->

---
layout: default
---

<div class="eyebrow">Why Now</div>
## 为什么要有它

<div class="hero-panel">
  <p class="quote-band">今天的问题，不是 skill 能不能写出来，而是写出来之后，怎么被可靠地交付、复用和治理。</p>
</div>

<div class="content-grid">
  <div class="content-card">
    <h3>来源分散</h3>
    <p class="bullets">skills 散落在仓库、文档和个人环境里，缺少统一入口。</p>
  </div>
  <div class="content-card">
    <h3>交付不统一</h3>
    <p class="bullets">不同团队靠手工复制、脚本、私有约定分发技能。</p>
  </div>
  <div class="content-card">
    <h3>版本不可控</h3>
    <p class="bullets">谁发布、发布了什么版本、谁在用，追踪成本高。</p>
  </div>
  <div class="content-card">
    <h3>企业交付复杂</h3>
    <p class="bullets">内网、权限、审计、稳定发布都让 skill 规模化落地变难。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Definition</div>
## 它到底是什么

<div class="hero-panel">
  <p class="quote-band">Skill Registry 不是单点工具，而是一条围绕 skill 的完整产品链路。</p>
  <p class="bullets">它把定义、校验、发布、搜索、安装、更新连接成一套标准化流程。</p>
</div>

<div class="pillars">
  <div class="pillar">
    <h3>标准化生产</h3>
    <p class="bullets">让 skill 有统一的包形态、校验和发布规则。</p>
  </div>
  <div class="pillar">
    <h3>标准化分发</h3>
    <p class="bullets">让 skill 可搜索、可安装、可更新，而不是靠口口相传。</p>
  </div>
  <div class="pillar">
    <h3>标准化治理</h3>
    <p class="bullets">让版本、权限、记录和可追溯性成为平台能力。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Audience</div>
## 它服务谁

<div class="role-strip">
  <div class="role-card">
    <h3>Skill 作者</h3>
    <p class="bullets">创建、整理、打包和发布技能。</p>
  </div>
  <div class="role-card">
    <h3>平台运营方</h3>
    <p class="bullets">维护规则、权限、发布流程和分发入口。</p>
  </div>
  <div class="role-card">
    <h3>使用团队</h3>
    <p class="bullets">发现、安装、更新并在业务场景中使用 skill。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Value</div>
## 它解决的核心问题

<div class="pillars">
  <div class="pillar">
    <h3>标准化发布</h3>
    <p class="bullets">把 skill 从“能写”变成“能稳定发布”。</p>
  </div>
  <div class="pillar">
    <h3>可发现可安装</h3>
    <p class="bullets">让技能像软件包一样可以搜索、选择和安装。</p>
  </div>
  <div class="pillar">
    <h3>可更新可治理</h3>
    <p class="bullets">让版本、升级、移除和记录进入统一体系。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Lifecycle</div>
## 能力全景

<div class="hero-panel">
  <p class="bullets">围绕 skill 全生命周期提供支撑，而不是只覆盖某一个操作点。</p>
</div>

<LifecycleFlow />

---
layout: default
---

<div class="eyebrow">Product Shape</div>
## 产品形态

<div class="content-grid" style="grid-template-columns: 1.1fr 0.9fr;">
  <div class="content-card">
    <ProductLoop />
  </div>
  <div class="content-card">
    <h3>四个关键表面</h3>
    <ul class="bullets">
      <li><strong>CLI</strong>：操作闭环入口</li>
      <li><strong>Catalog</strong>：元数据、校验、发布流程编排</li>
      <li><strong>npm-compatible registry</strong>：包存储与分发</li>
      <li><strong>Web</strong>：浏览、门户与面向人的入口</li>
    </ul>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Boundary</div>
## 边界与非目标

<div class="hero-panel">
  <p class="quote-band">它解决的是技能分发基础设施问题，不是所有 Agent 问题。</p>
</div>

<div class="boundary-list">
  <div class="boundary-item"><strong>不是</strong> 大模型平台</div>
  <div class="boundary-item"><strong>不是</strong> Agent 运行时本体</div>
  <div class="boundary-item"><strong>不是</strong> 通用工作流编排器</div>
  <div class="boundary-item"><strong>不是</strong> 所有知识资产的统一管理系统</div>
</div>

---
layout: center
class: text-center
---

<div class="eyebrow">Takeaway</div>
## 让 AI Agent 技能像软件包一样被管理

<div class="summary-grid">
  <div class="summary-card">
    <h3>它解决什么</h3>
    <p class="bullets">标准化 skill 交付，不只是一次性安装。</p>
  </div>
  <div class="summary-card">
    <h3>它覆盖什么</h3>
    <p class="bullets">从制作、发布到搜索、安装、更新和治理的全链路。</p>
  </div>
  <div class="summary-card">
    <h3>它不是什么</h3>
    <p class="bullets">不是模型平台，不是运行时，不是通用编排器。</p>
  </div>
</div>

<!--
结尾回收三点：它是什么、覆盖什么、边界是什么。
-->
