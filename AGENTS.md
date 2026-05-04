# Agent 工作指南

> 本文件用于指导后续 Agent 如何理解和参与 TomatoTogether 项目的开发工作。

---

## 📌 核心文档

| 文档 | 用途 |
|------|------|
| `TODO.md` | 开发任务清单，**请在开始任何工作前先阅读** |
| `README.md` | 项目概览和快速开始 |
| `docs/PRD.md` | 产品需求文档，包含完整的产品设计 |
| `docs/API.md` | API 接口文档，包含请求/响应格式 |
| `AGENTS.md` | 本文件，Agent 工作指南 |

---

## 📋 TODO.md 格式说明

`TODO.md` 使用 **LogSeq/列表式** 格式组织任务：

```
## 📋 分类标题

### 子分类
- [ ] 未完成的任务
- [x] 已完成的任务（使用 `[x]` 表示）

## 📝 设计决策记录
表格形式记录已确定的设计方案
```

**命名规范**：
- 设计决策使用 `D-XXX` 前缀（如 D-001, D-002）
- 任务项使用 `- [ ]` 表示未完成，`- [x]` 表示完成
- 保持 `D-XXX` 编号连续，**不要复用已用的编号**

---

## 🗺️ 开发阶段

项目分为 5 个主要阶段，详见 `docs/PRD.md` 第 7 节：

| 阶段 | 目标 | 主要功能 |
|------|------|----------|
| Phase 1 | 核心 MVP | 房间 + 基本番茄功能 |
| Phase 2 | WIP 待办 | 本地存储 + 项目管理 |
| Phase 3 | 账号与同步 | 云同步 + 账号系统 |
| Phase 4 | 统计 | 数据可视化 |
| Phase 5 | 细节打磨 | PWA + 主题 + 文档 |

**建议**：按阶段顺序开发，先完成 Phase 1 再进入 Phase 2。

---

## 🔑 关键设计决策

在开始实现前，请熟悉以下设计决策（详见 `TODO.md` 设计决策记录）：

### 认证层级
- **L1 公开**：无需认证
- **L2 房间成员**：使用 `room_token`（24 小时有效）
- **L3 持久化用户**：使用 JWT access_token（1 小时）+ refresh_token（30 天）

### 房间模型
- 房间名全局唯一
- 房主必须是 opt-in 用户（有密码的持久化用户）
- 支持旁观者模式（L1，可看不可操作）

### 番茄机制
- 客户端本地计时，服务端记录 `started_at`
- 使用 SSE tick 同步状态（5 秒一次）
- 跟随者与主导者同步番茄

### WIP 云同步
- 使用 `client_id`（客户端 UUID）映射 `server_id`（服务端 UUID）
- 同步冲突以 `created_at` 最早的服务端记录为准

### Heartbeat 机制
- SSE 客户端每 30 秒发送 ping
- 服务端 2 分钟无响应视为离线

---

## 📦 项目结构

```
tomatogether/
├── backend/                  # Go 后端
│   ├── cmd/server/          # 主入口
│   ├── internal/
│   │   ├── api/             # API 处理器
│   │   ├── models/          # 数据模型
│   │   ├── repository/      # 数据库操作
│   │   ├── service/         # 业务逻辑
│   │   └── sse/             # SSE 事件推送
│   ├── migrations/          # 数据库迁移（SQL 文件）
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   ├── go.mod
│   └── main.go
├── frontend/                 # SvelteKit 前端
│   ├── src/
│   │   ├── lib/             # 组件、工具
│   │   │   ├── i18n/        # 国际化 (i18n)
│   │   │   │   ├── locales.ts
│   │   │   │   ├── store.ts
│   │   │   │   └── index.ts
│   │   │   ├── api.ts       # API 客户端
│   │   │   └── store.ts     # Svelte stores
│   │   └── routes/          # 页面
│   ├── package.json
│   └── ...
├── docs/                     # 文档
│   ├── PRD.md
│   └── API.md
├── .env.example              # 环境变量示例
├── TODO.md                   # 任务清单
├── AGENTS.md                 # 本文件
└── README.md
```

---

## 🛠 工作流程

### 开始新任务
1. 阅读 `TODO.md` 了解当前进度
2. 找到对应的未完成任务
3. 在 PRD.md 或 API.md 中确认相关设计
4. 开始实现
5. 完成后更新 `TODO.md`（将 `- [ ]` 改为 `- [x]`）

### 新增任务
1. 在 `TODO.md` 对应分类下添加任务（使用 `- [ ]`）
2. 如果涉及设计决策，添加到「设计决策记录」表格
3. 更新「最后更新」时间

### 更新文档
1. 先备份原内容
2. 修改 PRD.md 或 API.md
3. 在 `TODO.md` 中标记对应任务为完成

---

## 🧪 测试规范

### 后端测试服务器
- 如果需要启动后端测试服务器，**不要占用默认的 8080 端口**
- 使用其他可用端口（如 8081、8082 等），避免影响用户正在运行的测试服务器
- 测试完成后**及时关闭测试服务器**

### 测试命令示例
```bash
# 查看 8080 端口是否被占用
lsof -i :8080

# 如果 8080 被占用，使用其他端口
PORT=8081 go run main.go

# 或编译后运行
go build -o server_test .
PORT=8081 ./server_test
```

### 前端测试
- 测试前确保后端服务器正常运行
- 如需修改 API 代理配置，检查 `vite.config.ts`
- 编译前清理缓存：`rm -rf .svelte-kit && npm run build`

---

## ⚠️ 注意事项

1. **不要删除已完成的任务标记**，只改为 `- [x]`
2. **新增设计决策时使用下一个可用编号**
3. **实现前先确认 API 签名**，避免返工
4. **SQL 文件命名规范**：`{version}_{description}.up.sql` / `{version}_{description}.down.sql`
5. **Go 代码规范**：遵循标准 Go 项目布局，使用 `internal/` 分离内部包

---

## 📝 工作日志规范

为方便后续 Agent 和实习生查阅开发进展，每个 Agent 在完成工作时应记录工作日志。

### 日志文件位置
- 工作日志存放在 `logs/` 目录下
- 文件命名格式：`YYYY-MM-DD_agentname_worklog.md`
- 示例：`logs/2026-05-04_pico_worklog.md`

### 日志格式

```markdown
# 工作日志 - YYYY-MM-DD

## 摘要
- 完成的主要工作
- 遇到的问题及解决方案
- 下一步计划

## 详细记录

### 任务 1: xxx
- 完成时间：hh:mm-hh:mm
- 完成内容：
  - 具体做了哪些事情
  - 修改了哪些文件
  - 关键代码片段

### 任务 2: xxx
...

## 修改的文件
- `backend/internal/xxx/yyy.go` - 新增/修改说明
- `frontend/src/xxx/yyy.svelte` - 新增/修改说明

## 遇到的问题
1. 问题描述 → 解决方案

## 测试验证
- 单元测试：xxx
- API 测试：xxx
```

### 更新规则
1. 完成任何有意义的开发工作后，**必须更新工作日志**
2. 日志应包含足够细节，方便他人复现和理解
3. 如果日志已存在，在末尾追加新内容（不要覆盖）

---

## 🏗️ 后端构建规范

后端使用 Makefile 管理构建，请使用以下命令：

### 构建命令
```bash
cd backend
make build      # 编译到 bin/tomatogether
make run        # 编译并运行
make clean      # 清理构建产物
make test       # 运行测试
make dev        # 开发模式构建（详细输出）
make help       # 显示帮助信息
```

### 构建产物位置
- **二进制文件**：`backend/bin/tomatogether`
- **构建目录**：`backend/bin/`（已加入 `.gitignore`）
- **日志文件**：`backend/server.log`（运行时生成）

### 注意事项
- 构建产物统一放在 `bin/` 目录，不要直接放在 `backend/` 根目录
- 测试时使用 `PORT=8081 ./bin/tomatogether` 避免端口冲突
- 启动时会自动执行数据库迁移，无需手动运行

---

## 🌐 国际化 (i18n) 指南

### 语言代码
- `zh-hans` - 简体中文
- `zh-hant` - 繁體中文
- `en` - English
- `ja` - 日本語

### 文件结构
```
frontend/src/lib/i18n/
├── locales.ts    # 语言配置（语言列表、locale 存储）
├── store.ts     # locale 状态管理
└── index.ts     # 翻译文本和工具函数
```

### 添加翻译
1. 在 `index.ts` 中为每种语言添加翻译键值对
2. 翻译键使用小写下划线格式（如 `room_name`、`error_invalid_password`）

### 在组件中使用
```svelte
<script>
  import { locale, t } from '$lib/i18n';
  
  // 翻译文本
  const greeting = t('hello', $locale);
  
  // 或直接在模板中
  // {t('room_name', $locale)}
</script>
```

### 在 store 中使用（错误处理）
```typescript
import { getErrorMessage, locale } from './i18n';
import { get } from 'svelte/store';

error.set(getErrorMessage(e, get(locale)));
```

### 注意事项
- 所有用户可见文本都必须使用 `t()` 函数翻译
- 错误信息需要通过 `getErrorMessage()` 转换
- 确认对话框等 JavaScript 原生文本也需要国际化

---

*最后更新：2026-05-04*