# 工作日志 - 2026-05-04

## 摘要
- 完成了 TomatoTogether 项目前后端初步框架的重建
- 后端：Go + SQLite + Gorilla/Mux，实现了核心的房间系统和番茄功能 API
- 前端：SvelteKit 5，使用 runes 模式，实现了房间创建/加入和番茄计时界面
- 前后端均编译通过

## 详细记录

### 后端框架搭建
- 完成时间：11:00-11:30
- 创建了完整的 Go 后端项目结构：
  - `main.go` - 主入口，启动服务器和数据库初始化
  - `internal/models/` - 数据模型（Room、RoomMember、RoomToken、Task、Project 等）
  - `internal/models/api.go` - API 请求/响应结构体
  - `internal/repository/` - 数据库操作层
  - `internal/service/` - 业务逻辑层
  - `internal/api/` - HTTP 处理器

### 核心功能实现
1. **房间系统**
   - 创建房间（房主自动创建）
   - 加入房间（支持密码验证）
   - 离开房间
   - 获取房间信息、用户列表

2. **认证系统**
   - RoomToken 生成和验证（24小时有效期）
   - 匿名用户和持久化用户区分
   - 密码加密（bcrypt）

3. **番茄系统**
   - 开始番茄
   - 跟随番茄
   - 取消跟随
   - 结束番茄
   - 获取番茄状态

4. **状态系统**
   - 更新用户状态（emoji + 消息）
   - 清除状态

### 前端框架搭建
- 完成时间：11:30-12:00
- 创建了 SvelteKit 5 前端项目：
  - `src/lib/api.ts` - API 客户端
  - `src/lib/store.ts` - Svelte store 状态管理
  - `src/app.css` - 全局样式
  - `src/routes/+layout.svelte` - 布局
  - `src/routes/+page.svelte` - 首页（创建/加入房间）
  - `src/routes/room/+page.svelte` - 房间页面（番茄计时）

### 问题解决
1. **Go 重复声明问题**：修复了 `contextKey` 类型重复声明
2. **Svelte 5 runes 模式**：将 `on:click` 改为 `onclick`，`$:` 改为 `$effect`，`let` 改为 `$state()`
3. **缺少 app.html**：创建了 SvelteKit 必需的 HTML 模板

## 修改的文件

### 后端（Go）
- `backend/main.go` - 新增
- `backend/internal/models/models.go` - 新增
- `backend/internal/models/api.go` - 新增
- `backend/internal/repository/repository.go` - 新增
- `backend/internal/service/service.go` - 新增
- `backend/internal/api/handler.go` - 新增

### 前端（SvelteKit）
- `frontend/src/app.html` - 新增
- `frontend/src/app.css` - 新增
- `frontend/src/lib/api.ts` - 新增
- `frontend/src/lib/store.ts` - 新增
- `frontend/src/routes/+layout.svelte` - 新增
- `frontend/src/routes/+page.svelte` - 新增
- `frontend/src/routes/room/+page.svelte` - 新增

## 测试验证
- Go 后端编译：`go build -o server .` ✓ 成功
- 前端编译：`npm run build` ✓ 成功（有 a11y 警告但不影响功能）

## 下一步计划
1. 添加 SSE 实时通信支持
2. 完善番茄跟随功能
3. 添加项目管理和 WIP 功能
4. 完善房主设置界面

## 下午更新 - 2026-05-04

### 改进 #1: 分离事件循环
- 计时器和用户轮询现在使用独立的 setInterval
- 计时器：每秒触发
- 用户轮询：每 5 秒触发

### 改进 #2: 登录流程重构
新增 `/join` 页面，分步骤引导用户：

1. **Step 1 - 房间名**：输入房间名，检查房间是否存在
2. **Step 2 - 房间密码**：如有密码则输入
3. **Step 3 - 用户名**：输入用户名
4. **Step 4 - 用户密码**：
   - 如果用户名已存在且为持久化用户 → 要求输入密码
   - 如果用户名已存在但为匿名用户 → 报错"用户名已被使用"
   - 如果用户名不存在 → 可选择创建持久化账号或匿名加入

### 后端新增接口
- `POST /api/rooms/:name/check-user` - 检查用户名是否存在

### 修改的文件
- `frontend/src/routes/+page.svelte` - 重构为创建房间页面
- `frontend/src/routes/join/+page.svelte` - 新增分步骤加入房间
- `frontend/src/routes/room/+page.svelte` - 分离事件循环
- `backend/internal/api/handler.go` - 新增 CheckUser 处理器
- `backend/internal/service/service.go` - 新增 CheckUsername 方法
- `backend/internal/models/api.go` - 新增 UserCheckResponse
- `TODO.md` - 更新设计决策
