# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目定位（Big Picture）

NewAPI Tools 是面向 **NewAPI（One API 分支）** 的增强管理中间件：

- 提供 Web 管理界面（前端：React/Vite/Tailwind）
- 提供后端聚合/管理 API（当前主要实现：`backend/`，Go + Gin + sqlx + Redis）
- 读取 NewAPI 主库（MySQL/Postgres），支持独立日志库，并使用本地 SQLite + Redis 做缓存/状态持久化
- 支持 Docker Compose 一体化部署

仓库里同时存在多套后端实现：
- **`backend/`：当前主要 Go 后端（推荐关注）**
- `backend-py/`：FastAPI 版本（保留/兼容用途）

在修改/排查问题前，优先确认目标是否为当前主后端 `backend/`。

---

## 常用命令

> 优先使用仓库根目录 `Makefile` 做一体化开发；子目录 `backend/Makefile` 和 `frontend/package.json` 提供更细粒度命令。

### 一体化开发（推荐）

```bash
make dev          # 前端 Vite dev server + backend go run
make run          # 构建前端并运行 backend
make build        # 构建前端并编译 backend 二进制
make clean

make frontend-dev
make frontend-build

make docker-build
make docker-run
```

说明：
- 前端 dev server 端口在 `frontend/vite.config.ts` 固定为 **3000**。
- 前端 dev server 会把 `/api` 代理到 `http://localhost:8000`（见 `frontend/vite.config.ts`），所以本地跑后端时通常需要设置：
  - `SERVER_PORT=8000`（后端监听 8000），或
  - 修改 Vite proxy 目标为后端实际端口。

### Go 后端（`backend/`）

```bash
cd backend

make dev          # Air 热重载（使用 .air.toml）
make run          # go run ./cmd/server
make build        # 构建 ../dist/newapi-tools-backend
make build-local  # 构建当前平台二进制

make test
make test-cover

make fmt
make lint
make deps
make deps-tree
```

运行单测（示例）：
```bash
cd backend

go test ./...

go test -run TestDashboard -v ./internal/service
```

### 前端（`frontend/`）

脚本定义见 `frontend/package.json`：

```bash
cd frontend

# 依赖安装：仓库同时存在 bun.lock 与 package-lock.json
# - 根目录 Makefile 使用 bun
# - Dockerfile 中使用 npm ci / npm install

bun run dev
bun run build
bun run lint

# 或（等价）
# npm run dev
# npm run build
# npm run lint
```

### Python 后端（`backend-py/`，如需）

```bash
cd backend-py
uv sync
uv run uvicorn app.main:app --reload --port 8000
uv run pytest
```

### Docker Compose（根目录）

`docker-compose.yml` 提供发布/部署形态（镜像 + Redis）：

```bash
cp .env.example .env
docker compose up -d
docker compose logs -f
docker compose down
```

---

## 代码结构与关键路径（Architecture Map）

### 1) HTTP 入口与路由组织（`backend/`）

- 入口：`backend/cmd/server/main.go`
  - 加载配置：`internal/config.Load()`
  - 初始化：logger / database / redis cache
  - 注册后台任务：`internal/tasks`（任务管理、warmup 状态、周期刷新）
  - 路由集中注册在 `main()` 中，通过各 handler 的 `Register*Routes()` 组织

- 路由层：`backend/internal/handler/*`
  - 各模块 handler 负责参数校验/HTTP 形态，业务逻辑下沉到 service
  - OpenAPI 文件位于 `backend/openapi.json` 和 `backend/internal/handler/openapi.json`

- 业务层：`backend/internal/service/*`
  - 典型模式：`service.NewXxxService()` 创建服务实例
  - 大表/日志相关服务应优先使用 `database.GetLog()`，以兼容独立日志库

- 中间件与认证：`backend/internal/auth`、`backend/internal/middleware`
  - `auth.AuthMiddleware()` 支持 API Key + JWT
  - 全局 CORS、请求日志、错误处理在 `backend/internal/middleware` 中

### 2) 存储与缓存

- 主数据库（NewAPI）：`backend/internal/database` 使用 sqlx 连接 MySQL/Postgres
  - DSN 优先使用 `SQL_DSN`（见 `backend/internal/config/config.go`）

- 日志数据库：支持 `LOG_SQL_DSN`
  - 日志查询应通过 `database.GetLog()`，避免假设 logs 表一定和主库同库

- 本地数据库（SQLite）：`backend/internal/database/local_store.go`
  - 默认位于 data 目录，用于本地配置、缓存、审计等状态

- Redis 与缓存：`backend/internal/cache`
  - 支持 `REDIS_CONN_STRING`
  - 保留基础 Redis 缓存，同时包含从旧实现迁移的高级缓存能力：扩展 Redis 操作、三层缓存管理器、时间槽缓存

### 3) 后台任务系统（`backend/internal/tasks/*`）

- 管理器：`backend/internal/tasks/manager.go`
  - 统一管理任务启动、停止、状态、超时、panic recover 与防重入

- 初始化任务清单：`backend/internal/tasks/init.go`
  - 包括索引检查、IP 记录强制开启、Abuse Broadcast 同步、模型状态轻量刷新、Analytics 预热、AI Ban 只读扫描等

- 预热状态：`backend/internal/tasks/warmup.go`
  - 为 `/api/system/warmup-status` 提供前端兼容的步骤、进度和状态

### 4) 前端构建与“嵌入式”发布形态

- 前端构建产物：`frontend/dist/`
- 根目录 `make embed-frontend` 会把 `frontend/dist/*` 复制到：
  - `backend/frontend/dist/`
- 后端内嵌与静态服务：`backend/frontend/embed.go`
  - `//go:embed all:dist`
  - `/assets/*` 静态资源
  - 其他非 API 路径走 SPA fallback（返回 `index.html`）

### 5) GeoIP（当前行为）

- 查询实现：`backend/internal/service/ip_geo.go`
  - 使用 GeoLite2-City 提供国家、省份、城市信息
  - 增量支持 GeoLite2-ASN、IP version、双栈识别、状态查询和 reload

---

## CI / Docker（与目录结构相关的注意点）

- GitHub Actions：`.github/workflows/build.yml`
  - 触发路径主要是 `backend/**`、`frontend/**` 和根 `Dockerfile`

- 根目录 `Dockerfile` 与 `backend/Dockerfile` 都存在（构建逻辑不同）。
  - 修改发布链路相关代码时，注意同步检查两份 Dockerfile 的入口路径与复制路径是否仍一致。
