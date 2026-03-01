package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CUser - password field removed; identity is now fully managed by Google OAuth.
// GoogleID is the stable "sub" claim from Google's ID token.
type CUser struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                    json:"-"`
	GoogleID  string         `gorm:"uniqueIndex;not null"     json:"google_id"`
	Email     string         `gorm:"uniqueIndex;not null"     json:"email"`
	Username  string         `gorm:"not null"                 json:"username"`
	AvatarURL string         `gorm:"default:''"               json:"avatar_url"`
	IsAdmin   bool           `gorm:"default:false"            json:"is_admin"`
}

// Leetcode mirrors the Django Leetcode model exactly.
type Leetcode struct {
	ID              uint              `gorm:"primaryKey;autoIncrement"                      json:"id"`
	UserID          uint              `gorm:"not null;uniqueIndex"                                   json:"user_id"`
	User            CUser             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Username        string            `gorm:"size:100;not null;uniqueIndex;default:''"      json:"username"`
	Name            string            `gorm:"size:100;default:'Scraping..'"                 json:"name"`
	LeetcodeRank    string            `gorm:"size:20;default:'Scraping..'"                  json:"leetcode_rank"`
	DailyRank       int               `gorm:"default:0"                                     json:"daily_rank"`
	WeeklyRank      int               `gorm:"default:0"                                     json:"weekly_rank"`
	MonthlyRank     int               `gorm:"default:0"                                     json:"monthly_rank"`
	PhotoURL        string            `gorm:"size:200;default:'Scraping..'"                 json:"photo_url"`
	TotalSolved     int               `gorm:"default:0"                                     json:"total_solved"`
	MatchedQues     int               `gorm:"default:0"                                     json:"matched_ques"`
	SubmissionDict  datatypes.JSONMap `gorm:"type:jsonb;default:'{}'" json:"submission_dict"`
	TotalSolvedDict datatypes.JSONMap `gorm:"type:jsonb;default:'{}'" json:"total_solved_dict"`
	MatchedQuesDict datatypes.JSONMap `gorm:"type:jsonb;default:'{}'" json:"matched_ques_dict"`
}

// Difficulty mirrors Django choices (including original typo).
type Difficulty string

const (
	DifficultyBasic        Difficulty = "Basic"
	DifficultyIntermediate Difficulty = "Intermidiate" // original typo preserved
	DifficultyAdvanced     Difficulty = "Advanced"
)

// Question mirrors the Django Question model exactly.
type Question struct {
	QuestionKey  uint       `gorm:"primaryKey;autoIncrement"         json:"question_key"`
	LeetcodeID   int        `gorm:"not null;default:0"               json:"leetcode_id"`
	Title        string     `gorm:"size:100;not null;default:''"     json:"title"`
	TitleSlug    string     `gorm:"size:100;not null;default:''"     json:"title_slug"`
	QuestionDate time.Time  `gorm:"not null"                         json:"question_date"`
	Difficulty   Difficulty `gorm:"size:20;not null;default:'Basic'" json:"difficulty"`
}

// LeaderboardInterval mirrors Django choices.
type LeaderboardInterval string

const (
	IntervalDay   LeaderboardInterval = "day"
	IntervalWeek  LeaderboardInterval = "week"
	IntervalMonth LeaderboardInterval = "month"
)

// LeaderboardEntry mirrors the Django LeaderboardEntry model.
type LeaderboardEntry struct {
	ID                      uint                `gorm:"primaryKey;autoIncrement"`
	UserID                  uint                `gorm:"not null;uniqueIndex:idx_user_interval"`
	User                    CUser               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Interval                LeaderboardInterval `gorm:"size:10;not null;uniqueIndex:idx_user_interval"`
	QuestionsSolved         int                 `gorm:"not null;default:0"`
	EarliestSolvedTimestamp int64               `gorm:"not null;default:0"`
}

// LeaderboardType mirrors Django choices.
type LeaderboardType string

const (
	LeaderboardDaily   LeaderboardType = "daily"
	LeaderboardWeekly  LeaderboardType = "weekly"
	LeaderboardMonthly LeaderboardType = "monthly"
)

// Leaderboard mirrors the Django Leaderboard model.
type Leaderboard struct {
	LeaderboardKey  uint            `gorm:"primaryKey;autoIncrement"                      json:"leaderboard_key"`
	LeaderboardType LeaderboardType `gorm:"size:20;not null;uniqueIndex;default:'daily'"  json:"leaderboard_type"`
	LeaderboardData datatypes.JSON  `gorm:"type:jsonb;default:'{}'"                       json:"leaderboard_data"`
}
