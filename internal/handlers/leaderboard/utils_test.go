package leaderboard

import (
	"testing"
	"time"

	"github.com/creative-computing-society/codeboard/internal/models"
)

// Tests that serializeLeetcode maps all fields from a Leetcode model
// to a leetcodeResponse correctly.
func TestSerializeLeetcode_FieldMapping(t *testing.T) {
	acc := models.Leetcode{
		Username:     "alice",
		Name:         "Alice Smith",
		LeetcodeRank: "1234",
		DailyRank:    1,
		WeeklyRank:   2,
		MonthlyRank:  3,
		PhotoURL:     "http://example.com/photo.png",
		SubmissionDict: map[string]any{},
	}
	resp := serializeLeetcode(acc)

	if resp.Username != "alice" {
		t.Errorf("Username: expected %q, got %q", "alice", resp.Username)
	}
	if resp.Name != "Alice Smith" {
		t.Errorf("Name: expected %q, got %q", "Alice Smith", resp.Name)
	}
	if resp.LeetcodeRank != "1234" {
		t.Errorf("LeetcodeRank: expected %q, got %q", "1234", resp.LeetcodeRank)
	}
	if resp.DailyRank != 1 {
		t.Errorf("DailyRank: expected 1, got %d", resp.DailyRank)
	}
	if resp.WeeklyRank != 2 {
		t.Errorf("WeeklyRank: expected 2, got %d", resp.WeeklyRank)
	}
	if resp.MonthlyRank != 3 {
		t.Errorf("MonthlyRank: expected 3, got %d", resp.MonthlyRank)
	}
	if resp.PhotoURL != "http://example.com/photo.png" {
		t.Errorf("PhotoURL: expected %q, got %q", "http://example.com/photo.png", resp.PhotoURL)
	}
}

// Tests that serializeLeetcode converts float64 timestamps in
// SubmissionDict into human-readable "2006-01-02 15:04:05" strings.
func TestSerializeLeetcode_SubmissionDictFloat64Timestamp(t *testing.T) {
	ts := int64(1700000000)
	acc := models.Leetcode{
		SubmissionDict: map[string]any{
			"two-sum": float64(ts),
		},
	}
	resp := serializeLeetcode(acc)
	expected := time.Unix(ts, 0).Format("2006-01-02 15:04:05")
	if got := resp.Submissions["two-sum"]; got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// Tests that serializeLeetcode handles int64 timestamps in
// SubmissionDict and skips entries with zero timestamp.
func TestSerializeLeetcode_SubmissionDictInt64AndZero(t *testing.T) {
	ts := int64(1700000000)
	acc := models.Leetcode{
		SubmissionDict: map[string]any{
			"valid":   ts,
			"zero_ts": int64(0),
		},
	}
	resp := serializeLeetcode(acc)
	if _, ok := resp.Submissions["zero_ts"]; ok {
		t.Error("zero timestamp should be excluded from submissions map")
	}
	if _, ok := resp.Submissions["valid"]; !ok {
		t.Error("valid timestamp should be present in submissions map")
	}
}

// Tests that buildSolvedSet creates a constant-time lookup set
// with all keys from MatchedQuesDict.
func TestBuildSolvedSet_ContainsAllKeys(t *testing.T) {
	acc := &models.Leetcode{
		MatchedQuesDict: map[string]any{
			"1": struct{}{},
			"2": struct{}{},
			"3": struct{}{},
		},
	}
	set := buildSolvedSet(acc)
	for _, k := range []string{"1", "2", "3"} {
		if _, ok := set[k]; !ok {
			t.Errorf("key %q missing from solved set", k)
		}
	}
}

// Tests that buildSolvedSet returns an empty map when
// MatchedQuesDict is nil or empty.
func TestBuildSolvedSet_EmptyDict(t *testing.T) {
	acc := &models.Leetcode{MatchedQuesDict: nil}
	set := buildSolvedSet(acc)
	if len(set) != 0 {
		t.Errorf("expected empty set, got %d entries", len(set))
	}
}

// Tests that serializeQuestion marks a question as "Solved" when
// its LeetcodeID is present in the solved set.
func TestSerializeQuestion_SolvedStatus(t *testing.T) {
	q := models.Question{
		LeetcodeID:   1,
		Title:        "Two Sum",
		TitleSlug:    "two-sum",
		QuestionDate: time.Now(),
		Difficulty:   "Easy",
	}
	solved := map[string]struct{}{"1": {}}
	resp := serializeQuestion(q, solved)
	if resp.Status != "Solved" {
		t.Errorf("expected Solved, got %q", resp.Status)
	}
}

// Tests that serializeQuestion marks a question as "Not Solved"
// when its LeetcodeID is absent from the solved set.
func TestSerializeQuestion_NotSolvedStatus(t *testing.T) {
	q := models.Question{
		LeetcodeID:   99,
		Title:        "Median of Two Sorted Arrays",
		TitleSlug:    "median-of-two-sorted-arrays",
		QuestionDate: time.Now(),
		Difficulty:   "Hard",
	}
	resp := serializeQuestion(q, map[string]struct{}{})
	if resp.Status != "Not Solved" {
		t.Errorf("expected Not Solved, got %q", resp.Status)
	}
}

// Tests that serializeQuestion constructs the correct LeetCode URL
// from the question's TitleSlug field.
func TestSerializeQuestion_URLConstruction(t *testing.T) {
	q := models.Question{
		LeetcodeID:   1,
		TitleSlug:    "two-sum",
		QuestionDate: time.Now(),
	}
	resp := serializeQuestion(q, map[string]struct{}{})
	want := "https://leetcode.com/problems/two-sum/"
	if resp.LeetcodeLink != want {
		t.Errorf("expected %q, got %q", want, resp.LeetcodeLink)
	}
}

// Tests that serializeQuestion formats the QuestionDate
// as "2006-01-02".
func TestSerializeQuestion_DateFormat(t *testing.T) {
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	q := models.Question{
		LeetcodeID:   5,
		QuestionDate: date,
	}
	resp := serializeQuestion(q, map[string]struct{}{})
	if resp.QuestionDate != "2024-06-15" {
		t.Errorf("expected %q, got %q", "2024-06-15", resp.QuestionDate)
	}
}

// Tests that serializeQuestionPublic omits the Status field and
// correctly maps all other question fields.
func TestSerializeQuestionPublic_NoStatus(t *testing.T) {
	q := models.Question{
		LeetcodeID:   3,
		Title:        "Longest Substring Without Repeating Characters",
		TitleSlug:    "longest-substring-without-repeating-characters",
		QuestionDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Difficulty:   "Medium",
	}
	resp := serializeQuestionPublic(q)
	if resp.Status != "" {
		t.Errorf("expected empty status, got %q", resp.Status)
	}
	if resp.Title != q.Title {
		t.Errorf("Title mismatch: %q vs %q", q.Title, resp.Title)
	}
	if resp.Difficulty != "Medium" {
		t.Errorf("Difficulty mismatch: %q vs %q", "Medium", resp.Difficulty)
	}
}

// Tests that serializeQuestionPublic constructs the correct URL
// for a public question response without user context.
func TestSerializeQuestionPublic_URLConstruction(t *testing.T) {
	q := models.Question{TitleSlug: "add-two-numbers", QuestionDate: time.Now()}
	resp := serializeQuestionPublic(q)
	want := "https://leetcode.com/problems/add-two-numbers/"
	if resp.LeetcodeLink != want {
		t.Errorf("expected %q, got %q", want, resp.LeetcodeLink)
	}
}
