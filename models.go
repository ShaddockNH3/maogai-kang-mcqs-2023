package main

import (
	"sync"
	"time"
)

// --- 配置常量 ---
const (
	maogaiQuestionSourceDir      = "clean_outputs/maogai_outputs"
	xigaiQuestionSourceDir       = "clean_outputs/xigai_outputs"
	maogaiMaxChapterIndex        = 8
	xigaiMaxChapterIndex         = 0 // 习概只有一个章节
	userDataBaseDir              = "user_data"
	incorrectQuestionsFile       = "incorrect_questions.json"        // 默认(毛概)错题文件
	maogaiIncorrectQuestionsFile = "maogai_incorrect_questions.json" // 毛概错题文件
	xigaiIncorrectQuestionsFile  = "xigai_incorrect_questions.json"  // 习概错题文件
	deleteIncorrectQuestionsFile = "deleted_incorrect_questions.json"
	questionStatsFile            = "question_stats.json"
)

// --- 数据结构定义 ---
type Question struct {
	QuestionNumber     string            `json:"question_number"`
	QuestionType       string            `json:"question_type"`
	QuestionText       string            `json:"question_text"`
	Options            map[string]string `json:"options"`
	CorrectAnswer      string            `json:"correct_answer"`
	GlobalCorrectCount int               `json:"correct_count"` // 未来可能用于全局统计
	GlobalErrorCount   int               `json:"error_count"`   // 未来可能用于全局统计
	OriginalChapterKey string            `json:"-"`             // 内部使用，标记原始章节
	OriginalIndex      int               `json:"-"`             // 内部使用，标记在原始章节中的索引
}

type QuestionOutput struct {
	QuizQuestionID         string            `json:"quiz_question_id"`         // 在当前测验/回顾中的唯一ID
	DisplayNumber          int               `json:"display_number"`           // 在当前列表中的显示序号 (1-based)
	OriginalChapter        string            `json:"original_chapter"`         // 原始章节键
	OriginalQuestionNumber string            `json:"original_question_number"` // 原始题号
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
	UserAnswer      string            `json:"user_answer,omitempty"` // 用户在答错时的答案
	Timestamp       time.Time         `json:"timestamp"`             // 答错的时间
	DeletedAt       time.Time         `json:"deleted_at,omitempty"`  // 题目被删除的时间
}

type UserQuestionStat struct {
	OriginalChapterKey     string    `json:"original_chapter_key"`
	OriginalQuestionNumber string    `json:"original_question_number"`
	CorrectCount           int       `json:"correct_count"`
	ErrorCount             int       `json:"error_count"`
	LastAnswered           time.Time `json:"last_answered"`
}

// UserSession 存储用户当前会话状态
type UserSession struct {
	UserID string
	// CurrentQuestions is used if frontend logic relies on server to step through questions,
	// e.g. for a simplified /api/review/next. If frontend receives all questions
	// from /start endpoints and manages navigation itself, this might be less critical
	// for those modes. For this iteration, we keep it for potential use with /api/review/next.
	CurrentQuestions     []QuestionOutput
	OriginalIncorrect    []UserIncorrectQuestion // Store the full incorrect questions for retrieval
	CurrentQuestionIndex int                     // Index for session.CurrentQuestions (e.g., /api/review/next)
	CurrentMode          string                  // "review", "quiz", "incorrect_review"
	CurrentCourse        string                  // "maogai" or "xigai" - 当前选择的课程
	mu                   sync.Mutex              // 保护会话内部数据
}

// --- 请求结构体 ---
type InitSessionRequest struct {
	UserID string `json:"user_id" vd:"required,min=1,max=50"` // UserID是必需的,长度1-50
}

type StartModeRequest struct {
	UserID        string   `json:"user_id" vd:"required"`
	Course        string   `json:"course" vd:"required"`         // "maogai" 或 "xigai"
	ChapterChoice []string `json:"chapter_choice" vd:"required"` // 例如 ["0", "1", "all"]
	OrderChoice   string   `json:"order_choice" vd:"required"`   // "sequential" 或 "random"
}

type GetNextQuestionRequest struct {
	UserID string `json:"user_id" vd:"required"`
}

type StartIncorrectReviewRequest struct {
	UserID string `json:"user_id" vd:"required"`
	Course string `json:"course" vd:"required"`
}

type SubmitAnswerRequest struct {
	UserID         string `json:"user_id" vd:"required"`
	QuizQuestionID string `json:"quiz_question_id" vd:"required"` // 题目在当前测验中的ID
	UserAnswer     string `json:"user_answer" vd:"required"`      // 用户选择的答案
	WasCorrect     bool   `json:"was_correct"`                    // 由前端判断并发送该答案是否正确
}

type DeleteIncorrectQuestionRequest struct {
	UserID                 string `json:"user_id" vd:"required"`
	OriginalChapter        string `json:"original_chapter" vd:"required"`
	OriginalQuestionNumber string `json:"original_question_number" vd:"required"`
}
