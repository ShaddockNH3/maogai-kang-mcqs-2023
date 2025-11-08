package main

import (
	"context"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// openBrowser 在服务器启动后打开浏览器
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		// 检查是否在WSL环境中
		if isWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			cmd = "xdg-open"
			args = []string{url}
		}
	}

	log.Printf("正在尝试打开浏览器: %s %v", cmd, args)
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		log.Printf("打开浏览器失败: %v", err)
	}
}

// isWSL 检查是否在WSL环境中运行
func isWSL() bool {
	// 检查/proc/version文件是否包含WSL
	if data, err := exec.Command("cat", "/proc/version").Output(); err == nil {
		return strings.Contains(string(data), "WSL")
	}
	return false
}

// main函数，程序入口
func main() {
	// 使用默认配置初始化 Hertz 服务器，监听在 0.0.0.0:8899
	h := server.Default(server.WithHostPorts("0.0.0.0:8899"))

	// 提供静态文件服务，路径为当前项目文件夹
	h.Static("/", "./") // 将根路径映射到当前文件夹

	// 根路径重定向到quiz.html
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.File("./quiz.html")
	})

	// API 路由组
	apiGroup := h.Group("/api")
	{
		sessionGroup := apiGroup.Group("/session")
		{
			// POST /api/session/init - 初始化用户会话 (现在需要 userID)
			sessionGroup.POST("/init", InitSessionHandler)
		}

		reviewGroup := apiGroup.Group("/review") // 速刷模式相关
		{
			// POST /api/review/start - 开始速刷
			reviewGroup.POST("/start", QuickReviewStartHandler)
			// POST /api/review/next - 获取速刷下一题 (可能被前端优化掉)
			reviewGroup.POST("/next", GetNextQuestionHandler)
		}

		quizGroup := apiGroup.Group("/quiz") // 答题模式相关
		{
			// POST /api/quiz/start - 开始答题
			quizGroup.POST("/start", QuizStartHandler)
			// POST /api/quiz/submit_answer - 提交答案
			quizGroup.POST("/submit_answer", SubmitAnswerHandler)
		}

		incorrectGroup := apiGroup.Group("/incorrect_questions") // 错题回顾相关
		{
			// POST /api/incorrect_questions/review/start - 开始错题回顾
			incorrectGroup.POST("/review/start", IncorrectQuestionsReviewStartHandler)
			// POST /api/incorrect_questions/review/submit_answer - 提交错题回顾中的答案
			incorrectGroup.POST("/review/submit_answer", SubmitIncorrectReviewAnswerHandler)
			// POST /api/incorrect_questions/delete - 从错题本中删除一题
			incorrectGroup.POST("/delete", DeleteIncorrectQuestionHandler)
		}

		userGroup := apiGroup.Group("/user") // 用户数据管理
		{
			// POST /api/user/data/clear - 清理用户数据
			userGroup.POST("/data/clear", UserDataClearHandler)
		}
	}

	log.Println("喵喵学习小助手 Go 后端已启动，监听于 http://0.0.0.0:8899")

	// 启动goroutine在服务器启动后打开浏览器
	go func() {
		time.Sleep(1 * time.Second) // 等待服务器启动
		openBrowser("http://localhost:8899")
	}()

	h.Spin() // 启动服务器并开始监听请求
}
