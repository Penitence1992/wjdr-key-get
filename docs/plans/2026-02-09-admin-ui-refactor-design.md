# 礼品码管理后台 UI 重构与任务监控刷新设计

日期: 2026-02-09

## 目标
- 统一后台视觉语言, 提升专业感与信息密度
- 任务监控页面支持自动刷新但不闪烁, 不丢输入
- 仅刷新任务列表区域, 保持交互稳定

## 设计系统
- 视觉风格: Data-Dense Dashboard (信息密集型控制台)
- 颜色
  - Primary: #1E40AF
  - Secondary: #3B82F6
  - CTA: #F59E0B
  - Background: #F8FAFC
  - Text: #1E3A8A
- 字体
  - 正文/标题: Fira Sans
  - 数据/ID: Fira Code
- 关键规则
  - 交互 150-250ms 过渡, 支持 prefers-reduced-motion
  - 明显 focus ring, 满足可访问性对比
  - 可点击元素统一 cursor-pointer

## 页面结构
- 顶栏: 标题 + 用户信息 + 退出
- 侧边栏: 选中态 + 左色条 + 背景强调
- 内容区: 标题 + 操作区 + 数据区
- 任务监控页拆分
  - 添加兑换码卡片
  - 任务控制条 (自动刷新/历史)
  - 任务列表容器 (可局部刷新)

## 交互与数据刷新
- 采用“静态壳 + 局部刷新”
- 首次进入
  - renderTasksShell() 渲染静态结构
  - refreshTasksTable() 请求并填充表格
- 自动刷新
  - 定时器只调用 refreshTasksTable()
  - 不影响表单输入, 不触发布局重绘
- 切换视图时停止定时器

## 组件拆分
- renderUsersShell() + refreshUsersTable()
- renderTasksShell() + refreshTasksTable()
- renderNotificationsShell() + refreshNotificationsTable()
- renderUserDetailsShell(fid) + refreshUserDetailsTable(fid)

## 错误处理
- API 异常提示 message, 保留现有内容
- 网络错误不清空表格
- 空列表显示 empty-state

## 验证清单
- 任务监控输入框在自动刷新时不丢值
- 刷新只更新任务表格区域, 无闪缩
- 切换视图后定时器停止
- 375/768/1024/1440 无横向滚动
- 表单与按钮可见 focus 状态
