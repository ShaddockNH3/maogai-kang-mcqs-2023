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
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

var (
	maogaiQuestionsByChapter map[string][]Question   // 毛概题目
	xigaiQuestionsByChapter  map[string][]Question   // 习概题目
	questionMapByID          map[string]Question     // 通过唯一ID (课程_章节_索引) 快速查找原始题目
	userSessions             map[string]*UserSession // 内存中的用户会话
	sessionsMu               sync.RWMutex            // 保护 userSessions 映射
)

// init 在程序启动时执行初始化操作
func init() {
	rand.Seed(time.Now().UnixNano()) // 初始化随机数生成器
	maogaiQuestionsByChapter = make(map[string][]Question)
	xigaiQuestionsByChapter = make(map[string][]Question)
	questionMapByID = make(map[string]Question)
	userSessions = make(map[string]*UserSession)

	// 确保用户数据根目录存在
	if err := os.MkdirAll(userDataBaseDir, os.ModePerm); err != nil {
		log.Fatalf("无法创建用户数据目录 %s: %v", userDataBaseDir, err)
	}
	loadAllQuestionsGlobal() // 加载所有题目到内存
}

// loadAllQuestionsGlobal 从嵌入文件系统加载所有章节的题目到全局变量
func loadAllQuestionsGlobal() {
	log.Println("喵~ 正在努力加载全局题库中...")

	// 加载毛概题库
	log.Println("加载毛概题库...")
	for i := 0; i <= maogaiMaxChapterIndex; i++ {
		chapterKey := strconv.Itoa(i)
		// 嵌入文件系统使用正斜杠，不使用filepath.Join
		filePath := maogaiQuestionSourceDir + "/" + chapterKey + ".json"
		loadChapterQuestions(filePath, chapterKey, "maogai", maogaiQuestionsByChapter)
	}

	// 加载习概题库
	log.Println("加载习概题库...")
	for i := 0; i <= xigaiMaxChapterIndex; i++ {
		chapterKey := strconv.Itoa(i)
		// 嵌入文件系统使用正斜杠，不使用filepath.Join
		filePath := xigaiQuestionSourceDir + "/" + chapterKey + ".json"
		loadChapterQuestions(filePath, chapterKey, "xigai", xigaiQuestionsByChapter)
	}

	log.Println("喵~ 全局题库加载完毕！")
}

// loadChapterQuestions 加载单个章节的题目
func loadChapterQuestions(filePath, chapterKey, course string, targetMap map[string][]Question) {
	fileData, err := embeddedFS.ReadFile(filePath)
	if err != nil {
		log.Printf("喵~ 提示：章节 %s (%s) 的题库文件 (%s) 没找到呢,跳过这个章节啦。错误: %v", chapterKey, course, filePath, err)
		targetMap[chapterKey] = []Question{} // 即使文件不存在,也初始化为空列表
		return
	}

	var questionsInChapter []Question
	if err := json.Unmarshal(fileData, &questionsInChapter); err != nil {
		log.Printf("喵呜！错误：解析章节 %s (%s) 的题库文件 (%s) 失败了。错误: %v", chapterKey, course, filePath, err)
		targetMap[chapterKey] = []Question{} // 解析失败也初始化为空列表
		return
	}

	for idx := range questionsInChapter {
		questionsInChapter[idx].OriginalChapterKey = chapterKey
		questionsInChapter[idx].OriginalIndex = idx
		// 使用 "课程_章节号_题目在文件中的索引" 作为唯一ID
		questionID := fmt.Sprintf("%s_%s_%d", course, chapterKey, idx)
		questionMapByID[questionID] = questionsInChapter[idx]
	}
	targetMap[chapterKey] = questionsInChapter
	log.Printf("加载了 %d 道 %s 第%s章题目", len(questionsInChapter), course, chapterKey)
}

// --- 用户数据持久化帮助函数 ---

// getIncorrectQuestionsFileName 根据课程返回对应的错题文件名
func getIncorrectQuestionsFileName(course string) string {
	if course == "xigai" {
		return xigaiIncorrectQuestionsFile
	}
	return maogaiIncorrectQuestionsFile
}

// getUserDataPath 获取用户特定数据文件的完整路径
func getUserDataPath(userID, fileName string) string {
	return filepath.Join(userDataBaseDir, userID, fileName)
}

// ensureUserDir 确保用户的个人数据目录存在，如果不存在则创建
func ensureUserDir(userID string) error {
	return os.MkdirAll(filepath.Join(userDataBaseDir, userID), os.ModePerm)
}

// loadUserJSONData 加载用户特定的JSON数据文件到指定的结构体
// 如果文件不存在或为空，会初始化目标结构体为空状态（例如，空切片或空映射）
func loadUserJSONData(userID, fileName string, target interface{}) error {
	if err := ensureUserDir(userID); err != nil { // 确保用户目录存在
		return err
	}
	filePath := getUserDataPath(userID, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，根据目标类型初始化为空
		switch v := target.(type) {
		case *[]UserIncorrectQuestion:
			*v = []UserIncorrectQuestion{}
		case *map[string]UserQuestionStat:
			*v = make(map[string]UserQuestionStat)
		default:
			// 对于其他类型，可以返回错误或尝试其他初始化，但通常是空切片/映射
			log.Printf("加载用户数据: 文件 %s 不存在，目标类型 %T 将保持默认零值或需要特定处理", filePath, target)
		}
		return nil // 文件不存在不是一个错误，表示用户还没有这类数据
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取用户文件 %s 失败: %w", filePath, err)
	}

	if len(data) == 0 { // 文件存在但为空
		switch v := target.(type) {
		case *[]UserIncorrectQuestion:
			*v = []UserIncorrectQuestion{}
		case *map[string]UserQuestionStat:
			*v = make(map[string]UserQuestionStat)
		default:
			log.Printf("加载用户数据: 文件 %s 为空，目标类型 %T 将保持默认零值或需要特定处理", filePath, target)
		}
		return nil // 空文件也表示没有数据
	}

	return json.Unmarshal(data, target)
}

// saveUserJSONData 将用户数据（通常是结构体或映射）序列化为JSON并保存到文件
func saveUserJSONData(userID, fileName string, data interface{}) error {
	if err := ensureUserDir(userID); err != nil { // 确保用户目录存在
		return err
	}
	filePath := getUserDataPath(userID, fileName)
	jsonData, err := json.MarshalIndent(data, "", "  ") // 使用缩进美化JSON输出
	if err != nil {
		return fmt.Errorf("JSON序列化用户数据到 %s 失败: %w", filePath, err)
	}
	return ioutil.WriteFile(filePath, jsonData, 0644) // 0644 文件权限
}

// --- 会话管理 ---

// getOrCreateUserSession 获取或创建用户会话。如果会话不存在，则在内存中创建一个新的。
func getOrCreateUserSession(userID string) *UserSession {
	sessionsMu.RLock()
	session, exists := userSessions[userID]
	sessionsMu.RUnlock()
	if exists {
		return session
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	// 再次检查，防止在获取写锁期间其他goroutine已创建会话 (双重检查锁定模式)
	session, exists = userSessions[userID]
	if exists {
		return session
	}

	newSession := &UserSession{
		UserID: userID,
		// CurrentQuestions, OriginalIncorrect, CurrentQuestionIndex, CurrentMode 会在特定模式开始时设置
	}
	userSessions[userID] = newSession
	return newSession
}

// _getQuestionsForProcessing 根据章节和顺序选择,从全局题库中筛选和排序题目
func _getQuestionsForProcessing(course string, chapterChoices []string, orderChoice string) []Question {
	var questionsToProcess []Question
	var targetChapterKeys []string
	isSelectAll := false

	// 根据课程选择题库和最大章节索引
	var questionsByChapter map[string][]Question
	var maxChapterIdx int

	if course == "xigai" {
		questionsByChapter = xigaiQuestionsByChapter
		maxChapterIdx = xigaiMaxChapterIndex
	} else {
		questionsByChapter = maogaiQuestionsByChapter
		maxChapterIdx = maogaiMaxChapterIndex
	}

	for _, choice := range chapterChoices {
		if strings.ToLower(choice) == "all" || choice == "9" { // "9" 作为 "all" 的别名兼容旧版或简化输入
			isSelectAll = true
			break
		}
	}

	if isSelectAll {
		for i := 0; i <= maxChapterIdx; i++ { // 加载0到maxChapterIdx的所有章节
			targetChapterKeys = append(targetChapterKeys, strconv.Itoa(i))
		}
		// 确保章节键按数字顺序排列
		sort.SliceStable(targetChapterKeys, func(i, j int) bool {
			numI, _ := strconv.Atoi(targetChapterKeys[i])
			numJ, _ := strconv.Atoi(targetChapterKeys[j])
			return numI < numJ
		})
	} else {
		// 只加载用户选择的特定章节
		for _, choice := range chapterChoices {
			if _, err := strconv.Atoi(choice); err == nil { // 确保 choice 是一个有效的数字字符串
				if _, ok := questionsByChapter[choice]; ok { // 确保章节数据存在
					targetChapterKeys = append(targetChapterKeys, choice)
				} else {
					log.Printf("警告: 请求的章节 %s (%s) 在题库中不存在,已跳过。", choice, course)
				}
			} else {
				log.Printf("警告: 无效的章节选择 %s,已跳过。", choice)
			}
		}
	}

	for _, chapKey := range targetChapterKeys {
		chapterQuestions, ok := questionsByChapter[chapKey]
		if ok {
			questionsToProcess = append(questionsToProcess, chapterQuestions...)
		}
	}

	if len(questionsToProcess) == 0 {
		return []Question{} // 如果没有选出任何题目,返回空切片
	}

	// 根据选择的顺序处理题目
	if orderChoice == "random" || orderChoice == "1" { // "1" 作为 "random" 的别名
		rand.Shuffle(len(questionsToProcess), func(i, j int) {
			questionsToProcess[i], questionsToProcess[j] = questionsToProcess[j], questionsToProcess[i]
		})
	}
	// 如果是 "sequential" 或 "0" (或任何其他值),则按原始顺序（章节顺序,章节内题目顺序）
	return questionsToProcess
}

// --- DTO转换函数 ---

// convertQuestionsToOutput 将原始 Question 结构体列表转换为 QuestionOutput 列表，用于API响应。
// 始终包含答案。
func convertQuestionsToOutput(questions []Question, sessionIndexOffset int, course string) []QuestionOutput {
	output := make([]QuestionOutput, len(questions))
	for i, q := range questions {
		output[i] = QuestionOutput{
			QuizQuestionID:         fmt.Sprintf("quiz_%s_%s_%d", course, q.OriginalChapterKey, q.OriginalIndex), // 唯一ID，格式: quiz_课程_章节_原始索引
			DisplayNumber:          sessionIndexOffset + i + 1,                                                  // 基于最终列表的显示序号 (1-based)
			OriginalChapter:        q.OriginalChapterKey,
			OriginalQuestionNumber: q.QuestionNumber,
			QuestionType:           q.QuestionType,
			QuestionText:           q.QuestionText,
			Options:                q.Options,
			CorrectAnswer:          q.CorrectAnswer, // 始终包含答案
		}
	}
	return output
}

// convertUserIncorrectToOutput 将用户错题列表 UserIncorrectQuestion 转换为 QuestionOutput 列表。
// 始终包含答案。
func convertUserIncorrectToOutput(incorrectQs []UserIncorrectQuestion, sessionIndexOffset int, course string) []QuestionOutput {
	output := make([]QuestionOutput, len(incorrectQs))
	for i, iq := range incorrectQs {
		// 为错题生成一个唯一的 QuizQuestionID，可以加上时间戳或随机数以区分同一道题的多次回顾（如果需要）
		// 这里简化处理，基于原始章节和题号，加上列表索引
		output[i] = QuestionOutput{
			QuizQuestionID:         fmt.Sprintf("incorrect_%s_%s_%s_%d", course, iq.OriginalChapter, iq.QuestionNumber, sessionIndexOffset+i),
			DisplayNumber:          sessionIndexOffset + i + 1,
			OriginalChapter:        iq.OriginalChapter,
			OriginalQuestionNumber: iq.QuestionNumber,
			QuestionType:           iq.QuestionType,
			QuestionText:           iq.QuestionText,
			Options:                iq.Options,
			CorrectAnswer:          iq.CorrectAnswer, // 始终包含答案
		}
	}
	return output
}

// --- API 处理函数 ---

// InitSessionHandler 处理用户会话初始化请求
// 现在接收客户端提供的 userID
func InitSessionHandler(ctx context.Context, c *app.RequestContext) {
	var req InitSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效的请求: user_id 是必需的。 " + err.Error()})
		return
	}

	userID := req.UserID

	// 检查用户数据目录是否存在，以判断是新用户还是返回用户
	userDir := filepath.Join(userDataBaseDir, userID)
	isNewUser := false
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		isNewUser = true
		// 确保为新用户创建目录
		if errDir := ensureUserDir(userID); errDir != nil {
			log.Printf("错误: 为用户 %s 创建目录 %s 失败: %v", userID, userDir, errDir)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "无法初始化用户数据存储区"})
			return
		}
		log.Printf("信息: 新用户 %s 首次使用，已创建用户目录: %s", userID, userDir)
	} else if err != nil {
		// 其他 os.Stat 错误
		log.Printf("错误: 检查用户目录 %s 时发生错误: %v", userDir, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "检查用户数据时出错"})
		return
	}

	session := getOrCreateUserSession(userID) // 获取或创建内存中的会话

	message := "用户会话已建立"
	if isNewUser {
		message = "新用户会话已创建并初始化成功"
		log.Printf("会话: 新用户 %s 的会话已在内存中创建。", userID)
	} else {
		log.Printf("会话: 用户 %s 的会话已从内存中获取或新建。", userID)
	}

	// 可以在这里预加载一些用户数据到会话中，如果需要的话
	// 例如: session.SomeData = loadSpecificDataForUser(userID)

	c.JSON(consts.StatusOK, utils.H{"user_id": session.UserID, "message": message, "is_new_user": isNewUser})
}

// QuickReviewStartHandler 处理开始速刷模式的请求。
// 现在返回所有选定问题及其答案给前端。
func QuickReviewStartHandler(ctx context.Context, c *app.RequestContext) {
	var req StartModeRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}

	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock() // 如果要修改会话状态（如 CurrentMode），则加锁
	defer session.mu.Unlock()

	selectedQuestions := _getQuestionsForProcessing(req.Course, req.ChapterChoice, req.OrderChoice)
	if len(selectedQuestions) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "所选范围没有题目。", "total_questions": 0, "questions": []QuestionOutput{}})
		return
	}

	outputQuestions := convertQuestionsToOutput(selectedQuestions, 0, req.Course) // 0 表示从列表开头计数
	session.CurrentMode = "review"                                                // 设置当前模式
	session.CurrentCourse = req.Course                                            // 设置当前课程
	// 如果 /api/review/next 仍然用于逐步获取，则需要存储这些问题
	// 否则，如果前端一次性处理所有问题，这一步可以省略或用于其他目的
	session.CurrentQuestions = outputQuestions
	session.CurrentQuestionIndex = 0 // 从第一题开始

	log.Printf("用户 %s 开始速刷模式，课程: %s, 章节: %v, 顺序: %s, 返回 %d 题", req.UserID, req.Course, req.ChapterChoice, req.OrderChoice, len(outputQuestions))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "速刷模式开始",
		"total_questions": len(outputQuestions),
		"questions":       outputQuestions, // 发送所有问题给前端
	})
}

// GetNextQuestionHandler 处理获取速刷模式下一题的请求。
// 这个接口的必要性取决于前端是否自行管理题目列表。如果前端一次性获取所有题目，则此接口可能多余。
// 保留它可能是为了支持一种简化的“点击下一题”逻辑，或者用于特定的复习流程。
func GetNextQuestionHandler(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.CurrentMode != "review" { // 确保当前是速刷模式
		c.JSON(consts.StatusBadRequest, utils.H{"error": "当前不处于速刷模式 (或会话模式不匹配)"})
		return
	}

	// 增加当前题目索引
	session.CurrentQuestionIndex++
	if session.CurrentQuestionIndex >= len(session.CurrentQuestions) {
		// 所有题目已浏览完毕
		c.JSON(consts.StatusOK, utils.H{"message": "速刷完成!", "quiz_completed": true, "question": nil})
		return
	}

	nextQuestionOutput := session.CurrentQuestions[session.CurrentQuestionIndex]
	log.Printf("用户 %s 在速刷模式下通过API获取下一题, 序号 %d (原始问题ID: %s)", req.UserID, nextQuestionOutput.DisplayNumber, nextQuestionOutput.QuizQuestionID)
	c.JSON(consts.StatusOK, utils.H{
		"question":       nextQuestionOutput,
		"quiz_completed": false,
	})
}

// QuizStartHandler 处理开始答题模式的请求。
// 现在返回所有选定问题及其答案给前端，前端负责隐藏答案直到用户提交。
func QuizStartHandler(ctx context.Context, c *app.RequestContext) {
	var req StartModeRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()

	selectedQuestions := _getQuestionsForProcessing(req.Course, req.ChapterChoice, req.OrderChoice)
	if len(selectedQuestions) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "所选范围没有题目。", "total_questions": 0, "questions": []QuestionOutput{}})
		return
	}

	outputQuestions := convertQuestionsToOutput(selectedQuestions, 0, req.Course)
	session.CurrentMode = "quiz"       // 设置模式，用于提交答案时的上下文
	session.CurrentCourse = req.Course // 设置当前课程

	// 如果前端完全管理题目列表和导航，则不在会话中存储 CurrentQuestions 和 CurrentQuestionIndex
	// session.CurrentQuestions = outputQuestions
	// session.CurrentQuestionIndex = 0

	log.Printf("用户 %s 开始答题模式，课程: %s, 章节: %v, 顺序: %s, 返回 %d 题", req.UserID, req.Course, req.ChapterChoice, req.OrderChoice, len(outputQuestions))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "答题模式开始",
		"total_questions": len(outputQuestions),
		"questions":       outputQuestions, // 发送所有问题给前端
	})
}

// SubmitAnswerHandler 处理用户在答题模式下提交的答案。
// 主要职责是记录用户答题统计和错题。前端已处理答案校验。
func SubmitAnswerHandler(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest
	if err := c.BindAndValidate(&req); err != nil {
		log.Printf("[DEBUG] SubmitAnswerHandler 绑定请求失败: %v", err)
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	log.Printf("[DEBUG] SubmitAnswerHandler 收到请求: UserID=%s, QuizQuestionID=%s, UserAnswer=%s, WasCorrect=%t",
		req.UserID, req.QuizQuestionID, req.UserAnswer, req.WasCorrect)

	// 从用户 session 获取当前课程
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	currentCourse := session.CurrentCourse
	session.mu.Unlock()

	if currentCourse == "" {
		log.Printf("错误: 用户 %s 的 session 中没有课程信息", req.UserID)
		c.JSON(consts.StatusBadRequest, utils.H{"error": "会话中缺少课程信息"})
		return
	}

	// 从 QuizQuestionID 中解析出原始题目信息 (课程、章节号和原始索引)
	// QuizQuestionID 格式为 "quiz_课程_章节号_原始索引"
	parts := strings.Split(strings.TrimPrefix(req.QuizQuestionID, "quiz_"), "_")
	if len(parts) < 3 { // 至少需要课程、章节和索引三部分
		log.Printf("错误: 无法从 QuizQuestionID %s 解析原始题目信息 (答题模式)", req.QuizQuestionID)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "内部服务器错误，无法解析题目ID (quiz_submit)"})
		return
	}
	originalQuestionIDKey := parts[0] + "_" + parts[1] + "_" + parts[2] // 重组为 "课程_章节号_原始索引"
	originalQuestion, ok := questionMapByID[originalQuestionIDKey]
	if !ok {
		log.Printf("错误: 找不到 QuizQuestionID %s (解析为Key: %s) 对应的原始题目 (答题模式)", req.QuizQuestionID, originalQuestionIDKey)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "内部服务器错误，找不到原始题目 (quiz_submit_map)"})
		return
	}

	// 加载或初始化用户统计数据
	userStats := make(map[string]UserQuestionStat)
	if err := loadUserJSONData(req.UserID, questionStatsFile, &userStats); err != nil {
		log.Printf("错误: 用户 %s 加载统计数据失败 (答题提交): %v", req.UserID, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户统计数据失败"})
		return
	}

	statKey := fmt.Sprintf("%s_%s", originalQuestion.OriginalChapterKey, originalQuestion.QuestionNumber) // 统计文件中的键
	statEntry, statExists := userStats[statKey]
	if !statExists {
		statEntry = UserQuestionStat{
			OriginalChapterKey:     originalQuestion.OriginalChapterKey,
			OriginalQuestionNumber: originalQuestion.QuestionNumber,
		}
	}

	if req.WasCorrect {
		statEntry.CorrectCount++
	} else {
		statEntry.ErrorCount++
		// 如果答错，则记录到课程特定的错题本
		incorrectFileName := getIncorrectQuestionsFileName(currentCourse)
		userIncorrect := []UserIncorrectQuestion{}
		if err := loadUserJSONData(req.UserID, incorrectFileName, &userIncorrect); err != nil {
			log.Printf("错误: 用户 %s 加载错题本失败 (答题提交): %v", req.UserID, err)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户错题本失败"})
			return
		}

		// 检查是否重复添加 (基于题目文本和原始章节，避免同一道题记录多次)
		isDuplicate := false
		for _, iq := range userIncorrect {
			if iq.QuestionText == originalQuestion.QuestionText && iq.OriginalChapter == originalQuestion.OriginalChapterKey {
				isDuplicate = true
				log.Printf("信息: 用户 %s 题目 %s (章节 %s, 课程 %s) 已在错题本中，不再重复添加。", req.UserID, originalQuestion.QuestionNumber, originalQuestion.OriginalChapterKey, currentCourse)
				break
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
				UserAnswer:      req.UserAnswer, // 记录用户当时的错误答案
				Timestamp:       time.Now(),
			})
			if err := saveUserJSONData(req.UserID, incorrectFileName, userIncorrect); err != nil {
				log.Printf("错误: 用户 %s 保存错题本失败 (答题提交): %v", req.UserID, err)
				c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存用户错题本失败"})
				return
			}
			log.Printf("信息: 用户 %s 错题 %s (章节 %s) 已添加至错题本。", req.UserID, originalQuestion.QuestionNumber, originalQuestion.OriginalChapterKey)
		}
	}
	statEntry.LastAnswered = time.Now()
	userStats[statKey] = statEntry

	if err := saveUserJSONData(req.UserID, questionStatsFile, userStats); err != nil {
		log.Printf("错误: 用户 %s 保存统计数据失败 (答题提交): %v", req.UserID, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存用户统计数据失败"})
		return
	}

	log.Printf("用户 %s 答题模式提交: QID %s, 用户答案 %s, 是否正确 (前端判断): %t. 统计和错题记录已更新。", req.UserID, req.QuizQuestionID, req.UserAnswer, req.WasCorrect)
	c.JSON(consts.StatusOK, utils.H{
		"message": "答案已记录 (前端校验)",
		// 后端不再指示下一题或完成状态，前端基于其完整的题目列表进行管理
	})
}

// IncorrectQuestionsReviewStartHandler 处理开始错题回顾模式的请求。
// 返回用户的所有错题及其答案。
func IncorrectQuestionsReviewStartHandler(ctx context.Context, c *app.RequestContext) {
	var req StartIncorrectReviewRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	defer session.mu.Unlock()

	incorrectFileName := getIncorrectQuestionsFileName(req.Course)
	userIncorrectRaw := []UserIncorrectQuestion{}
	if err := loadUserJSONData(req.UserID, incorrectFileName, &userIncorrectRaw); err != nil {
		log.Printf("用户 %s 加载 %s 课程错题本失败 (回顾模式开始): %v", req.UserID, req.Course, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户错题本失败"})
		return
	}

	if len(userIncorrectRaw) == 0 {
		c.JSON(consts.StatusOK, utils.H{"message": "错题簿是空的哦！太棒了！", "total_questions": 0, "questions": []QuestionOutput{}})
		return
	}

	// 将错题随机打乱顺序
	rand.Shuffle(len(userIncorrectRaw), func(i, j int) {
		userIncorrectRaw[i], userIncorrectRaw[j] = userIncorrectRaw[j], userIncorrectRaw[i]
	})

	outputQuestions := convertUserIncorrectToOutput(userIncorrectRaw, 0, req.Course) // 转换为API输出格式
	session.CurrentMode = "incorrect_review"
	session.CurrentCourse = req.Course // 设置当前课程
	// 如果前端需要服务器逐步推送，则存储
	// session.CurrentQuestions = outputQuestions
	// session.CurrentQuestionIndex = 0

	log.Printf("用户 %s 开始错题回顾模式, 返回 %d 题", req.UserID, len(outputQuestions))
	c.JSON(consts.StatusOK, utils.H{
		"message":         "错题回顾模式开始",
		"total_questions": len(outputQuestions),
		"questions":       outputQuestions, // 发送所有错题给前端
	})
}

// SubmitIncorrectReviewAnswerHandler 处理用户在错题回顾中提交的答案。
// 由于前端已有答案并进行校验，此接口主要用于记录日志或未来可能的统计（当前仅日志）。
// 错题本身不会在此从错题本中移除，移除逻辑可能需要单独的接口或机制。
func SubmitIncorrectReviewAnswerHandler(ctx context.Context, c *app.RequestContext) {
	var req SubmitAnswerRequest // 复用 SubmitAnswerRequest 结构
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}

	// 此处不修改会话状态或持久化数据，因为前端已包含答案并进行校验。
	// 主要用于服务端日志记录，了解用户对错题的再次作答情况。
	log.Printf("用户 %s 错题回顾提交: QID %s, 用户答案 %s, 是否正确 (前端判断): %t. (仅记录日志)",
		req.UserID, req.QuizQuestionID, req.UserAnswer, req.WasCorrect)

	// 未来可以考虑：如果用户在回顾中答对了错题，是否从错题本中移除或标记。
	// 这需要更复杂的逻辑，例如解析 QuizQuestionID 找到原始错题记录并更新。

	c.JSON(consts.StatusOK, utils.H{
		"message": "错题回顾答案已由服务器记录(日志), 由前端校验正确性。",
	})
}

// DeleteIncorrectQuestionHandler 处理从错题本中删除特定题目的请求
func DeleteIncorrectQuestionHandler(ctx context.Context, c *app.RequestContext) {
	var req DeleteIncorrectQuestionRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}

	// 从 session 获取当前课程
	session := getOrCreateUserSession(req.UserID)
	session.mu.Lock()
	currentCourse := session.CurrentCourse
	session.mu.Unlock()

	if currentCourse == "" {
		log.Printf("警告: 用户 %s 的 session 中没有课程信息，默认使用毛概", req.UserID)
		currentCourse = "maogai"
	}

	// 加载课程特定的错题文件
	incorrectFileName := getIncorrectQuestionsFileName(currentCourse)
	userIncorrect := []UserIncorrectQuestion{}
	if err := loadUserJSONData(req.UserID, incorrectFileName, &userIncorrect); err != nil {
		log.Printf("错误: 用户 %s 删除错题时加载 %s 课程错题本失败: %v", req.UserID, currentCourse, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "加载用户错题本失败"})
		return
	}

	if len(userIncorrect) == 0 {
		log.Printf("信息: 用户 %s 请求删除错题，但错题本已为空。", req.UserID)
		c.Status(consts.StatusNoContent)
		return
	}

	var updatedIncorrect []UserIncorrectQuestion
	var foundAndDeleted bool
	var deletedQuestion UserIncorrectQuestion

	// 遍历现有错题，找出要删除的题目
	for _, iq := range userIncorrect {
		if iq.OriginalChapter == req.OriginalChapter && iq.QuestionNumber == req.OriginalQuestionNumber {
			foundAndDeleted = true
			deletedQuestion = iq
			// 保留原始答错时间
			// 新增删除时间标记
			deletedQuestion.DeletedAt = time.Now() // 记录删除时间
			log.Printf("信息: 用户 %s 从错题本中删除题目: 章节 %s, 题号 %s", req.UserID, req.OriginalChapter, req.OriginalQuestionNumber)
		} else {
			updatedIncorrect = append(updatedIncorrect, iq)
		}
	}

	// 仅当确实删除了题目时才更新文件
	if foundAndDeleted {
		// 保存更新后的课程特定错题本
		if err := saveUserJSONData(req.UserID, incorrectFileName, updatedIncorrect); err != nil {
			log.Printf("错误: 用户 %s 保存更新后的 %s 课程错题本失败: %v", req.UserID, currentCourse, err)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存更新后的错题本失败"})
			return
		}

		// 加载已删除错题历史
		deletedIncorrect := []UserIncorrectQuestion{}
		if err := loadUserJSONData(req.UserID, deleteIncorrectQuestionsFile, &deletedIncorrect); err != nil {
			log.Printf("警告: 用户 %s 加载已删除错题历史失败: %v", req.UserID, err)
			// 继续执行，可能是首次删除，文件不存在
		}

		// 将新删除的题目添加到历史记录中
		deletedIncorrect = append(deletedIncorrect, deletedQuestion)

		// 保存更新后的删除历史
		if err := saveUserJSONData(req.UserID, deleteIncorrectQuestionsFile, deletedIncorrect); err != nil {
			log.Printf("错误: 用户 %s 保存已删除错题历史失败: %v", req.UserID, err)
			// 不阻止主流程，因为主要操作（从错题本删除）已成功
		} else {
			log.Printf("信息: 用户 %s 的已删除错题已记录到历史文件中，删除时间为: %v", req.UserID, deletedQuestion.DeletedAt)
		}
	} else {
		log.Printf("警告: 用户 %s 请求删除错题 (章节 %s, 题号 %s)，但在错题本中未找到该题。", req.UserID, req.OriginalChapter, req.OriginalQuestionNumber)
	}

	// 按照要求，成功处理后不返回任何内容体
	c.Status(consts.StatusNoContent)
}

// UserDataClearHandler 处理清除用户数据的请求（错题本和统计数据）。
func UserDataClearHandler(ctx context.Context, c *app.RequestContext) {
	var req GetNextQuestionRequest // 仅为了获取 UserID
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "无效请求: " + err.Error()})
		return
	}

	userID := req.UserID
	log.Printf("用户 %s 请求清理其数据...", userID)

	// 清理毛概错题文件
	maogaiIncorrectPath := getUserDataPath(userID, maogaiIncorrectQuestionsFile)
	if _, err := os.Stat(maogaiIncorrectPath); err == nil { // 文件存在
		if err := os.Rename(maogaiIncorrectPath, maogaiIncorrectPath+time.Now().Format(".2006_01_02_15_04_05.bak")); err != nil {
			log.Printf("错误: 用户 %s 清理毛概错题文件 %s 失败: %v", userID, maogaiIncorrectPath, err)
		} else {
			log.Printf("信息: 用户 %s 的毛概错题文件 %s 已清理。", userID, maogaiIncorrectPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("错误: 检查毛概错题文件 %s 时发生错误: %v", maogaiIncorrectPath, err)
	}

	// 清理习概错题文件
	xigaiIncorrectPath := getUserDataPath(userID, xigaiIncorrectQuestionsFile)
	if _, err := os.Stat(xigaiIncorrectPath); err == nil { // 文件存在
		if err := os.Rename(xigaiIncorrectPath, xigaiIncorrectPath+time.Now().Format(".2006_01_02_15_04_05.bak")); err != nil {
			log.Printf("错误: 用户 %s 清理习概错题文件 %s 失败: %v", userID, xigaiIncorrectPath, err)
		} else {
			log.Printf("信息: 用户 %s 的习概错题文件 %s 已清理。", userID, xigaiIncorrectPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("错误: 检查习概错题文件 %s 时发生错误: %v", xigaiIncorrectPath, err)
	}

	// 清理统计文件
	statsPath := getUserDataPath(userID, questionStatsFile)
	if _, err := os.Stat(statsPath); err == nil { // 文件存在
		if err := os.Rename(statsPath, statsPath+time.Now().Format(".2006_01_02_15_04_05.bak")); err != nil {
			log.Printf("错误: 用户 %s 清理统计文件 %s 失败: %v", userID, statsPath, err)
			c.JSON(consts.StatusInternalServerError, utils.H{"error": "清理用户统计数据时发生部分或全部失败"})
			return // 如果统计文件清理失败，可能需要报告更严重的错误
		} else {
			log.Printf("信息: 用户 %s 的统计文件 %s 已清理。", userID, statsPath)
		}
	} else if !os.IsNotExist(err) { // 其他错误
		log.Printf("错误: 检查统计文件 %s 时发生错误: %v", statsPath, err)
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "检查用户统计数据时出错"})
		return
	}

	// 可选：从内存会话中清除用户会话，如果用户当前有活动会话
	sessionsMu.Lock()
	delete(userSessions, userID)
	sessionsMu.Unlock()
	log.Printf("信息: 用户 %s 的内存会话（如果存在）已清除。", userID)

	// 也可以考虑删除用户的主目录，但这取决于是否还有其他类型的数据
	// userDirPath := filepath.Join(userDataBaseDir, userID)
	// if err := os.RemoveAll(userDirPath); err != nil {
	//    log.Printf("警告: 用户 %s 清理主数据目录 %s 失败: %v", userID, userDirPath, err)
	// } else {
	//    log.Printf("信息: 用户 %s 的主数据目录 %s 已清理。", userID, userDirPath)
	// }

	c.JSON(consts.StatusOK, utils.H{"message": "用户数据（错题本和统计）已成功清理。"})
}
