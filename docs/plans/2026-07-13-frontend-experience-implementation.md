# Frontend Experience Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 Kafka Manager 重构为支持日间/夜间主题、Topic 上下文工作区和专业 JSON 消息体验的高可用研发控制台。

**Architecture:** 保留现有 React + TypeScript 和 API，不引入大型组件库。通过主题 Context、统一 SVG 图标/反馈/布局组件和显式 Topic 工作区状态重建前端信息架构；CSS 使用语义化设计令牌覆盖双主题与响应式布局。

**Tech Stack:** React 18、TypeScript、Vite、Vitest、原生 CSS、内联 SVG。

---

### Task 1: 主题与基础组件

**Files:**
- Create: `web/src/theme/ThemeProvider.tsx`
- Create: `web/src/components/Icon.tsx`
- Create: `web/src/components/Toast.tsx`
- Modify: `web/src/components/Common.tsx`
- Test: `web/src/theme/ThemeProvider.test.tsx`

1. 测试主题默认跟随系统、手动切换和 localStorage 持久化。
2. 实现 `light`、`dark`、`system` 主题 Context。
3. 实现统一内联 SVG 图标、Tooltip/可访问名称、Toast 和 Tabs。
4. 统一 Dialog 的 Escape、ARIA 和背景滚动锁定。

### Task 2: 应用框架和设计令牌

**Files:**
- Modify: `web/src/main.tsx`
- Modify: `web/src/App.tsx`
- Replace: `web/src/styles.css`
- Remove: `web/src/backups.css`
- Remove: `web/src/features.css`

1. 用 ThemeProvider 包裹应用。
2. 重做可折叠导航、移动抽屉、顶部集群上下文栏和主题切换。
3. 建立深浅主题语义令牌、统一排版/按钮/表格/表单/状态样式。
4. 添加 375/768/1024/1440 响应式规则、焦点和 reduced-motion。

### Task 3: Dashboard 异常优先重构

**Files:**
- Modify: `web/src/pages/DashboardPage.tsx`

1. 顶部展示离线、ISR 和 Lag 告警摘要。
2. 重构规模指标、集群健康卡和 Lag 趋势 SVG。
3. 增加直达 Topics 与 Consumer Groups 操作。

### Task 4: Topic 工作区

**Files:**
- Modify: `web/src/App.tsx`
- Replace: `web/src/pages/TopicsPage.tsx`
- Create: `web/src/pages/TopicWorkspace.tsx`
- Test: `web/src/pages/TopicWorkspace.test.tsx`

1. 建模选中 Topic 和工作区 Tab。
2. Topic 列表增加健康摘要、复制和消息快捷入口。
3. 实现概览、消息、分区、配置四个标签。
4. 将分区扩容、配置修改和删除集中到对应区域。

### Task 5: JSON 消息体验

**Files:**
- Create: `web/src/components/JsonViewer.tsx`
- Create: `web/src/components/JsonViewer.test.tsx`
- Replace: `web/src/pages/MessagesPage.tsx`

1. 测试 JSON 识别、格式化、折叠和原始视图。
2. 实现递归 JSON Viewer、语法着色、复制和展开控制。
3. 消息列表增加 Value 预览与 JSON/Text 类型标记。
4. 消息查询支持可选固定 Topic；Topic 工作区自动继承。
5. 生产器默认 JSON 模式，发送前校验，并支持 Text 模式。

### Task 6: 其余页面一致性

**Files:**
- Modify: `web/src/pages/ConsumersPage.tsx`
- Modify: `web/src/pages/SettingsPage.tsx`
- Modify: `web/src/pages/AuditPage.tsx`
- Modify: `web/src/features/auth/LoginPage.tsx`

1. 统一页面头、卡片、筛选、表格、空状态和危险操作。
2. 消费组突出 Lag 与异常分区。
3. 配置页按集群连接和保留策略分组。
4. 审计页增加可读状态和过滤布局。
5. 登录页适配深浅主题并移除概念稿式装饰。

### Task 7: 验证与文档

**Files:**
- Modify: `README.md`

1. 运行前端测试、类型检查和生产构建。
2. 运行 Go 全量测试，确认嵌入式构建无回归。
3. 使用真实 Kafka 配置启动，验证登录、Dashboard、Topic 和消息 API。
4. 在可用浏览器中检查双主题和主要宽度；若浏览器工具不可用，明确记录限制。
5. 检查 Git diff、构建产物和敏感配置，提交实现。
