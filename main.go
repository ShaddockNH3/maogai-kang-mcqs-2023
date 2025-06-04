package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/google/uuid"
)

// --- 配置常量 ---
const (
	questionSourceDir      = "clean_outputs"
	maxChapterIndex        = 8
	userDataBaseDir        = "user_data"
	incorrectQuestionsFile = "incorrect_questions.json"
	questionStatsFile      = "question_stats.json"
)

// --- 数据结构定义 ---
type Question struct {
	QuestionNumber     string            `json:"question_number"`
	QuestionType       string            `json:"question_type"`
	QuestionText       string            `json:"question_text"`
	Options            map[string]string `json:"options"`
	CorrectAnswer      string            `json:"correct_answer"`
	GlobalCorrectCount int               `json:"correct_count"`
	GlobalErrorCount   int               `json:"error_count"`
	OriginalChapterKey string            `json:"-"`
	OriginalIndex      int               `json:"-"`
}

type QuestionOutput struct {
	QuizQuestionID         string            `json:"quiz_question_id"`
	DisplayNumber          int               `json:"display_number"`
	OriginalChapter        string            `json:"original_chapter"`
	OriginalQuestionNumber string            `json:"original_question_number"`
	QuestionType           string            `json:"question_type"`
	QuestionText           string            `json:"question_text"`
	Options                map[string]string `json:"options"`
	CorrectAnswer          string            `json:"correct_answer"` // 答案将始终包含
}

type UserIncorrectQuestion struct {
	QuestionNumber  string            `json:"question_number"`
	QuestionType    string            `json:"question_type"`
	QuestionText    string            `json:"question_text"`
	Options         map[string]string `json:"options"`
	CorrectAnswer   string            `json:"correct_answer"`
	OriginalChapter string            `json:"original_chapter"`
	UserAnswer      string            `json:"user_answer,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

type UserQuestionStat struct {
	OriginalChapterKey     string    `json:"original_chapter_key"`
	OriginalQuestionNumber string    `json:"original_question_number"`
	CorrectCount           int       `json:"correct_count"`
	ErrorCount             int       `json:"error_count"`
	LastAnswered           time.Time `json:"last_answered"`
}

type UserSession struct {
	UserID               string
	CurrentQuestions     []QuestionOutput        // 当前模式下用于显示的题目列表 (包含答案)
	OriginalIncorrect    []UserIncorrectQuestion // 在错题回顾模式下，这个可能不再直接使用，因为CurrentQuestions会包含答案
	CurrentQuestionIndex int
	CurrentMode          string 
	mu                   sync.Mutex
}

var (
	allQuestionsByChapter map[string][]Question
	questionMapByID       map[string]Question
	userSessions          map[string]*UserSession
	sessionsMu            sync.RWMutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
	allQuestionsByChapter = make(map[string][]Question)
	questionMapByID = make(map[string]Question)
	userSessions = make(map[string]*UserSession)
	if err := os.MkdirAll(userDataBaseDir, os.ModePerm); err != nil {
		log.Fatalf("无法创建用户数据目录 %s: %v", userDataBaseDir, err)
	}
	loadAllQuestionsGlobal()
}

func loadAllQuestionsGlobal() {
	log.Println("喵~ 正在努力加载全局题库中...")
	for i := 0; i <= maxChapterIndex; i++ {
		chapterKey := strconv.Itoa(i)
		filePath := filepath.Join(questionSourceDir, chapterKey+".json")
		fileData, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Printf("喵~ 提示：章节 %s 的题库文件 ('%s') 没找到呢，跳过这个章节啦。错误: %v", chapterKey, filePath, err)
			allQuestionsByChapter[chapterKey] = []Question{}
			continue
		}
		var questionsInChapter []Question
		if err := json.Unmarshal(fileData, &questionsInChapter); err != nil {
			log.Printf("喵呜！错误：解析章节 %s 的题库文件 ('%s') 失败了。错误: %v", chapterKey, filePath, err)
			allQuestionsByChapter[chapterKey] = []Question{}
			continue
		}
		for idx := range questionsInChapter {
			questionsInChapter[idx].OriginalChapterKey = chapterKey
			questionsInChapter[idx].OriginalIndex = idx
			questionID := fmt.Sprintf("%s_%d", chapterKey, idx)
			questionMapByID[questionID] = questionsInChapter[idx]
		}
		allQuestionsByChapter[chapterKey] = questionsInChapter
	}
	log.Println("喵~ 全局题库加载完毕！")
}

func getUserDataPath(userID, fileName string) string { return filepath.Join(userDataBaseDir, userID, fileName) }
func ensureUserDir(userID string) error { return os.MkdirAll(filepath.Join(userDataBaseDir, userID), os.ModePerm) }
func loadUserJSONData(userID, fileName string, target interface{}) error { if err := ensureUserDir(userID); err != nil { return err }; filePath := getUserDataPath(userID, fileName); if _, err := os.Stat(filePath); os.IsNotExist(err) { switch v := target.(type) { case *[]UserIncorrectQuestion: *v = []UserIncorrectQuestion{}; case *map[string]UserQuestionStat: *v = make(map[string]UserQuestionStat) }; return nil }; data, err := ioutil.ReadFile(filePath); if err != nil { return fmt.Errorf("读取用户文件 %s 失败: %w", filePath, err) }; if len(data) == 0 { switch v := target.(type) { case *[]UserIncorrectQuestion: *v = []UserIncorrectQuestion{}; case *map[string]UserQuestionStat: *v = make(map[string]UserQuestionStat) }; return nil }; return json.Unmarshal(data, target) }
func saveUserJSONData(userID, fileName string, data interface{}) error { if err := ensureUserDir(userID); err != nil { return err }; filePath := getUserDataPath(userID, fileName); jsonData, err := json.MarshalIndent(data, "", "  "); if err != nil { return fmt.Errorf("JSON序列化用户数据到 %s 失败: %w", filePath, err) }; return ioutil.WriteFile(filePath, jsonData, 0644) }
func getOrCreateUserSession(userID string) *UserSession { sessionsMu.RLock(); session, exists := userSessions[userID]; sessionsMu.RUnlock(); if exists { return session }; sessionsMu.Lock(); defer sessionsMu.Unlock(); session, exists = userSessions[userID]; if exists { return session }; newSession := &UserSession{ UserID: userID }; userSessions[userID] = newSession; return newSession }
func _getQuestionsForProcessing(chapterChoices []string, orderChoice string) []Question { var questionsToProcess []Question; var targetChapterKeys []string; isSelectAll := false; for _, choice := range chapterChoices { if strings.ToLower(choice) == "all" || choice == "9" { isSelectAll = true; break } }; if isSelectAll { for i := 0; i <= maxChapterIndex; i++ { targetChapterKeys = append(targetChapterKeys, strconv.Itoa(i)) }; sort.SliceStable(targetChapterKeys, func(i, j int) bool { numI, _ := strconv.Atoi(targetChapterKeys[i]); numJ, _ := strconv.Atoi(targetChapterKeys[j]); return numI < numJ }) } else { for _, choice := range chapterChoices { if _, err := strconv.Atoi(choice); err == nil { if _, ok := allQuestionsByChapter[choice]; ok { targetChapterKeys = append(targetChapterKeys, choice) } } } }; for _, chapKey := range targetChapterKeys { chapterQuestions, ok := allQuestionsByChapter[chapKey]; if ok { questionsToProcess = append(questionsToProcess, chapterQuestions...) } }; if len(questionsToProcess) == 0 { return []Question{} }; if orderChoice == "random" || orderChoice == "1" { rand.Shuffle(len(questionsToProcess), func(i, j int) { questionsToProcess[i], questionsToProcess[j] = questionsToProcess[j], questionsToProcess[i] }) }; return questionsToProcess }

// convertQuestionsToOutput now always includes the answer by default.
// The 'includeAnswer' parameter is kept for potential future use but defaults to true.
func convertQuestionsToOutput(questions []Question, sessionIndexOffset int, forceIncludeAnswer ...bool) []QuestionOutput {
	includeAnswer := true // Default to true
	if len(forceIncludeAnswer) > 0 {
		includeAnswer = forceIncludeAnswer[0] // Allow overriding (though not used in current refactor)
	}

	output := make([]QuestionOutput, len(questions))
	for i, q := range questions {
		outQ := QuestionOutput{
			QuizQuestionID:         fmt.Sprintf("quiz_%s_%d", q.OriginalChapterKey, q.OriginalIndex),
			DisplayNumber:          sessionIndexOffset + i + 1,
			OriginalChapter:        q.OriginalChapterKey,
			OriginalQuestionNumber: q.QuestionNumber,
			QuestionType:           q.QuestionType,
			QuestionText:           q.QuestionText,
			Options:                q.Options,
		}
		if includeAnswer { // This will always be true now based on new requirement
			outQ.CorrectAnswer = q.CorrectAnswer
		}
		output[i] = outQ
	}
	return output
}

// convertUserIncorrectToOutput now always includes the answer.
func convertUserIncorrectToOutput(incorrectQs []UserIncorrectQuestion, sessionIndexOffset int) []QuestionOutput {
	output := make([]QuestionOutput, len(incorrectQs))
	for i, iq := range incorrectQs {
		output[i] = QuestionOutput{
			QuizQuestionID:         fmt.Sprintf("incorrect_%s_%s_%d", iq.OriginalChapter, iq.QuestionNumber, sessionIndexOffset+i),
			DisplayNumber:          sessionIndexOffset + i + 1,
			OriginalChapter:        iq.OriginalChapter,
			OriginalQuestionNumber: iq.QuestionNumber,
			QuestionType:           iq.QuestionType,
			QuestionText:           iq.QuestionText,
			Options:                iq.Options,
			CorrectAnswer:          iq.CorrectAnswer, // Always include answer for incorrect review
		}
	}
	return output
}

func InitSessionHandler(ctx context.Context, c *app.RequestContext) { userID := uuid.NewString(); _ = getOrCreateUserSession(userID); log.Printf("新用户会话初始化: %s", userID); c.JSON(consts.StatusOK, utils.H{"user_id": userID, "message": "会话已初始化"}) }
type StartModeRequest struct { UserID string `json:"user_id" vd:"required"`; ChapterChoice []string `json:"chapter_choice" vd:"required"`; OrderChoice string `json:"order_choice" vd:"required"` }

func QuickReviewStartHandler(ctx context.Context, c *app.RequestContext) { var req StartModeRequest; if err := c.BindAndValidate(&req); err != nil { c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()}); return }; session := getOrCreateUserSession(req.UserID); session.mu.Lock(); defer session.mu.Unlock(); selectedQuestions := _getQuestionsForProcessing(req.ChapterChoice, req.OrderChoice); if len(selectedQuestions) == 0 { c.JSON(consts.StatusOK, utils.H{"message": "所选范围没有题目。", "total_questions": 0, "question": nil}); return }; session.CurrentQuestions = convertQuestionsToOutput(selectedQuestions, 0, true); session.CurrentQuestionIndex = 0; session.CurrentMode = "review"; log.Printf("用户 %s 开始速刷模式，章节: %v, 顺序: %s, 共 %d 题", req.UserID, req.ChapterChoice, req.OrderChoice, len(session.CurrentQuestions)); c.JSON(consts.StatusOK, utils.H{ "message": "速刷模式开始", "total_questions": len(session.CurrentQuestions), "question": session.CurrentQuestions[0] }) }
type GetNextQuestionRequest struct { UserID string `json:"user_id" vd:"required"` }

func GetNextQuestionHandler(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()

	// This handler is now ONLY for "review" (Quick Review) mode.
	if session.CurrentMode != "review" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "当前不处于速刷模式"})
		return
	}
	
	session.CurrentQuestionIndex++
	if session.CurrentQuestionIndex >= len(session.CurrentQuestions) {
		c.JSON(consts.StatusOK, utils.H{"message": "速刷完成!", "quiz_completed": true, "question": nil})
		return
	}
	
	nextQuestionOutput := session.CurrentQuestions[session.CurrentQuestionIndex] // Already includes answer
	log.Printf("用户 %s 在速刷模式下获取下一题, 序号 %d", req.UserID, nextQuestionOutput.DisplayNumber)
	c.JSON(consts.StatusOK, utils.H{
		"question":       nextQuestionOutput,
		"quiz_completed": false,
	})
}

func QuizStartHandler(ctx context.Context, c *app.RequestContext) {
	var req StartModeRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()
	selectedQuestions := _getQuestionsForProcessing(req.ChapterChoice, req.OrderChoice)
	if len(selectedQuestions) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "所选范围没有题目。", "total_questions": 0, "question": nil})
		return
	}
	session.CurrentQuestions = convertQuestionsToOutput(selectedQuestions, 0, true) // Always include answer
	session.CurrentQuestionIndex = 0
	session.CurrentMode = "quiz"
	log.Printf("用户 %s 开始答题模式，章节: %v, 顺序: %s, 共 %d 题", req.UserID, req.ChapterChoice, req.OrderChoice, len(session.CurrentQuestions))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "答题模式开始",
		"total_questions": len(session.CurrentQuestions),
		"question":        session.CurrentQuestions[0],
	})
}

type SubmitAnswerRequest struct {
	UserID           string `json:"user_id" vd:"required"`
	QuizQuestionID   string `json:"quiz_question_id" vd:"required"`
	UserAnswer       string `json:"user_answer" vd:"required"`
	WasCorrect       bool   `json:"was_correct"` // Frontend sends if it was correct
}

func SubmitAnswerHandler(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	if session.CurrentMode != "quiz" {
		session.mu.Unlock()
		c.JSON(consts.StatusBadRequest, utils.H{"error": "当前不处于答题模式"})
		return
	}
	if session.CurrentQuestionIndex >= len(session.CurrentQuestions) {
		session.mu.Unlock()
		c.JSON(consts.StatusOK, utils.H{"message": "答题已完成!", "quiz_completed": true})
		return
	}
	currentOutputQuestion := session.CurrentQuestions[session.CurrentQuestionIndex]
	if currentOutputQuestion.QuizQuestionID != req.QuizQuestionID {
		session.mu.Unlock()
		c.JSON(consts.StatusBadRequest, utils.H{"error": "提交的题目ID与当前题目不符"})
		return
	}
	parts := strings.Split(strings.TrimPrefix(req.QuizQuestionID, "quiz_"), "_")
	if len(parts) != 2 {
		session.mu.Unlock()
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "内部服务器错误，无法解析题目ID"})
		return
	}
	originalQuestionIDKey := parts[0] + "_" + parts[1]
	originalQuestion, ok := questionMapByID[originalQuestionIDKey]
	if !ok {
		session.mu.Unlock()
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "内部服务器错误，找不到原始题目"})
		return
	}
	
	// Frontend now determines correctness, backend just logs and updates stats
	userStats := make(map[string]UserQuestionStat)
	if err := loadUserJSONData(req.UserID, questionStatsFile, &userStats); err != nil {
		session.mu.Unlock()
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户统计数据失败"})
		return
	}
	statKey := fmt.Sprintf("%s_%s", originalQuestion.OriginalChapterKey, originalQuestion.QuestionNumber)
	statEntry, statExists := userStats[statKey]
	if !statExists {
		statEntry = UserQuestionStat{
			OriginalChapterKey:     originalQuestion.OriginalChapterKey,
			OriginalQuestionNumber: originalQuestion.QuestionNumber,
		}
	}

	if req.WasCorrect {
		statEntry.CorrectCount++
		log.Printf("用户 %s 答对题目 %s (原始ID %s) - 前端判断", req.UserID, currentOutputQuestion.DisplayNumber, originalQuestion.QuestionNumber)
	} else {
		statEntry.ErrorCount++
		log.Printf("用户 %s 答错题目 %s (原始ID %s)，用户答案: %s, 正确答案: %s - 前端判断", req.UserID, currentOutputQuestion.DisplayNumber, originalQuestion.QuestionNumber, req.UserAnswer, originalQuestion.CorrectAnswer)
		userIncorrect := []UserIncorrectQuestion{}
		if err := loadUserJSONData(req.UserID, incorrectQuestionsFile, &userIncorrect); err != nil {
			session.mu.Unlock()
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户错题本失败"})
			return
		}
		isDuplicate := false
		for _, iq := range userIncorrect {
			if iq.QuestionText == originalQuestion.QuestionText && iq.OriginalChapter == originalQuestion.OriginalChapterKey {
				isDuplicate = true; break
			}
		}
		if !isDuplicate {
			userIncorrect = append(userIncorrect, UserIncorrectQuestion{
				QuestionNumber:  originalQuestion.QuestionNumber,
				QuestionType:    originalQuestion.QuestionType,
				QuestionText:    originalQuestion.QuestionText,
				Options:         originalQuestion.Options,
				CorrectAnswer:   originalQuestion.CorrectAnswer,
				OriginalChapter: originalQuestion.OriginalChapterKey,
				UserAnswer:      req.UserAnswer,
				Timestamp:       time.Now(),
			})
			if err := saveUserJSONData(req.UserID, incorrectQuestionsFile, userIncorrect); err != nil {
				session.mu.Unlock()
				c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存用户错题本失败"})
				return
			}
		}
	}
	statEntry.LastAnswered = time.Now()
	userStats[statKey] = statEntry
	if err := saveUserJSONData(req.UserID, questionStatsFile, userStats); err != nil {
		session.mu.Unlock()
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存用户统计数据失败"})
		return
	}

	session.CurrentQuestionIndex++
	quizCompleted := session.CurrentQuestionIndex >= len(session.CurrentQuestions)
	var nextQ *QuestionOutput = nil
	if !quizCompleted {
		q := session.CurrentQuestions[session.CurrentQuestionIndex] // This already has the answer
		nextQ = &q
	}
	session.mu.Unlock()
	c.JSON(consts.StatusOK, utils.H{
		"next_question":            nextQ,
		"quiz_completed":           quizCompleted,
		"total_questions_answered": session.CurrentQuestionIndex,
		"message":                  "答案已记录 (前端校验)",
	})
}

func IncorrectQuestionsReviewStartHandler(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()
	userIncorrectRaw := []UserIncorrectQuestion{}
	if err := loadUserJSONData(req.UserID, incorrectQuestionsFile, &userIncorrectRaw); err != nil {
		log.Printf("用户 %s 加载错题本失败 (回顾模式): %v", req.UserID, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户错题本失败"})
		return
	}
	if len(userIncorrectRaw) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "错题簿是空的哦！", "total_questions": 0, "question": nil})
		return
	}
	rand.Shuffle(len(userIncorrectRaw), func(i, j int) {
		userIncorrectRaw[i], userIncorrectRaw[j] = userIncorrectRaw[j], userIncorrectRaw[i]
	})
	
	session.CurrentQuestions = convertUserIncorrectToOutput(userIncorrectRaw, 0) // Always include answer
	session.CurrentQuestionIndex = 0
	session.CurrentMode = "incorrect_review"
	log.Printf("用户 %s 开始错题回顾模式, 共 %d 题", req.UserID, len(session.CurrentQuestions))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "错题回顾模式开始",
		"total_questions": len(session.CurrentQuestions),
		"question":        session.CurrentQuestions[0], 
	})
}

// SubmitIncorrectReviewAnswerHandler now only records, frontend validates
func SubmitIncorrectReviewAnswerHandler(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.CurrentMode != "incorrect_review" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "当前不处于错题回顾模式"})
		return
	}
	if session.CurrentQuestionIndex >= len(session.CurrentQuestions) {
		c.JSON(consts.StatusOK, utils.H{"message": "错题回顾已完成!", "quiz_completed": true, "question": nil})
		return
	}

	currentOutputQuestion := session.CurrentQuestions[session.CurrentQuestionIndex] // This already has the answer
	if currentOutputQuestion.QuizQuestionID != req.QuizQuestionID {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "提交的错题ID与当前题目不符"})
		return
	}
	
	// Log the attempt. Frontend already validated.
	log.Printf("用户 %s 在错题回顾中尝试了题目 %s (ID %s), 用户答案: %s, 前端判断正确性: %t",
		req.UserID, currentOutputQuestion.DisplayNumber, currentOutputQuestion.OriginalQuestionNumber, req.UserAnswer, req.WasCorrect)
	
	session.CurrentQuestionIndex++
	quizCompleted := session.CurrentQuestionIndex >= len(session.CurrentQuestions)
	var nextQ *QuestionOutput = nil
	if !quizCompleted {
		nextQ = &session.CurrentQuestions[session.CurrentQuestionIndex] // This already includes answer
	}

	c.JSON(consts.StatusOK, utils.H{
		"next_question":            nextQ,
		"quiz_completed":           quizCompleted,
		"total_questions_answered": session.CurrentQuestionIndex,
		"message":                  "错题回顾答案已记录 (前端校验)",
	})
}

func UserDataClearHandler(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	incorrectPath := getUserDataPath(req.UserID, incorrectQuestionsFile)
	if _, err := os.Stat(incorrectPath); err == nil {
		if err := os.Remove(incorrectPath); err != nil {
			log.Printf("用户 %s 清理错题文件 %s 失败: %v", req.UserID, incorrectPath, err)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "清理用户错题失败"})
			return
		}
	}
	statsPath := getUserDataPath(req.UserID, questionStatsFile)
	if _, err := os.Stat(statsPath); err == nil {
		if err := os.Remove(statsPath); err != nil {
			log.Printf("用户 %s 清理统计文件 %s 失败: %v", req.UserID, statsPath, err)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "清理用户统计失败"})
			return
		}
	}
	log.Printf("用户 %s 的数据已清理。", req.UserID)
	c.JSON(consts.StatusOK, utils.H{"message": "用户数据已成功清理。"})
}

func main() {
	h := server.Default(server.WithHostPorts("0.0.0.0:8888"))
	apiGroup := h.Group("/api")
	{
		sessionGroup := apiGroup.Group("/session")
		{
			sessionGroup.POST("/init", InitSessionHandler)
		}
		reviewGroup := apiGroup.Group("/review") 
		{
			reviewGroup.POST("/start", QuickReviewStartHandler)
			reviewGroup.POST("/next", GetNextQuestionHandler) 
		}
		quizGroup := apiGroup.Group("/quiz") 
		{
			quizGroup.POST("/start", QuizStartHandler)
			quizGroup.POST("/submit_answer", SubmitAnswerHandler)
		}
		incorrectGroup := apiGroup.Group("/incorrect_questions") 
		{
			incorrectGroup.POST("/review/start", IncorrectQuestionsReviewStartHandler)
			incorrectGroup.POST("/review/submit_answer", SubmitIncorrectReviewAnswerHandler)
		}
		userGroup := apiGroup.Group("/user")
		{
			userGroup.POST("/data/clear", UserDataClearHandler)
		}
	}
	log.Println("喵喵学习小助手 Go 后端已启动，监听于 http://0.0.0.0:8888")
	h.Spin()
}

