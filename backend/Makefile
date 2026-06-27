# Makefile for NewAPI Tools backend module

BINARY_NAME=newapi-tools-backend
MAIN_PATH=./cmd/server
BUILD_DIR=../dist

GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: help dev run build build-local clean install-air test test-cover fmt lint deps deps-tree install-tools

help:
	@echo "$(GREEN)NewAPI Tools Backend - 可用命令:$(NC)"
	@echo ""
	@echo "$(YELLOW)开发命令:$(NC)"
	@echo "  make dev          - 启动 Air 热重载开发"
	@echo "  make run          - 直接运行后端"
	@echo "  make install-air  - 安装 Air 热重载工具"
	@echo ""
	@echo "$(YELLOW)构建命令:$(NC)"
	@echo "  make build        - 构建生产版本到 ../dist"
	@echo "  make build-local  - 构建当前平台二进制"
	@echo "  make clean        - 清理构建文件"
	@echo ""
	@echo "$(YELLOW)质量命令:$(NC)"
	@echo "  make test         - 运行测试"
	@echo "  make test-cover   - 运行测试并生成覆盖率"
	@echo "  make fmt          - 格式化代码"
	@echo "  make lint         - 运行 golangci-lint"
	@echo "  make deps         - 整理并下载依赖"
	@echo "  make deps-tree    - 查看依赖树"

install-air:
	@echo "$(GREEN)正在安装 Air 热重载工具...$(NC)"
	@go install github.com/air-verse/air@latest
	@echo "$(GREEN)✅ Air 安装完成$(NC)"

dev:
	@if ! command -v air &> /dev/null; then \
		echo "$(YELLOW)⚠️  Air 未安装，正在自动安装...$(NC)"; \
		$(MAKE) install-air; \
	fi
	@echo "$(GREEN)🚀 启动后端热重载开发模式$(NC)"
	@air

run:
	@echo "$(GREEN)▶️  直接运行后端...$(NC)"
	@go run $(MAIN_PATH)

build:
	@echo "$(GREEN)📦 构建生产版本...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✅ 构建完成: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-local:
	@echo "$(GREEN)📦 构建本地版本...$(NC)"
	@go build -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✅ 构建完成: ./$(BINARY_NAME)$(NC)"

test:
	@echo "$(GREEN)🧪 运行测试...$(NC)"
	@go test -v ./...

test-cover:
	@echo "$(GREEN)🧪 运行测试并生成覆盖率...$(NC)"
	@go test -v -cover -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ 覆盖率报告已生成: coverage.html$(NC)"

fmt:
	@echo "$(GREEN)🎨 格式化代码...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ 格式化完成$(NC)"

lint:
	@echo "$(GREEN)🔍 检查代码...$(NC)"
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "$(YELLOW)⚠️  golangci-lint 未安装，请运行 make install-tools$(NC)"; \
		exit 1; \
	fi
	@golangci-lint run

deps:
	@echo "$(GREEN)📚 更新依赖...$(NC)"
	@go mod tidy
	@go mod download
	@echo "$(GREEN)✅ 依赖更新完成$(NC)"

deps-tree:
	@echo "$(GREEN)🌳 依赖树:$(NC)"
	@go mod graph

install-tools: install-air
	@echo "$(GREEN)🔧 安装开发工具...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✅ 所有工具安装完成$(NC)"

clean:
	@echo "$(YELLOW)🧹 清理构建文件...$(NC)"
	@rm -rf tmp/
	@rm -f $(BINARY_NAME)
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -f coverage.out coverage.html build-errors.log
	@echo "$(GREEN)✅ 清理完成$(NC)"
