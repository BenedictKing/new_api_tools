# NewAPI Tools Makefile

GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: help dev run build clean frontend-dev frontend-build embed-frontend sync-openapi generate-api-types docker-build docker-run

help:
	@echo "$(GREEN)NewAPI Tools - 可用命令:$(NC)"
	@echo ""
	@echo "$(YELLOW)开发:$(NC)"
	@echo "  make dev            - 前端 Vite dev server + backend 后端"
	@echo "  make run            - 构建前端并运行 backend 后端"
	@echo "  make frontend-dev   - 前端开发服务器"
	@echo ""
	@echo "$(YELLOW)构建:$(NC)"
	@echo "  make build          - 构建前端并编译 backend 二进制"
	@echo "  make frontend-build - 仅构建前端"
	@echo "  make clean          - 清理构建文件"
	@echo ""
	@echo "$(YELLOW)Docker:$(NC)"
	@echo "  make docker-build   - 构建 Docker 镜像"
	@echo "  make docker-run     - 运行 Docker 容器"
	@echo "  make generate-api-types - 从后端 OpenAPI 生成前端 TS 类型"

dev: sync-openapi
	@echo "$(GREEN)🚀 启动前后端开发模式...$(NC)"
	@cd frontend && bun run dev &
	@cd backend && go run ./cmd/server

run: sync-openapi embed-frontend
	@cd backend && go run ./cmd/server

build: sync-openapi embed-frontend
	@echo "$(GREEN)🔨 构建 backend 模块...$(NC)"
	@mkdir -p dist
	@cd backend && go build -o ../dist/newapi-tools ./cmd/server

embed-frontend:
	@echo "$(GREEN)📦 构建前端...$(NC)"
	@cd frontend && bun run build
	@echo "$(GREEN)📋 嵌入前端到 backend 模块...$(NC)"
	@rm -rf backend/frontend/dist
	@mkdir -p backend/frontend/dist
	@cp -r frontend/dist/* backend/frontend/dist/

sync-openapi:
	@cp backend/openapi.json backend/internal/handler/openapi.json

clean:
	@cd backend && $(MAKE) clean
	@rm -rf frontend/dist
	@rm -rf backend/frontend/dist

frontend-dev:
	@cd frontend && bun run dev

frontend-build:
	@cd frontend && bun run build

docker-build:
	@echo "$(GREEN)🐳 构建 Docker 镜像...$(NC)"
	@docker build -t newapi-tools:latest .

docker-run:
	@echo "$(GREEN)🐳 运行 Docker 容器...$(NC)"
	@docker run -d --name newapi-tools -p 1145:8000 -e SERVER_HOST=0.0.0.0 newapi-tools:latest

generate-api-types: sync-openapi ## 从后端 OpenAPI 生成前端 TypeScript 类型
	@echo "$(GREEN)🔄 生成前端 TypeScript 类型...$(NC)"
	@cd frontend && bun run generate:api
	@echo "$(GREEN)✅ API types regenerated$(NC)"
