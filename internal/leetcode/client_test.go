package leetcode

import (
	"encoding/json"
	"testing"
	"time"
)

// ── ProcessSubmissions ────────────────────────────────────────────────────────

// Tests that ProcessSubmissions maps titleSlugs to their correct
// question IDs using the allQuestions lookup table.
func TestProcessSubmissions_BasicMapping(t *testing.T) {
	questions := []QuestionListEntry{
		{FrontendQuestionID: "1", TitleSlug: "two-sum"},
		{FrontendQuestionID: "2", TitleSlug: "add-two-numbers"},
	}
	submitted := []SubmissionEntry{
		{TitleSlug: "two-sum", Timestamp: "1700000000"},
		{TitleSlug: "add-two-numbers", Timestamp: "1700000100"},
	}
	result := ProcessSubmissions(submitted, questions)

	if ts := result[1]; ts != 1700000000 {
		t.Errorf("question 1: expected ts 1700000000, got %d", ts)
	}
	if ts := result[2]; ts != 1700000100 {
		t.Errorf("question 2: expected ts 1700000100, got %d", ts)
	}
}

// Tests that ProcessSubmissions keeps the latest timestamp when
// a question appears multiple times in the submissions list.
func TestProcessSubmissions_KeepsLatestTimestamp(t *testing.T) {
	questions := []QuestionListEntry{
		{FrontendQuestionID: "1", TitleSlug: "two-sum"},
	}
	submitted := []SubmissionEntry{
		{TitleSlug: "two-sum", Timestamp: "1000"},
		{TitleSlug: "two-sum", Timestamp: "9999"}, // later
		{TitleSlug: "two-sum", Timestamp: "500"},
	}
	result := ProcessSubmissions(submitted, questions)
	if ts := result[1]; ts != 9999 {
		t.Errorf("expected latest ts 9999, got %d", ts)
	}
}

// Tests that ProcessSubmissions ignores submitted questions whose
// titleSlug does not appear in the allQuestions list.
func TestProcessSubmissions_UnknownSlugIgnored(t *testing.T) {
	questions := []QuestionListEntry{
		{FrontendQuestionID: "1", TitleSlug: "known-slug"},
	}
	submitted := []SubmissionEntry{
		{TitleSlug: "unknown-slug", Timestamp: "12345"},
	}
	result := ProcessSubmissions(submitted, questions)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

// Tests that ProcessSubmissions returns an empty map when
// given empty input slices.
func TestProcessSubmissions_EmptyInputs(t *testing.T) {
	result := ProcessSubmissions(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// Tests that ProcessSubmissions skips questions with non-numeric
// FrontendQuestionIDs without panicking.
func TestProcessSubmissions_SkipsInvalidQuestionID(t *testing.T) {
	questions := []QuestionListEntry{
		{FrontendQuestionID: "not-a-number", TitleSlug: "some-slug"},
	}
	submitted := []SubmissionEntry{
		{TitleSlug: "some-slug", Timestamp: "12345"},
	}
	result := ProcessSubmissions(submitted, questions)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// ── MatchQuestionsToSolved ────────────────────────────────────────────────────

// Tests that MatchQuestionsToSolved includes a question only when
// the submission timestamp is strictly after the question's publish timestamp.
func TestMatchQuestionsToSolved_SolvedAfterPublish(t *testing.T) {
	questions := []QuestionRecord{
		{LeetcodeID: 1, QuestionTimestamp: 1000},
	}
	solved := map[int]int64{1: 2000} // solved after publish
	result := MatchQuestionsToSolved(questions, solved)
	if _, ok := result[1]; !ok {
		t.Error("expected question 1 to be matched")
	}
}

// Tests that MatchQuestionsToSolved excludes a question when its
// submission timestamp is before or equal to the publish timestamp.
func TestMatchQuestionsToSolved_SolvedBeforePublish_Excluded(t *testing.T) {
	questions := []QuestionRecord{
		{LeetcodeID: 2, QuestionTimestamp: 5000},
		{LeetcodeID: 3, QuestionTimestamp: 3000},
	}
	solved := map[int]int64{
		2: 4999, // before publish
		3: 3000, // equal — not strictly greater
	}
	result := MatchQuestionsToSolved(questions, solved)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

// Tests that MatchQuestionsToSolved returns an empty map when
// there are no matching question IDs in the solved set.
func TestMatchQuestionsToSolved_NoOverlap(t *testing.T) {
	questions := []QuestionRecord{
		{LeetcodeID: 10, QuestionTimestamp: 100},
	}
	solved := map[int]int64{99: 500} // different ID
	result := MatchQuestionsToSolved(questions, solved)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

// ── CalSolvedIntervals ────────────────────────────────────────────────────────

// Tests that CalSolvedIntervals correctly places a question into the
// daily bucket when solved within the last 24 hours.
func TestCalSolvedIntervals_DailyBucket(t *testing.T) {
	now := time.Now()
	recentTS := now.Add(-1 * time.Hour).Unix() // 1 hour ago
	q := QuestionRecord{LeetcodeID: 1, QuestionTimestamp: now.Add(-2 * time.Hour).Unix()}
	solvedDict := map[string]any{"1": float64(recentTS)}

	day, week, month := CalSolvedIntervals([]QuestionRecord{q}, solvedDict)

	if _, ok := day[1]; !ok {
		t.Error("expected question 1 in daily bucket")
	}
	if _, ok := week[1]; !ok {
		t.Error("expected question 1 in weekly bucket")
	}
	if _, ok := month[1]; !ok {
		t.Error("expected question 1 in monthly bucket")
	}
}

// Tests that CalSolvedIntervals places a question in weekly and monthly
// buckets only when solved more than 24h ago but within 7 days.
func TestCalSolvedIntervals_WeeklyNotDaily(t *testing.T) {
	now := time.Now()
	ts := now.Add(-3 * 24 * time.Hour).Unix() // 3 days ago
	q := QuestionRecord{LeetcodeID: 2, QuestionTimestamp: now.Add(-4 * 24 * time.Hour).Unix()}
	solvedDict := map[string]any{"2": float64(ts)}

	day, week, month := CalSolvedIntervals([]QuestionRecord{q}, solvedDict)

	if _, ok := day[2]; ok {
		t.Error("question 2 should NOT be in daily bucket")
	}
	if _, ok := week[2]; !ok {
		t.Error("expected question 2 in weekly bucket")
	}
	if _, ok := month[2]; !ok {
		t.Error("expected question 2 in monthly bucket")
	}
}

// Tests that CalSolvedIntervals excludes a question solved before
// its QuestionTimestamp (pre-publish solve).
func TestCalSolvedIntervals_SolvedBeforePublish_Excluded(t *testing.T) {
	now := time.Now()
	publish := now.Add(-1 * time.Hour).Unix()
	solvedTS := now.Add(-2 * time.Hour).Unix() // solved before publish
	q := QuestionRecord{LeetcodeID: 5, QuestionTimestamp: publish}
	solvedDict := map[string]any{"5": float64(solvedTS)}

	day, week, month := CalSolvedIntervals([]QuestionRecord{q}, solvedDict)

	if _, ok := day[5]; ok {
		t.Error("question 5 should NOT be in day bucket (solved before publish)")
	}
	if _, ok := week[5]; ok {
		t.Error("question 5 should NOT be in week bucket (solved before publish)")
	}
	if _, ok := month[5]; ok {
		t.Error("question 5 should NOT be in month bucket (solved before publish)")
	}
}

// Tests that CalSolvedIntervals handles json.Number type in solvedDict,
// correctly parsing it as an int64 timestamp.
func TestCalSolvedIntervals_JSONNumberType(t *testing.T) {
	now := time.Now()
	ts := now.Add(-1 * time.Hour).Unix()
	q := QuestionRecord{LeetcodeID: 7, QuestionTimestamp: now.Add(-2 * time.Hour).Unix()}
	solvedDict := map[string]any{"7": json.Number(string(rune('0' + ts%10)))} // use json.Number

	// Reconstruct with valid json.Number.
	solvedDict["7"] = json.Number(itoa64(ts))

	day, _, _ := CalSolvedIntervals([]QuestionRecord{q}, solvedDict)
	if _, ok := day[7]; !ok {
		t.Error("expected question 7 in daily bucket when solvedDict value is json.Number")
	}
}

// Tests that CalSolvedIntervals skips entries in solvedDict with
// an unrecognized type without panicking.
func TestCalSolvedIntervals_UnknownType_Skipped(t *testing.T) {
	q := QuestionRecord{LeetcodeID: 9, QuestionTimestamp: 100}
	solvedDict := map[string]any{"9": "not-a-number"}

	// Should not panic.
	day, week, month := CalSolvedIntervals([]QuestionRecord{q}, solvedDict)
	if len(day)+len(week)+len(month) != 0 {
		t.Error("unexpected entries in buckets for unknown type")
	}
}

// itoa64 converts an int64 to its decimal string representation.
func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
