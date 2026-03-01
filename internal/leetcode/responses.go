package leetcode

import (
	"sync"
	"time"
)

// UserProfile represents the public profile information returned
// from LeetCode's GraphQL API.
type UserProfile struct {
	Ranking    any    `json:"ranking"`
	UserAvatar string `json:"userAvatar"`
	RealName   string `json:"realName"`
}

// SubmissionEntry represents a recent accepted submission.
type SubmissionEntry struct {
	TitleSlug string `json:"titleSlug"`
	Timestamp string `json:"timestamp"`
}

// LanguageCount represents the number of problems solved per language.
type LanguageCount struct {
	LanguageName   string `json:"languageName"`
	ProblemsSolved int    `json:"problemsSolved"`
}

// QuestionListEntry represents a question entry from the global
// LeetCode problem list.
type QuestionListEntry struct {
	FrontendQuestionID string `json:"frontendQuestionId"`
	TitleSlug          string `json:"titleSlug"`
}

// QuestionRecord represents a question stored internally
// with its publish timestamp.
type QuestionRecord struct {
	LeetcodeID        int
	TitleSlug         string
	QuestionTimestamp int64
}

// qCacheState stores cached question list data and its fetch timestamp.
type qCacheState struct {
	mu        sync.RWMutex
	entries   []QuestionListEntry
	fetchedAt time.Time
}
