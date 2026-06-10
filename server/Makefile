.PHONY: run build wire tidy lint test clean migrate help

APP_NAME   := kindergarten-server
CMD_DIR    := cmd/api
BIN_DIR    := bin
CONFIG     ?= configs/app.ini

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

run: ## 启动开发服务（热加载需配合 air）
	CONFIG_PATH=$(CONFIG) go run $(CMD_DIR)/main.go

build: ## 编译为二进制文件 → bin/
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)/main.go

wire: ## 重新生成 Wire 依赖注入代码
	cd internal/app && wire

generate: ## 执行 go generate（含 Wire）
	go generate ./internal/app

tidy: ## 整理依赖
	go mod tidy

lint: ## 运行 golangci-lint 静态检查
	golangci-lint run ./...

test: ## 运行全部测试
	go test -v -race -count=1 ./...

test-cover: ## 运行测试并生成覆盖率报告
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

clean: ## 清理构建产物
	rm -rf $(BIN_DIR) coverage.out coverage.html

vet: ## 运行 go vet 检查
	go vet ./...

fmt: ## 格式化代码
	gofmt -w .

check: vet lint test ## 完整检查：vet + lint + test
