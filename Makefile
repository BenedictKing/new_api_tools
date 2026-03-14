# NewAPI Tools Makefile

GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: help dev run build clean frontend-dev frontend-build embed-frontend generate-api-types

help:
	@echo "$(GREEN)NewAPI Tools - 可用命令:$(NC)"
	@echo ""
	@echo "$(YELLOW)开发:$(NC)"
	@echo "  make dev            - Go 后端热重载开发(不含前端)"
	@echo "  make run            - 构建前端并运行 Go 后端"
	@echo "  make frontend-dev   - 前端开发服务器"
	@echo ""
	@echo "$(YELLOW)构建:$(NC)"
	@echo "  make build          - 构建前端并编译 Go 后端"
	@echo "  make frontend-build - 仅构建前端"
	@echo "  make clean          - 清理构建文件"
	@echo ""
	@echo "$(YELLOW)Docker:$(NC)"
	@echo "  make docker-build   - 构建 Docker 镜像"
	@echo "  make docker-run     - 运行 Docker 容器"
	@echo "  make generate-api-types - 从后端 swagger 生成前端 TS 类型"

dev:
	@echo "$(GREEN)🚀 启动前后端开发模式...$(NC)"
	@cd frontend && bun run dev &
	@cd backend-go && $(MAKE) dev

run: embed-frontend
	@cd backend-go && $(MAKE) run

build: embed-frontend
	@cd backend-go && $(MAKE) build

embed-frontend:
	@echo "$(GREEN)📦 构建前端...$(NC)"
	@cd frontend && bun run build
	@echo "$(GREEN)📋 嵌入前端到 Go 后端...$(NC)"
	@rm -rf backend-go/frontend/dist
	@mkdir -p backend-go/frontend/dist
	@cp -r frontend/dist/* backend-go/frontend/dist/

clean:
	@cd backend-go && $(MAKE) clean
	@rm -rf frontend/dist

frontend-dev:
	@cd frontend && bun run dev

frontend-build:
	@cd frontend && bun run build

docker-build:
	@echo "$(GREEN)🐳 构建 Docker 镜像...$(NC)"
	@docker build -t newapi-tools:latest .

docker-run:
	@echo "$(GREEN)🐳 运行 Docker 容器...$(NC)"
	@docker run -d --name newapi-tools -p 3000:3000 newapi-tools:latest

generate-api-types: ## 从后端 swagger 生成前端 TypeScript 类型
	@echo "$(GREEN)📐 生成后端 Swagger 文档...$(NC)"
	@cd backend-go && swag init -g main.go -o docs --parseInternal
	@echo "$(GREEN)🔄 生成前端 TypeScript 类型...$(NC)"
	@cd frontend && bun run generate:api
	@echo "$(GREEN)✅ API types regenerated$(NC)"
