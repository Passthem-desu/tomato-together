# 工作日志 - 2026-05-04

## 摘要
- 用户发现现有代码的用户系统设计不符合需求
- 原始设计：全局用户系统，用户名全局唯一
- 期望设计：房间隔离用户系统，每个房间有独立的用户体系
- **决定：推翻重来，重新设计数据库、API、文档**

## 完成的工作

### 1. 重写 PRD.md
- 采用「房间隔离用户」设计理念
- 移除全局用户系统，改用 `room_members` 作为核心用户表
- 更新数据模型：`user_id` → `member_id`
- 更新访问层级说明
- 更新功能模块描述
- 更新开发计划

### 2. 重写 API.md
- **重大变更 v2.0**：房间隔离用户设计
- 新增「设计理念：房间隔离用户」章节
- 重写房间接口：创建房间同时创建房主成员
- 重写成员接口：匿名/持久化用户升级
- 更新所有接口的请求/响应格式
- 更新错误码
- 更新数据库模型速查

### 3. 重写 TODO.md
- 新增「重大变更：房间隔离用户设计」章节
- 更新开发计划任务列表
- 新增 v2.0 设计决策记录（D-001~D-015）
- 新增数据库结构速查
- 标记历史设计决策为已废弃

### 4. 新建数据库迁移
- `000002_room_isolated_users.up.sql` - 新数据库结构
- `000002_room_isolated_users.down.sql` - 回滚脚本
- 核心变更：
  - `room_members` 成为核心用户表（替代 `users`）
  - 所有表的外键从 `user_id` 改为 `member_id`
  - 新增 `room_id` 字段用于房间隔离
  - 移除 `users` 表

## 修改的文件
- `docs/PRD.md` - 完全重写
- `docs/API.md` - 完全重写
- `TODO.md` - 完全重写
- `backend/migrations/000002_room_isolated_users.up.sql` - 新建
- `backend/migrations/000002_room_isolated_users.down.sql` - 新建

## 下一步计划
1. 实现后端代码（models, repositories, services, handlers）
2. 实现前端代码（更新所有用到 user 的地方）
3. 测试新的房间隔离用户流程