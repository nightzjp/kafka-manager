# 浏览器导航与状态恢复设计

## 目标

把 Kafka Manager 的集群、一级页面、Topic 和 Topic 标签页编码进 URL，使刷新、前进后退、收藏和复制链接都能恢复到同一工作位置。

## URL 结构

- `/clusters/{cluster}/dashboard`
- `/clusters/{cluster}/topics`
- `/clusters/{cluster}/topics/{topic}/{overview|messages|partitions|config}`
- `/clusters/{cluster}/messages`
- `/clusters/{cluster}/consumers`
- `/clusters/{cluster}/settings`
- `/clusters/{cluster}/audit`

路径段使用标准 URL 编码，确保带点号、空格或中文的 Topic 名称可安全往返。未知页面和非法标签页降级到总览或概览，不产生白屏。

## 状态与数据恢复

URL 是导航状态的唯一来源。App 监听 `popstate`，菜单、集群选择器和 Topic 标签切换统一通过 History API 更新 URL。

直接打开 Topic 深链接时，前端先读取集群列表，再通过现有 Topic 搜索接口精确匹配 Topic 名称。恢复期间显示加载态；找不到 Topic 时显示带返回入口的错误，而不是静默跳转。

## 边界

本批次不引入 React Router，避免为当前较小路由表增加依赖。Go 静态资源处理器已经对非资源路径返回 `index.html`，支持 History API 路由。

## 验证

- 路径解析和生成做纯函数单元测试，包括特殊字符与非法路径。
- TypeScript、Vite、Go 全量回归。
- 通过 HTTP 直接请求深层路径，确认服务器返回 SPA 首页。
