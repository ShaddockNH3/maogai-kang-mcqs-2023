# Makefile for 喵喵学习小助手

.PHONY: all build clean test help

# 默认目标
all: build

# 构建当前平台的版本
build:
	go build -o quiz .

# 为所有平台构建
build-all: build-linux build-windows build-macos-amd64 build-macos-arm64

# Linux AMD64
build-linux:
	GOOS=linux GOARCH=amd64 go build -o quiz-linux-amd64 .

# Windows AMD64
build-windows:
	GOOS=windows GOARCH=amd64 go build -o quiz-windows-amd64.exe .

# macOS AMD64 (Intel)
build-macos-amd64:
	GOOS=darwin GOARCH=amd64 go build -o quiz-darwin-amd64 .

# macOS ARM64 (Apple Silicon)
build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -o quiz-darwin-arm64 .

# 清理构建产物
clean:
	rm -f quiz quiz-*.exe quiz-*

# 运行测试
test:
	go test ./...

# 运行应用
run:
	go run .

# 显示帮助信息
help:
	@echo "喵喵学习小助手 - 构建工具"
	@echo ""
	@echo "可用目标:"
	@echo "  build          - 构建当前平台的版本"
	@echo "  build-all      - 为所有平台构建 (Linux, Windows, macOS)"
	@echo "  build-linux    - 构建Linux AMD64版本"
	@echo "  build-windows  - 构建Windows AMD64版本"
	@echo "  build-macos-amd64  - 构建macOS Intel版本"
	@echo "  build-macos-arm64  - 构建macOS Apple Silicon版本"
	@echo "  clean          - 清理构建产物"
	@echo "  test           - 运行测试"
	@echo "  run            - 运行应用"
	@echo "  help           - 显示此帮助信息"
	@echo ""
	@echo "使用示例:"
	@echo "  make build-all    # 构建所有平台的版本"
	@echo "  make build-linux  # 只构建Linux版本"
	@echo "  make clean        # 清理构建文件"