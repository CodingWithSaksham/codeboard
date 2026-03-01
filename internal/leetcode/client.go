package leetcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// httpClient is a shared HTTP client configured for connection reuse.
//
// Explicit transport settings prevent file descriptor exhaustion
// when many goroutines execute concurrent refresh operations.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	},
}

// sendQuery executes a GraphQL query against LeetCode's API.
//
// It marshals the request body, performs the HTTP POST,
// validates the response status, and decodes the JSON payload.
//
// Returns:
//
//	map[string]any: Parsed JSON response
//	error:          If request, transport, or decoding fails
func sendQuery(query string, variables map[string]any) (map[string]any, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	// bytes.NewReader does not copy the slice; bytes.NewBuffer does.
	req, err := http.NewRequest(http.MethodPost, leetcodeGraphQL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	// REQUIRED: Without this, Leetcode's CSRF Middleware acts up, not allowing
	// us to query their GraphQL API
	req.Header.Add("Referrer-Policy", "same-origin")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-2xx status: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("json decode error: %w", err)
	}
	return result, nil
}

// FetchUserProfile retrieves a user's public profile information
// from LeetCode's GraphQL API.
//
// Params:
//
//	username: The LeetCode username whose public profile
//	          should be retrieved.
//
// Returns:
//
//	*UserProfile: Parsed profile data including ranking,
//	              avatar URL, and real name.
//	error:        If the user does not exist or the API call fails.
//
// The function validates the GraphQL response structure and
// extracts the nested matchedUser.profile object.
func FetchUserProfile(username string) (*UserProfile, error) {
	resp, err := sendQuery(matchedUserQuery, map[string]any{"username": username})
	if err != nil {
		return nil, err
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no data field in response")
	}
	matched, ok := data["matchedUser"].(map[string]any)
	if !ok || matched == nil {
		return nil, fmt.Errorf("matchedUser is nil – user may not exist on LeetCode")
	}
	profile, ok := matched["profile"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no profile field")
	}

	p := UserProfile{
		RealName:   profile["realName"].(string),
		UserAvatar: profile["userAvatar"].(string),
		Ranking:    profile["ranking"],
	}
	return &p, nil
}

// FetchSubmittedQuestions retrieves recent accepted submissions
// for a given user.
//
// Params:
//
//	username: LeetCode username.
//	limit:    Maximum number of recent accepted submissions
//	          to retrieve from the API.
//
// Returns:
//
//	[]SubmissionEntry: Slice of accepted submission records.
//	error:             If the API request fails.
//
// The function extracts titleSlug and timestamp fields from the
// recentAcSubmissionList response.
func FetchSubmittedQuestions(username string, limit int) ([]SubmissionEntry, error) {
	resp, err := sendQuery(questionsSubmittedQuery, map[string]any{
		"username": username, "limit": limit,
	})
	if err != nil {
		return nil, err
	}
	data := resp["data"].(map[string]any)
	list, _ := data["recentAcSubmissionList"].([]any)
	entries := make([]SubmissionEntry, 0, len(list))
	for _, item := range list {
		m := item.(map[string]any)
		entries = append(entries, SubmissionEntry{
			TitleSlug: m["titleSlug"].(string),
			Timestamp: m["timestamp"].(string),
		})
	}
	return entries, nil
}

// FetchLanguageProblemCount retrieves the number of problems solved
// per programming language for a given user.
//
// Params:
//
//	username: LeetCode username.
//
// Returns:
//
//	[]LanguageCount: Slice of language statistics.
//	error:           If the API request fails.
//
// The function extracts languageName and problemsSolved fields
// from the matchedUser.languageProblemCount response.
func FetchLanguageProblemCount(username string) ([]LanguageCount, error) {
	resp, err := sendQuery(languageProblemCountQuery, map[string]any{"username": username})
	if err != nil {
		return nil, err
	}

	data := resp["data"].(map[string]any)
	matched := data["matchedUser"].(map[string]any)
	list, _ := matched["languageProblemCount"].([]any)
	counts := make([]LanguageCount, 0, len(list))

	for _, item := range list {
		m := item.(map[string]any)
		counts = append(counts, LanguageCount{
			LanguageName:   m["languageName"].(string),
			ProblemsSolved: int(m["problemsSolved"].(float64)),
		})
	}
	return counts, nil
}

// FetchAllQuestions retrieves the full LeetCode question list.
//
// Returns:
//
//	[]QuestionListEntry: Complete list of question metadata.
//	error:               If the API request fails.
//
// Results are cached for qCacheTTL to prevent repeated
// large (≈5000 item) API calls during scheduled refresh jobs.
// If cached data is still fresh, it is returned immediately.
func FetchAllQuestions() ([]QuestionListEntry, error) {
	// Fast path: return cached list if still fresh.
	qCache.mu.RLock()
	if len(qCache.entries) > 0 && time.Since(qCache.fetchedAt) < qCacheTTL {
		entries := qCache.entries
		qCache.mu.RUnlock()
		return entries, nil
	}
	qCache.mu.RUnlock()

	resp, err := sendQuery(allQuestionListQuery, map[string]any{
		"categorySlug": "all-code-essentials",
		"skip":         0,
		"limit":        5000,
		"filters":      map[string]any{},
	})
	if err != nil {
		return nil, err
	}

	data := resp["data"].(map[string]any)
	pql := data["problemsetQuestionList"].(map[string]any)
	list, _ := pql["questions"].([]any)
	entries := make([]QuestionListEntry, 0, len(list))

	for _, item := range list {
		m := item.(map[string]any)
		entries = append(entries, QuestionListEntry{
			FrontendQuestionID: fmt.Sprintf("%v", m["frontendQuestionId"]),
			TitleSlug:          m["titleSlug"].(string),
		})
	}

	qCache.mu.Lock()
	qCache.entries = entries
	qCache.fetchedAt = time.Now()
	qCache.mu.Unlock()

	return entries, nil
}

// FetchAllUserData concurrently retrieves profile information,
// recent accepted submissions, and language statistics for a user.
//
// Params:
//
//	username: LeetCode username.
//
// Returns:
//
//	profile:     User profile information.
//	submissions: Recent accepted submissions.
//	langCounts:  Per-language solve statistics.
//	error:       If profile retrieval fails.
//
// The three API calls execute in parallel to reduce overall
// latency to the duration of the slowest individual request.
// Submission and language errors are logged but do not abort
// the entire operation unless profile retrieval fails.
func FetchAllUserData(username string) (
	profile *UserProfile,
	submissions []SubmissionEntry,
	langCounts []LanguageCount,
	err error,
) {
	type profileResult struct {
		v   *UserProfile
		err error
	}
	type subsResult struct {
		v   []SubmissionEntry
		err error
	}
	type langResult struct {
		v   []LanguageCount
		err error
	}

	profileCh := make(chan profileResult, 1)
	subsCh := make(chan subsResult, 1)
	langCh := make(chan langResult, 1)

	go func() {
		v, e := FetchUserProfile(username)
		profileCh <- profileResult{v, e}
	}()
	go func() {
		v, e := FetchSubmittedQuestions(username, 500)
		subsCh <- subsResult{v, e}
	}()
	go func() {
		v, e := FetchLanguageProblemCount(username)
		langCh <- langResult{v, e}
	}()

	pr := <-profileCh
	sr := <-subsCh
	lr := <-langCh

	if pr.err != nil {
		return nil, nil, nil, pr.err
	}
	if sr.err != nil {
		log.Printf("[leetcode] FetchSubmittedQuestions for %s: %v", username, sr.err)
	}
	if lr.err != nil {
		log.Printf("[leetcode] FetchLanguageProblemCount for %s: %v", username, lr.err)
	}
	return pr.v, sr.v, lr.v, nil
}

// ProcessSubmissions maps accepted submission entries to their
// latest accepted timestamp per question ID.
//
// Params:
//
//	submitted:    Slice of accepted submission entries returned
//	              from the LeetCode API.
//	allQuestions: Full question list used to map titleSlug → question ID.
//
// Returns:
//
//	map[int]int64: A map of questionID → latest accepted timestamp (Unix).
//
// The function builds a lookup table from titleSlug to numeric question ID,
// then performs a single pass over submissions to compute the latest
// timestamp per question. Intermediate allocations are avoided.
func ProcessSubmissions(submitted []SubmissionEntry, allQuestions []QuestionListEntry) map[int]int64 {
	titleSlugToID := make(map[string]int, len(allQuestions))
	for _, q := range allQuestions {
		id, err := strconv.Atoi(q.FrontendQuestionID)
		if err != nil {
			continue
		}
		titleSlugToID[q.TitleSlug] = id
	}

	latest := make(map[int]int64, len(submitted))
	for _, s := range submitted {
		id, ok := titleSlugToID[s.TitleSlug]
		if !ok {
			continue
		}
		ts, _ := strconv.ParseInt(s.Timestamp, 10, 64)
		if existing, seen := latest[id]; !seen || ts > existing {
			latest[id] = ts
		}
	}
	return latest
}

// MatchQuestionsToSolved filters questions that were solved
// after their publish timestamp.
//
// Params:
//
//	questions:         Slice of internally stored questions with
//	                   publish timestamps.
//	solvedSubmissions: Map of questionID → accepted timestamp.
//
// Returns:
//
//	map[int]int64: A map of questionID → accepted timestamp
//	               for questions solved after publication.
//
// A question is considered valid only if the submission timestamp
// is strictly greater than the question's publish timestamp.
func MatchQuestionsToSolved(questions []QuestionRecord, solvedSubmissions map[int]int64) map[int]int64 {
	matched := make(map[int]int64, len(questions))
	for _, q := range questions {
		if ts, ok := solvedSubmissions[q.LeetcodeID]; ok && ts > q.QuestionTimestamp {
			matched[q.LeetcodeID] = ts
		}
	}
	return matched
}

// CalSolvedIntervals categorizes solved questions into daily,
// weekly, and monthly intervals relative to the current time.
//
// Params:
//
//	questions:  Slice of question records with publish timestamps.
//	solvedDict: Map of questionID (string) → submission timestamp
//	            stored as dynamic JSON-compatible types.
//
// Returns:
//
//	day:   Map of questionID → timestamp solved within last 24 hours.
//	week:  Map of questionID → timestamp solved within last 7 days.
//	month: Map of questionID → timestamp solved within last 30 days.
//
// Only submissions that occurred after the question's publish
// timestamp are considered valid.
func CalSolvedIntervals(
	questions []QuestionRecord,
	solvedDict map[string]any,
) (
	day map[int]int64,
	week map[int]int64,
	month map[int]int64,
) {
	now := time.Now()
	oneDayAgo := now.Add(-24 * time.Hour).Unix()
	oneWeekAgo := now.Add(-7 * 24 * time.Hour).Unix()
	oneMonthAgo := now.Add(-30 * 24 * time.Hour).Unix()

	day = make(map[int]int64)
	week = make(map[int]int64)
	month = make(map[int]int64)

	for _, q := range questions {
		raw, ok := solvedDict[strconv.Itoa(q.LeetcodeID)]
		if !ok {
			continue
		}

		var ts int64
		switch v := raw.(type) {
		case float64:
			ts = int64(v)
		case int64:
			ts = v
		case json.Number:
			ts, _ = v.Int64()
		default:
			log.Printf("[leetcode] CalSolvedIntervals: unexpected type %T for question %d", raw, q.LeetcodeID)
			continue
		}

		if ts < q.QuestionTimestamp {
			continue // solved before it was posted — not eligible
		}
		if ts >= oneDayAgo {
			day[q.LeetcodeID] = ts
		}
		if ts >= oneWeekAgo {
			week[q.LeetcodeID] = ts
		}
		if ts >= oneMonthAgo {
			month[q.LeetcodeID] = ts
		}
	}
	return
}
