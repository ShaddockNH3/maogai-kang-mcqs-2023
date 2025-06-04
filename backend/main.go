package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil" // For Go versions < 1.16, for >= 1.16 os.ReadFile is preferred for simple reads
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"
)

// Question 结构定义了从 JSON 文件中读取的单个题目信息
type Question struct {
	QuestionNumber string            `json:"question_number"` // 原始题库中的题号
	QuestionType   string            `json:"question_type"`   // 题目类型 (例如: "单选题", "多选题")
	QuestionText   string            `json:"question_text"`   // 题目文本
	Options        map[string]string `json:"options"`         // 选项 (例如: {"A": "选项A", "B": "选项B"})
	CorrectAnswer  string            `json:"correct_answer"`  // 正确答案 (例如: "A", "ABC")
}

// QuestionOutput 是发送给前端的题目数据格式
type QuestionOutput struct {
	DisplayQuestionNumber string            `json:"display_question_number"` // 题目在当前测验中的序号 (例如 "1", "2", ..., "50")
	OriginalQuestionID    string            `json:"original_question_id"`  // 题目在原始题库中的ID
	QuestionType          string            `json:"question_type"`
	QuestionText          string            `json:"question_text"`
	Options               map[string]string `json:"options"`
	CorrectAnswer         string            `json:"correct_answer"` // 将正确答案发送给前端，由前端进行判断
}

// UserSession 存储单个用户答题会话的状态
type UserSession struct {
	UserID               string
	SelectedQuestions    []QuestionOutput // 为该用户选择的题目列表
	CurrentQuestionIndex int              // 当前题目在 SelectedQuestions 中的索引 (0-based)
	WronglyAnswered      []QuestionOutput // 用户答错的题目列表
	mu                   sync.Mutex       // 用于保护此会话数据的互斥锁
}

var (
	allQuestions []Question // 存储从 JSON 文件加载的所有题目
	userSessions = make(map[string]*UserSession) // 存储活跃用户会话的映射
	sessionsMu   sync.Mutex     // 用于保护 userSessions 映射的互斥锁
	questionPoolSize = 10       // 每个答题会话的题目数量
)

// loadQuestions 从指定的 JSON 文件路径加载题目数据
func loadQuestions(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开题目文件 '%s': %w", filePath, err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("无法读取题目文件 '%s': %w", filePath, err)
	}

	err = json.Unmarshal(bytes, &allQuestions)
	if err != nil {
		return fmt.Errorf("无法解析来自 '%s' 的题目 JSON 数据: %w", filePath, err)
	}

	if len(allQuestions) == 0 {
		return fmt.Errorf("未从文件 '%s' 加载任何题目", filePath)
	}
	log.Printf("成功从 %s 加载 %d 道题目", filePath, len(allQuestions))
	return nil
}

// selectRandomQuestions 从 allQuestions 中随机选择指定数量的不重复题目
func selectRandomQuestions() []QuestionOutput {
	n := len(allQuestions)
	if n == 0 {
		return []QuestionOutput{}
	}

	numToSelect := questionPoolSize
	if n < numToSelect {
		numToSelect = n
		log.Printf("警告: 可用题目数量 (%d) 少于期望的题目池大小 (%d)。将使用所有可用题目。", n, questionPoolSize)
	}
	
	perm := rand.Perm(n) 
	
	selected := make([]QuestionOutput, 0, numToSelect)
	for i := 0; i < numToSelect; i++ {
		q := allQuestions[perm[i]]
		selected = append(selected, QuestionOutput{
			DisplayQuestionNumber: strconv.Itoa(i + 1), // 当前测验中的序号 (1-based)
			OriginalQuestionID:    q.QuestionNumber,    // 原始题库中的题号
			QuestionType:          q.QuestionType,
			QuestionText:          q.QuestionText,
			Options:               q.Options,
			CorrectAnswer:         q.CorrectAnswer,
		})
	}
	return selected
}

// StartQuizResponse 是 /quiz/start 端点的响应结构
type StartQuizResponse struct {
	UserID         string         `json:"user_id"`          // 用户会话ID
	Question       QuestionOutput `json:"question"`         // 第一道题目
	TotalQuestions int            `json:"total_questions"`  // 本次答题的总题目数
}

// startQuizHandler 处理开始新答题的请求
func startQuizHandler(ctx context.Context, c *app.RequestContext) {
	userID := uuid.NewString() 
	
	selectedQs := selectRandomQuestions()
	if len(selectedQs) == 0 {
		log.Println("错误: 无法开始答题，因为没有可用的题目。")
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "没有可用的题目来开始答题。"})
		return
	}

	session := &UserSession{
		UserID:               userID,
		SelectedQuestions:    selectedQs,
		CurrentQuestionIndex: 0, // 从第一道题开始 (0-based index)
		WronglyAnswered:      make([]QuestionOutput, 0),
	}

	sessionsMu.Lock()
	userSessions[userID] = session
	sessionsMu.Unlock()

	firstQuestion := session.SelectedQuestions[0] // 获取第一道题 (index 0)
	log.Printf("用户 %s 开始答题，共 %d 道题。第一题显示序号: %s", userID, len(selectedQs), firstQuestion.DisplayQuestionNumber)

	c.JSON(consts.StatusOK, StartQuizResponse{
		UserID:         userID,
		Question:       firstQuestion,
		TotalQuestions: len(selectedQs),
	})
}

// AnswerRequest 定义了提交答案请求的结构
type AnswerRequest struct {
	UserID                string `json:"user_id" vd:"required"`
	DisplayQuestionNumber string `json:"display_question_number" vd:"required"` // 前端提交的当前题目在测验中的显示序号
	WasCorrect            bool   `json:"was_correct"`
}

// AnswerResponse 是提交答案后的响应结构
type AnswerResponse struct {
	NextQuestion           *QuestionOutput `json:"next_question,omitempty"`
	Message                string          `json:"message,omitempty"`
	QuizCompleted          bool            `json:"quiz_completed"`
	TotalQuestionsAnswered int             `json:"total_questions_answered"` // 已回答的题目数量 (1-based for display)
	WrongAnswersCount      int             `json:"wrong_answers_count,omitempty"`
}

// answerHandler 处理提交答案并获取下一道题的请求
func answerHandler(ctx context.Context, c *app.RequestContext) {
	var req AnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效的请求: " + err.Error()})
		return
	}

	sessionsMu.Lock()
	session, ok := userSessions[req.UserID]
	sessionsMu.Unlock() 

	if !ok {
		c.JSON(consts.StatusNotFound, utils.H{"error": "未找到用户会话。请重新开始答题。"})
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.CurrentQuestionIndex >= len(session.SelectedQuestions) {
		 log.Printf("用户 %s 尝试在答题完成后继续提交答案 (已完成 %d 题)。", req.UserID, len(session.SelectedQuestions))
		 c.JSON(consts.StatusOK, AnswerResponse{
			Message:                "答题已完成。",
			QuizCompleted:          true,
			TotalQuestionsAnswered: len(session.SelectedQuestions),
			WrongAnswersCount:      len(session.WronglyAnswered),
		})
		return
	}
	
	answeredQuestion := session.SelectedQuestions[session.CurrentQuestionIndex]
	// 验证前端提交的 DisplayQuestionNumber 是否与当前服务器期望的题目序号一致
	if answeredQuestion.DisplayQuestionNumber != req.DisplayQuestionNumber {
		log.Printf("用户 %s 提交答案时题目序号不匹配。期望显示序号: %s, 收到: %s (原始ID: %s)",
			req.UserID, answeredQuestion.DisplayQuestionNumber, req.DisplayQuestionNumber, answeredQuestion.OriginalQuestionID)
		c.JSON(consts.StatusBadRequest, utils.H{"error": fmt.Sprintf("题目序号不匹配。期望序号 %s, 收到 %s。", answeredQuestion.DisplayQuestionNumber, req.DisplayQuestionNumber)})
		return
	}

	if !req.WasCorrect {
		session.WronglyAnswered = append(session.WronglyAnswered, answeredQuestion)
		log.Printf("用户 %s 答错了题目 (显示序号: %s, 原始ID: %s)。", req.UserID, answeredQuestion.DisplayQuestionNumber, answeredQuestion.OriginalQuestionID)
	} else {
		log.Printf("用户 %s 答对了题目 (显示序号: %s, 原始ID: %s)。", req.UserID, answeredQuestion.DisplayQuestionNumber, answeredQuestion.OriginalQuestionID)
	}

	session.CurrentQuestionIndex++ 
	totalAnswered := session.CurrentQuestionIndex // 这代表已完成的题目数量 (1-based count because index moved past it)

	if session.CurrentQuestionIndex >= len(session.SelectedQuestions) {
		log.Printf("用户 %s 完成了所有题目。共答错 %d 道。", req.UserID, len(session.WronglyAnswered))
		c.JSON(consts.StatusOK, AnswerResponse{
			Message:                "恭喜！所有题目已回答完毕！",
			QuizCompleted:          true,
			TotalQuestionsAnswered: totalAnswered,
			WrongAnswersCount:      len(session.WronglyAnswered),
		})
	} else {
		nextQ := session.SelectedQuestions[session.CurrentQuestionIndex]
		log.Printf("用户 %s 获取下一题 (显示序号: %s, 原始ID: %s)。", req.UserID, nextQ.DisplayQuestionNumber, nextQ.OriginalQuestionID)
		c.JSON(consts.StatusOK, AnswerResponse{
			NextQuestion:           &nextQ,
			QuizCompleted:          false,
			TotalQuestionsAnswered: totalAnswered,
		})
	}
}

// ResultsResponse 定义了获取答题结果的响应结构
type ResultsResponse struct {
	UserID               string           `json:"user_id"`
	WronglyAnswered      []QuestionOutput `json:"wrongly_answered"`
	TotalAttempted       int              `json:"total_attempted"`       // 用户已尝试回答的题目数
	TotalCorrect         int              `json:"total_correct"`
	TotalQuestionsInQuiz int              `json:"total_questions_in_quiz"`
}

// resultsHandler 处理获取用户答题结果的请求
func resultsHandler(ctx context.Context, c *app.RequestContext) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "必须提供 user_id 查询参数"})
		return
	}
	
	sessionsMu.Lock()
	session, ok := userSessions[userID]
	sessionsMu.Unlock()

	if !ok {
		c.JSON(consts.StatusNotFound, utils.H{"error": "未找到用户会话。"})
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	
	totalAttempted := session.CurrentQuestionIndex 
	if session.SelectedQuestions == nil { 
		totalAttempted = 0
	}
	
	totalCorrect := totalAttempted - len(session.WronglyAnswered)
	if totalCorrect < 0 { 
		totalCorrect = 0
	}

	log.Printf("用户 %s 查询结果：共尝试 %d 题，答对 %d 题，总题数 %d。", userID, totalAttempted, totalCorrect, len(session.SelectedQuestions))
	c.JSON(consts.StatusOK, ResultsResponse{
		UserID:               session.UserID,
		WronglyAnswered:      session.WronglyAnswered,
		TotalAttempted:       totalAttempted,
		TotalCorrect:         totalCorrect,
		TotalQuestionsInQuiz: len(session.SelectedQuestions),
	})
}


func main() {
	rand.Seed(time.Now().UnixNano()) // 确保每次运行随机题目顺序不同

	err := loadQuestions("../clean_outputs/0.json") 
	if err != nil {
		log.Fatalf("致命错误: 加载题目失败: %v", err)
	}

	if len(allQuestions) < questionPoolSize {
		log.Printf("警告: 可用题目数量 (%d) 少于期望的题目池大小 (%d)。将使用所有 %d 道可用题目。", len(allQuestions), questionPoolSize, len(allQuestions))
		questionPoolSize = len(allQuestions)
        if questionPoolSize == 0 {
            log.Fatalf("致命错误: 没有可用的题目来运行答题程序。正在退出。")
        }
	}

	h := server.Default() 

	quizGroup := h.Group("/quiz")
	{
		quizGroup.POST("/start", startQuizHandler)
		quizGroup.POST("/answer", answerHandler)
		quizGroup.GET("/results", resultsHandler)
	}

	log.Println("服务器正在启动，默认监听端口为 8888 (例如 http://localhost:8888)")
	log.Println("API 端点:")
	log.Println("  POST /quiz/start     - 开始新的答题会话")
	log.Println("  POST /quiz/answer    - 提交答案并获取下一题")
	log.Println("  GET  /quiz/results?user_id=<ID> - 获取指定用户的答题结果")
	
	h.Spin()
}

