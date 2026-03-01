package leaderboard

import (
	"fmt"
	"strconv"
	"time"

	"github.com/creative-computing-society/codeboard/internal/models"
)

// leetcodeResponse represents the serialized response returned
// for a linked LeetCode account.
type leetcodeResponse struct {
	Username     string            `json:"username"`
	Name         string            `json:"name"`
	LeetcodeRank string            `json:"leetcode_rank"`
	DailyRank    int               `json:"daily_rank"`
	WeeklyRank   int               `json:"weekly_rank"`
	MonthlyRank  int               `json:"monthly_rank"`
	PhotoURL     string            `json:"photo_url"`
	Submissions  map[string]string `json:"submissions"`
}

// serializeLeetcode converts a models.Leetcode entity into a
// leetcodeResponse suitable for JSON output.
//
// It formats submission timestamps into human-readable strings
// and avoids unnecessary JSON re-marshalling for performance.
func serializeLeetcode(acc models.Leetcode) leetcodeResponse {
	submissions := make(map[string]string, len(acc.SubmissionDict))

	for k, v := range acc.SubmissionDict {
		var ts int64

		switch tv := v.(type) {
		case float64:
			ts = int64(tv)
		case int64:
			ts = tv
		}

		if ts > 0 {
			submissions[k] = time.Unix(ts, 0).
				Format("2006-01-02 15:04:05")
		}
	}

	return leetcodeResponse{
		Username:     acc.Username,
		Name:         acc.Name,
		LeetcodeRank: acc.LeetcodeRank,
		DailyRank:    acc.DailyRank,
		WeeklyRank:   acc.WeeklyRank,
		MonthlyRank:  acc.MonthlyRank,
		PhotoURL:     acc.PhotoURL,
		Submissions:  submissions,
	}
}

// questionResponse represents the serialized response structure
// for a coding question.
type questionResponse struct {
	LeetcodeID   int    `json:"leetcode_id"`
	Title        string `json:"title"`
	LeetcodeLink string `json:"leetcode_link"`
	QuestionDate string `json:"questionDate"`
	Difficulty   string `json:"difficulty"`
	Status       string `json:"status,omitempty"`
}

// buildSolvedSet converts MatchedQuesDict into a constant-time
// lookup set.
//
// It is called once per request to avoid repeated database
// queries when computing question solve status.
func buildSolvedSet(acc *models.Leetcode) map[string]struct{} {
	set := make(map[string]struct{}, len(acc.MatchedQuesDict))

	for k := range acc.MatchedQuesDict {
		set[k] = struct{}{}
	}

	return set
}

// serializeQuestion converts a Question model into a questionResponse,
// including the user's solve status.
func serializeQuestion(
	q models.Question,
	solved map[string]struct{},
) questionResponse {
	status := "Not Solved"
	if _, ok := solved[strconv.Itoa(q.LeetcodeID)]; ok {
		status = "Solved"
	}

	return questionResponse{
		LeetcodeID:   q.LeetcodeID,
		Title:        q.Title,
		LeetcodeLink: fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug),
		QuestionDate: q.QuestionDate.Format("2006-01-02"),
		Difficulty:   string(q.Difficulty),
		Status:       status,
	}
}

// serializeQuestionPublic converts a Question model into a
// questionResponse without including user-specific solve status.
func serializeQuestionPublic(q models.Question) questionResponse {
	return questionResponse{
		LeetcodeID:   q.LeetcodeID,
		Title:        q.Title,
		LeetcodeLink: fmt.Sprintf("https://leetcode.com/problems/%s/", q.TitleSlug),
		QuestionDate: q.QuestionDate.Format("2006-01-02"),
		Difficulty:   string(q.Difficulty),
	}
}
