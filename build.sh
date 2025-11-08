#!/bin/bash

# 构建脚本 - 替代Makefile的简单版本

set -e

show_help() {
    echo "喵喵学习小助手 - 构建工具"
    echo ""
    echo "用法: $0 [目标]"
    echo ""
    echo "可用目标:"
    echo "  build          - 构建当前平台的版本"
    echo "  build-all      - 为所有平台构建 (Linux, Windows, macOS)"
    echo "  build-linux    - 构建Linux AMD64版本"
    echo "  build-windows  - 构建Windows AMD64版本"
    echo "  build-macos-amd64  - 构建macOS Intel版本"
    echo "  build-macos-arm64  - 构建macOS Apple Silicon版本"
    echo "  clean          - 清理构建产物"
    echo "  test           - 运行测试"
    echo "  run            - 运行应用"
    echo "  help           - 显示此帮助信息"
    echo ""
    echo "使用示例:"
    echo "  ./build.sh build-all    # 构建所有平台的版本"
    echo "  ./build.sh build-linux  # 只构建Linux版本"
    echo "  ./build.sh clean        # 清理构建文件"
}

build_current() {
    echo "🔨 构建当前平台版本..."
    go build -o quiz .
    echo "✅ 构建完成: quiz"
}

build_linux() {
    echo "🐧 构建Linux AMD64版本..."
    GOOS=linux GOARCH=amd64 go build -o quiz-linux-amd64 .
    echo "✅ 构建完成: quiz-linux-amd64"
}

build_windows() {
    echo "🪟 构建Windows AMD64版本..."
    GOOS=windows GOARCH=amd64 go build -o quiz-windows-amd64.exe .
    echo "✅ 构建完成: quiz-windows-amd64.exe"
}

build_macos_amd64() {
    echo "🍎 构建macOS Intel版本..."
    GOOS=darwin GOARCH=amd64 go build -o quiz-darwin-amd64 .
    echo "✅ 构建完成: quiz-darwin-amd64"
}

build_macos_arm64() {
    echo "🍎 构建macOS Apple Silicon版本..."
    GOOS=darwin GOARCH=arm64 go build -o quiz-darwin-arm64 .
    echo "✅ 构建完成: quiz-darwin-arm64"
}

build_all() {
    echo "🔨 构建所有平台版本..."
    build_linux
    build_windows
    build_macos_amd64
    build_macos_arm64
    echo "✅ 所有平台构建完成！"
}

clean() {
    echo "🧹 清理构建产物..."
    rm -f quiz quiz-*.exe quiz-*
    echo "✅ 清理完成"
}

run_tests() {
    echo "🧪 运行测试..."
    go test ./...
    echo "✅ 测试完成"
}

run_app() {
    echo "🚀 启动应用..."
    go run .
}

case "${1:-help}" in
    "build")
        build_current
        ;;
    "build-all")
        build_all
        ;;
    "build-linux")
        build_linux
        ;;
    "build-windows")
        build_windows
        ;;
    "build-macos-amd64")
        build_macos_amd64
        ;;
    "build-macos-arm64")
        build_macos_arm64
        ;;
    "clean")
        clean
        ;;
    "test")
        run_tests
        ;;
    "run")
        run_app
        ;;
    "help"|*)
        show_help
        ;;
esac