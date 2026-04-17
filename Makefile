.PHONY: build run clean fmt vet tidy test lint docker

APP_NAME   := gonio
BUILD_DIR  := bin
MAIN_PATH  := cmd/server/main.go

# 构建
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# 运行
run:
	go run $(MAIN_PATH)

# 清理
clean:
	rm -rf $(BUILD_DIR)/

# 代码格式化
fmt:
	go fmt ./...

# 代码检查
vet:
	go vet ./...

# 下载依赖
tidy:
	go mod tidy

# 运行测试
test:
	go test -v -race -count=1 ./...

# 运行测试并生成覆盖率报告
test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 静态检查（需要安装 golangci-lint）
lint:
	golangci-lint run ./...

# Docker 构建
docker:
	docker build -t $(APP_NAME):latest .

# 一键检查：格式化 + 静态分析 + 测试
check: fmt vet lint test
