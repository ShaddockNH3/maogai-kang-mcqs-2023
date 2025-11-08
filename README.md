# 喵喵学习小助手

一个基于Go和Vue.js的现代化学习辅助工具，支持多种答题模式和错题管理。

## ✨ 特性

- 🐱 萌化的界面设计和交互体验
- 📚 支持多种答题模式（速刷、正式答题、错题回顾）
- 📊 详细的答题统计和进度跟踪
- 💾 本地数据持久化，支持用户数据管理
- 🌐 现代化的Web界面，支持响应式设计
- 🚀 跨平台支持（Windows、Linux、macOS）

## 🚀 快速开始

### 环境要求

- Go 1.21 或更高版本
- 现代浏览器（支持ES6+）

### 安装和运行

1. **克隆项目**
   ```bash
   git clone https://github.com/maogai/maogai.git
   cd maogai
   ```

2. **运行应用**
   ```bash
   # 使用Go直接运行
   go run .

   # 或者使用Makefile
   make run
   ```

3. **打开浏览器**

   应用启动后会自动打开浏览器访问 `http://localhost:8899`

## 🏗️ 构建

### 使用构建脚本

项目提供了便捷的构建脚本（推荐）：

```bash
# 查看所有可用命令
./build.sh help

# 构建当前平台版本
./build.sh build

# 构建所有平台版本
./build.sh build-all

# 构建特定平台版本
./build.sh build-linux      # Linux AMD64
./build.sh build-windows    # Windows AMD64
./build.sh build-macos-amd64  # macOS Intel
./build.sh build-macos-arm64  # macOS Apple Silicon

# 清理构建产物
./build.sh clean

# 运行测试
./build.sh test
```

### 使用Makefile（需要安装make）

```bash
# 构建当前平台版本
make build

# 构建所有平台版本
make build-all

# 其他命令查看帮助
make help
```

### 手动构建

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o quiz-linux-amd64 .

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o quiz-windows-amd64.exe .

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o quiz-darwin-amd64 .

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o quiz-darwin-arm64 .
```

## 📦 发布

### 自动发布（GitHub Actions）

当推送版本标签时，GitHub Actions会自动构建所有平台的版本并创建Release：

1. **创建版本标签**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **自动构建和发布**

   GitHub Actions会自动：
   - 为Linux、Windows、macOS构建可执行文件
   - 创建GitHub Release
   - 上传所有构建产物

### 本地发布

使用发布脚本来构建所有平台版本：

```bash
# 构建并打包所有平台版本
./release.sh v1.0.0

# 这会创建 release-v1.0.0/ 目录，包含：
# - quiz-linux-amd64
# - quiz-windows-amd64.exe
# - quiz-darwin-amd64
# - quiz-darwin-arm64
# - checksums.sha256
```

### 手动发布

如果需要手动创建发布：

1. **构建所有平台版本**
   ```bash
   ./build.sh build-all
   ```

2. **创建压缩包**（可选）
   ```bash
   # Linux
   tar -czf quiz-linux-amd64.tar.gz quiz-linux-amd64

   # Windows
   zip quiz-windows-amd64.zip quiz-windows-amd64.exe

   # macOS
   zip quiz-darwin-amd64.zip quiz-darwin-amd64
   zip quiz-darwin-arm64.zip quiz-darwin-arm64
   ```

3. **上传到GitHub Release**

## 📁 项目结构

```
maogai/
├── .github/
│   └── workflows/
│       └── release.yml          # GitHub Actions 发布工作流
├── clean_outputs/               # 题库数据文件
├── user_data/                   # 用户数据存储目录（运行时生成）
├── .gitignore                   # Git忽略文件
├── build.sh                     # 构建脚本（推荐）
├── Makefile                     # 构建脚本（需要make）
├── release.sh                   # 发布脚本
├── go.mod                       # Go模块文件
├── go.sum                       # Go依赖校验文件
├── main.go                      # 应用程序入口
├── models.go                    # 数据结构定义
├── utils.go                     # 工具函数和API处理器
├── quiz.html                    # 前端界面
└── README.md                    # 项目说明
```

## 🎯 使用说明

### 首次使用

1. 启动应用后，在浏览器中输入用户ID
2. 系统会自动创建用户账户并初始化数据

### 答题模式

- **速刷模式**：快速浏览题目，查看答案
- **答题模式**：正式答题，记录成绩和错题
- **错题回顾**：专项练习错题，提高弱项

### 数据管理

- 用户数据保存在 `user_data/` 目录下
- 支持数据清理和用户切换
- 错题本支持删除和历史记录

## 🛠️ 开发

### 运行测试

```bash
make test
```

### 开发模式

```bash
# 热重载开发
go run .
```

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📞 联系方式

- 项目地址: https://github.com/maogai/maogai
- 问题反馈: [GitHub Issues](https://github.com/maogai/maogai/issues)

---

**🎉 祝学习愉快！**