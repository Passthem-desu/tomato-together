# TomatoTogether

> 朋友之间的小圈子陪伴式番茄钟应用

## 文档

- [PRD - 产品需求文档](./docs/PRD.md)
- [API - 接口文档](./docs/API.md)

## 技术栈

- **后端**: Go + SQLite + SSE
- **前端**: Svelte / SvelteKit（可选）

## 快速开始

### 🐳 Docker（推荐）

使用预构建的容器镜像快速运行：

```bash
docker run -d \
  --name tomatogether \
  -p 8080:8080 \
  -v tomatogether-data:/app/data \
  ghcr.io/passthem-desu/tomato-together:latest
```

打开浏览器访问 `http://localhost:8080` 即可。

### 📦 Docker Compose

创建 `docker-compose.yml`：

```yaml
services:
  tomatogether:
    image: ghcr.io/passthem-desu/tomato-together:latest
    container_name: tomatogether
    ports:
      - "8080:8080"
    volumes:
      - tomatogether-data:/app/data
    environment:
      - PORT=8080
      - JWT_SECRET=change-me-to-a-random-string
      # 可选配置
      # - SINGLE_ROOM_MODE=false
      # - CORS_ORIGINS=*
      # - JWT_ACCESS_TOKEN_EXPIRY=3600
      # - JWT_REFRESH_TOKEN_EXPIRY=2592000
      # - ROOM_TOKEN_EXPIRY=86400
    restart: unless-stopped

volumes:
  tomatogether-data:
```

启动：

```bash
docker compose up -d
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务监听端口 |
| `JWT_SECRET` | (内置默认) | JWT 签名密钥，**生产环境务必修改** |
| `DB_PATH` | `/app/data/tomatogether.db` | SQLite 数据库路径 |
| `STATIC_DIR` | `/app/static` | 前端静态文件目录 |
| `SINGLE_ROOM_MODE` | `false` | 单房间模式 |
| `CORS_ORIGINS` | `*` | CORS 允许的来源 |
| `JWT_ACCESS_TOKEN_EXPIRY` | `3600` | Access Token 有效期（秒） |
| `JWT_REFRESH_TOKEN_EXPIRY` | `2592000` | Refresh Token 有效期（秒） |
| `ROOM_TOKEN_EXPIRY` | `86400` | Room Token 有效期（秒） |

### 🛠️ 本地开发

后端：

```bash
cd backend
cp ../.env.example .env   # 配置环境变量
make run                   # 编译并运行
```

前端：

```bash
cd frontend
npm install
npm run dev                # 开发服务器（自动代理 /api 到后端）
```

## 许可证

[GNU General Public License v3.0](LICENSE)