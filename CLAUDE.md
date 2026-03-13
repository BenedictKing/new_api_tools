# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目定位（Big Picture）

NewAPI Tools 是面向 **NewAPI（One API 分支）** 的增强管理中间件：

- 提供 Web 管理界面（前端：React/Vite/Tailwind）
- 提供后端聚合/管理 API（主要实现：`backend-go`，Gin + GORM + Redis）
- 读取 NewAPI 主库（MySQL/Postgres），并使用本地 SQLite + Redis 做缓存/状态持久化
- 支持 Docker Compose 一体化部署

仓库里同时存在多套后端实现：
- **`backend-go/`：当前主要 Go 后端（推荐关注）**
- `backend-py/`：FastAPI 版本（保留/兼容用途）
- `backend/`：另一个 Go module（CI 里也有引用；与 `backend-go` 并存）

在修改/排查问题前，先确认你要改的是哪套后端（`backend-go` vs `backend` vs `backend-py`）。

---

## 常用命令

> 优先使用仓库根目录 `Makefile` 做一体化开发；子目录 `backend-go/Makefile` 和 `frontend/package.json` 提供更细粒度命令。

### 一体化开发（推荐）

```bash
make dev          # 前端 Vite dev server + backend-go Air 热重载
make run          # 构建前端并运行 backend-go
make build        # 构建前端并编译 backend-go 二进制
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

### Go 后端（`backend-go/`）

```bash
cd backend-go

make dev          # Air 热重载（使用 .air.toml）
make run          # go run（会尝试复制前端产物到 backend-go/frontend/dist）
make build        # 构建 ../dist/newapi-tools
make build-local  # 构建当前平台二进制到 ./newapi-tools

make test
make test-cover

make fmt
make lint
```

运行单测（示例）：
```bash
cd backend-go

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

### 1) HTTP 入口与路由组织（`backend-go/`）

- 入口：`backend-go/main.go`
  - 加载配置：`internal/config.Load()`
  - 初始化：logger / database / redis cache / jwt / geoip
  - 启动后台任务：`internal/tasks.InitTasks()` + `TaskManager.Start()`
  - 路由集中注册在 `setupRouter()`（同文件）

- 路由层：`backend-go/internal/handler/*`
  - `internal/handler/common.go` 定义统一响应结构与 `Success()/Error()` 等工具函数
  - 各模块 handler 负责参数校验/HTTP 形态，业务逻辑下沉到 service

- 业务层：`backend-go/internal/service/*`
  - 典型模式：`service.NewXxxService()` -> handler 持有单例 `var xxxService = ...`

- 中间件：`backend-go/internal/middleware/auth.go`
  - JWT 鉴权：`AuthMiddleware()`
  - 可选 API Key：`APIKeyMiddleware()`

### 2) 存储与缓存

- 主数据库（NewAPI）：`internal/database` 使用 GORM 连接 MySQL/Postgres
  - DSN 优先使用 `SQL_DSN`（见 `internal/config/config.go`）

- 本地数据库（SQLite）：`internal/database` 会初始化一个独立 SQLite 文件
  - 默认路径：`./data/local.db`（可通过 `DATABASE_LOCAL_DB_PATH` / 配置覆盖）
  - 用途：缓存、分析状态、审计/配置等本地持久化（见 `internal/database/database.go` 的表创建逻辑）

- Redis：`internal/cache`
  - 支持 `REDIS_CONN_STRING` 或分离配置
  - 预热阶段会尝试把 SQLite 缓存恢复到 Redis（见 `internal/tasks/warmup.go`）

### 3) 后台任务系统（`backend-go/internal/tasks/*`）

- 管理器：`internal/tasks/manager.go`
  - 支持两类任务：启动即跑（`Register`）与预热完成后启动（`StartAfterWarmup`）

- 初始化任务清单：`internal/tasks/init.go`
  - 立即任务：缓存预热、索引检查、IP 记录强制开启、GeoIP 更新、过期缓存清理
  - 预热后任务：缓存刷新、日志同步、AI 封禁扫描、模型状态刷新

- 预热流程：`internal/tasks/warmup.go`
  - 多阶段渐进式 warmup，并通过 `SignalWarmupDone()` 解锁后续任务

### 4) 前端构建与“嵌入式”发布形态

- 前端构建产物：`frontend/dist/`
- 根目录 `make embed-frontend` 会把 `frontend/dist/*` 复制到：
  - `backend-go/frontend/dist/`
- 后端内嵌与静态服务：`backend-go/frontend/embed.go`
  - `//go:embed all:dist`
  - `/assets/*` 静态资源
  - 其他非 API 路径走 SPA fallback（返回 `index.html`）

### 5) GeoIP（当前行为）

- 配置默认值见 `backend-go/internal/config/config.go`
  - `GEOIP_DB_PATH` 默认 `/app/data/geoip`
  - `GEOIP_UPDATE_URL` 默认下载 GeoLite2-Country.mmdb
- 查询实现：`backend-go/pkg/geoip`
  - 当前仅使用 **Country + ASN**（不包含 City 信息）

---

## CI / Docker（与目录结构相关的注意点）

- GitHub Actions：`.github/workflows/build.yml`
  - 触发路径主要是 `backend/**`、`frontend/**` 和根 `Dockerfile`
  - 若你改动集中在 `backend-go/**`，CI 是否会触发需要额外确认/调整 workflow 的 paths

- 根目录 `Dockerfile` 与 `backend-go/Dockerfile` 都存在（构建逻辑不同）。
  - 修改发布链路相关代码时，注意同步检查两份 Dockerfile 的入口路径与复制路径是否仍一致。
