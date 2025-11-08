#!/bin/bash

# 发布脚本 - 为所有平台构建并打包

set -e

echo "🐱 喵喵学习小助手 - 发布构建脚本"
echo "=================================="

# 版本号
if [ -z "$1" ]; then
    echo "请提供版本号，例如: $0 v1.0.0"
    exit 1
fi

VERSION=$1
echo "构建版本: $VERSION"

# 清理旧的构建产物
echo "🧹 清理旧的构建产物..."
make clean

# 构建所有平台
echo "🔨 构建所有平台版本..."
make build-all

# 创建发布目录
RELEASE_DIR="release-$VERSION"
echo "📦 创建发布目录: $RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# 移动构建产物到发布目录
echo "📁 整理构建产物..."
mv quiz-linux-amd64 "$RELEASE_DIR/"
mv quiz-windows-amd64.exe "$RELEASE_DIR/"
mv quiz-darwin-amd64 "$RELEASE_DIR/"
mv quiz-darwin-arm64 "$RELEASE_DIR/"

# 创建校验和
echo "🔐 生成校验和..."
cd "$RELEASE_DIR"
sha256sum * > checksums.sha256
cd ..

echo "✅ 发布构建完成！"
echo ""
echo "📋 构建产物位于: $RELEASE_DIR/"
echo "📄 文件列表:"
ls -la "$RELEASE_DIR/"
echo ""
echo "🔗 接下来你可以:"
echo "  1. 提交代码: git add . && git commit -m 'Release $VERSION'"
echo "  2. 创建标签: git tag $VERSION"
echo "  3. 推送标签: git push origin $VERSION"
echo "  4. GitHub Actions 会自动创建 Release"