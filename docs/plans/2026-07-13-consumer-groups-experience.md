# Consumer Groups Experience Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将消费组列表和详情重构为清晰、响应式的 Lag 诊断工作台。

**Architecture:** 保留现有 Consumer Group API 和页面级状态，通过纯函数生成汇总、排序和过滤结果，并由 React 页面渲染诊断表格及右侧详情抽屉。样式使用现有语义化主题变量，不增加 UI 依赖。

**Tech Stack:** React 18、TypeScript、Vitest、原生 CSS。

---

### Task 1: 诊断数据模型

**Files:**
- Create: `web/src/pages/consumers-model.ts`
- Create: `web/src/pages/consumers-model.test.ts`

1. 先编写失败测试，覆盖总 Lag、积压消费组数、异常分区数、Lag 降序和分区过滤。
2. 运行 `pnpm --dir web test -- --run web/src/pages/consumers-model.test.ts`，确认因函数不存在而失败。
3. 实现 `summarizeConsumerGroups`、`sortConsumerGroups` 和 `filterPartitions`。
4. 重跑定向测试并确认通过。

### Task 2: 消费组诊断列表

**Files:**
- Modify: `web/src/pages/ConsumersPage.tsx`
- Modify: `web/src/components/Icon.tsx`

1. 增加搜索、状态筛选和 Lag 排序状态。
2. 渲染四项汇总指标和语义化诊断表格。
3. 使用可聚焦的消费组名称按钮打开详情，不使用整行点击。
4. 为窄屏提供紧凑信息层级。

### Task 3: 右侧详情抽屉

**Files:**
- Modify: `web/src/pages/ConsumersPage.tsx`
- Modify: `web/src/components/Common.tsx`

1. 基于现有弹窗焦点管理实现可复用 `Drawer`。
2. 详情顶部展示状态、协议、成员数、总 Lag 和异常分区。
3. 分区表格按 Lag 排序并增加 Topic/Partition 搜索、仅积压筛选与进度条。
4. 将 Offset 重置放入独立危险区，增加提交中状态并保留名称确认。

### Task 4: 双主题与响应式样式

**Files:**
- Modify: `web/src/polish.css`

1. 添加消费组汇总、表格、Lag 强度、抽屉、分区进度和危险区样式。
2. 在 1024px、768px 和 560px 断点调整列显示与抽屉宽度。
3. 检查深浅主题对比度、焦点状态和 reduced-motion。

### Task 5: 验证与提交

1. 运行定向 Vitest、前端完整测试、TypeScript 和生产构建。
2. 运行 `go test ./...` 与 `go vet ./...`。
3. 使用真实 Consumer Group API 检查字段与空状态。
4. 运行 `git diff --check`，确认配置和构建产物仍被忽略。
5. 提交消费组体验重构。
