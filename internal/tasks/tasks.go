package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creative-computing-society/codeboard/internal/models"

	lc "github.com/creative-computing-society/codeboard/internal/leetcode"

	"github.com/creative-computing-society/codeboard/internal/db"

	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

// ── types ─────────────────────────────────────────────────────────────────────

type userRow struct {
	Username  string
	Solved    int
	Timestamp int64
}

type LeaderboardEntryData struct {
	Username string `json:"username"`
	PhotoURL string `json:"photo_url"`
	QuesSolv int    `json:"ques_solv"`
	LastSolv string `json:"last_solv"`
}

// ── GetUserData ───────────────────────────────────────────────────────────────

func GetUserData(username string, id uint) {
	log.Printf("[task] GetUserData started for %s (id=%d)", username, id)

	// Three LeetCode API calls run concurrently instead of sequentially.
	// Old: ~90s worst case (3 × 30s timeout in series).
	// New: ~30s worst case (all three race in parallel).
	userProfile, submittedQuestions, languageCounts, err := lc.FetchAllUserData(username)
	if err != nil {
		log.Printf("[task] GetUserData: fetch failed for %s: %v", username, err)
		return
	}

	// FetchAllQuestions uses an in-memory 1-hour cache — during RefreshUserData
	// with N users this is fetched once and returned from cache N-1 times.
	allQuestions, err := lc.FetchAllQuestions()
	if err != nil {
		log.Printf("[task] GetUserData: FetchAllQuestions failed: %v", err)
	}

	// Single pass: submitted → map[questionID]latestTimestamp (merged two passes).
	latestSolved := lc.ProcessSubmissions(submittedQuestions, allQuestions)

	var instance models.Leetcode
	if err := db.DB.First(&instance, id).Error; err != nil {
		log.Printf("[task] GetUserData: instance not found for id %d: %v", id, err)
		return
	}

	// JSONMap is already map[string]any in memory — copy it directly.
	// Old code did json.Marshal(instance.TotalSolvedDict) → json.Unmarshal to
	// get an identical map back, allocating and discarding all the bytes.
	totalSolvedDict := make(map[string]any, len(instance.TotalSolvedDict)+len(latestSolved))
	maps.Copy(totalSolvedDict, instance.TotalSolvedDict)

	for qID, ts := range latestSolved {
		key := strconv.Itoa(qID)
		if _, exists := totalSolvedDict[key]; !exists {
			totalSolvedDict[key] = ts
		}
	}

	var dbQuestions []models.Question
	db.DB.Select("leetcode_id", "title_slug", "question_date").Find(&dbQuestions)
	qRecords := make([]lc.QuestionRecord, len(dbQuestions))
	for i, q := range dbQuestions {
		qRecords[i] = lc.QuestionRecord{
			LeetcodeID:        q.LeetcodeID,
			TitleSlug:         q.TitleSlug,
			QuestionTimestamp: q.QuestionDate.Unix(),
		}
	}

	solvedIntMap := make(map[int]int64, len(totalSolvedDict))
	for k, v := range totalSolvedDict {
		id, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		switch tv := v.(type) {
		case float64:
			solvedIntMap[id] = int64(tv)
		case int64:
			solvedIntMap[id] = tv
		}
	}
	matchedQues := lc.MatchQuestionsToSolved(qRecords, solvedIntMap)

	totalSolvedCount := 0
	for _, c := range languageCounts {
		totalSolvedCount += c.ProblemsSolved
	}

	// Build JSONMaps directly without marshal/unmarshal roundtrips.
	submissionMap := make(datatypes.JSONMap, len(latestSolved))
	for k, v := range latestSolved {
		submissionMap[strconv.Itoa(k)] = v
	}
	totalSolvedMap := make(datatypes.JSONMap, len(totalSolvedDict))
	maps.Copy(totalSolvedMap, totalSolvedDict)

	matchedMap := make(datatypes.JSONMap, len(matchedQues))
	for k, v := range matchedQues {
		matchedMap[strconv.Itoa(k)] = v
	}

	rankStr := ""
	if userProfile.Ranking != nil {
		rankStr = fmt.Sprintf("%v", userProfile.Ranking)
	}

	// Targeted UPDATE — only the columns that actually changed, not all columns.
	// db.DB.Save(&instance) issues UPDATE with every field including the large
	// JSONB blobs even when they haven't changed.
	if err := db.DB.Model(&instance).Updates(map[string]any{
		"name":              userProfile.RealName,
		"leetcode_rank":     rankStr,
		"photo_url":         userProfile.UserAvatar,
		"total_solved":      totalSolvedCount,
		"matched_ques":      len(matchedQues),
		"submission_dict":   submissionMap,
		"total_solved_dict": totalSolvedMap,
		"matched_ques_dict": matchedMap,
	}).Error; err != nil {
		log.Printf("[task] GetUserData: failed to update %s: %v", username, err)
		return
	}
	log.Printf("[task] GetUserData: data for %s saved successfully", username)
}

// ── RefreshUserData ───────────────────────────────────────────────────────────

func RefreshUserData() {
	log.Println("[task] RefreshUserData started")

	var users []models.Leetcode
	// SELECT only the two columns we actually pass to GetUserData.
	if err := db.DB.Select("id", "username").Find(&users).Error; err != nil || len(users) == 0 {
		log.Println("[task] RefreshUserData: no users found")
		return
	}

	done := make(chan struct{}, len(users))
	for _, u := range users {
		go func(username string, id uint) {
			defer func() { done <- struct{}{} }()
			GetUserData(username, id)
		}(u.Username, u.ID)
	}
	for range users {
		<-done
	}

	CalculateLeaderboards()
	log.Println("[task] RefreshUserData: completed")
}

// ── CalculateLeaderboards ─────────────────────────────────────────────────────

func CalculateLeaderboards() {
	log.Println("[task] CalculateLeaderboards started")

	if err := generateLeaderboardEntries(); err != nil {
		log.Printf("[task] CalculateLeaderboards: generate entries failed: %v", err)
		return
	}

	// Two targeted SELECTs cover everything needed.
	var users []models.Leetcode
	if err := db.DB.Select("id", "username", "photo_url").Find(&users).Error; err != nil {
		log.Printf("[task] CalculateLeaderboards: failed to fetch users: %v", err)
		return
	}
	var allEntries []models.LeaderboardEntry
	if err := db.DB.Select("user_id", "interval", "questions_solved", "earliest_solved_timestamp").
		Find(&allEntries).Error; err != nil {
		log.Printf("[task] CalculateLeaderboards: failed to fetch entries: %v", err)
		return
	}

	type entryKey struct {
		UserID   uint
		Interval models.LeaderboardInterval
	}
	entryMap := make(map[entryKey]models.LeaderboardEntry, len(allEntries))
	for _, e := range allEntries {
		entryMap[entryKey{e.UserID, e.Interval}] = e
	}

	daily := make([]userRow, 0, len(users))
	weekly := make([]userRow, 0, len(users))
	monthly := make([]userRow, 0, len(users))
	photoMap := make(map[string]string, len(users))

	for _, u := range users {
		photoMap[u.Username] = u.PhotoURL
		de := entryMap[entryKey{u.ID, models.IntervalDay}]
		we := entryMap[entryKey{u.ID, models.IntervalWeek}]
		me := entryMap[entryKey{u.ID, models.IntervalMonth}]
		daily = append(daily, userRow{u.Username, de.QuestionsSolved, de.EarliestSolvedTimestamp})
		weekly = append(weekly, userRow{u.Username, we.QuestionsSolved, we.EarliestSolvedTimestamp})
		monthly = append(monthly, userRow{u.Username, me.QuestionsSolved, me.EarliestSolvedTimestamp})
	}

	var wg sync.WaitGroup
	wg.Add(3)

	var oneDay, oneWeek, oneMonth map[string]LeaderboardEntryData

	go func() {
		defer wg.Done()
		sortRows(daily)
		oneDay = buildLeaderboardMap(daily, photoMap)
	}()

	go func() {
		defer wg.Done()
		sortRows(weekly)
		oneWeek = buildLeaderboardMap(weekly, photoMap)
	}()

	go func() {
		defer wg.Done()
		sortRows(monthly)
		oneMonth = buildLeaderboardMap(monthly, photoMap)
	}()

	wg.Wait()

	// One UPDATE statement per rank column (3 total), each touching all rows.
	// Old: called bulkUpdateRanks three separate times which was already batched,
	// but we can now share the IN-list build across all three columns.
	bulkUpdateAllRanks(daily, weekly, monthly)

	wg.Add(3)

	go func() {
		defer wg.Done()
		upsertLeaderboard(models.LeaderboardDaily, oneDay)
	}()

	go func() {
		defer wg.Done()
		upsertLeaderboard(models.LeaderboardWeekly, oneWeek)
	}()

	go func() {
		defer wg.Done()
		upsertLeaderboard(models.LeaderboardMonthly, oneMonth)
	}()

	log.Println("[task] CalculateLeaderboards: completed successfully")
}

// ── generateLeaderboardEntries ────────────────────────────────────────────────
// Single INSERT ... ON CONFLICT for ALL users across ALL three intervals.
// Old: one upsert per user in a loop → N round-trips.
// New: one upsert for all users → 1 round-trip.

func generateLeaderboardEntries() error {
	var dbQuestions []models.Question
	if err := db.DB.Select("leetcode_id", "title_slug", "question_date").Find(&dbQuestions).Error; err != nil {
		return fmt.Errorf("failed to fetch questions: %w", err)
	}
	qRecords := make([]lc.QuestionRecord, len(dbQuestions))
	for i, q := range dbQuestions {
		qRecords[i] = lc.QuestionRecord{
			LeetcodeID:        q.LeetcodeID,
			TitleSlug:         q.TitleSlug,
			QuestionTimestamp: q.QuestionDate.Unix(),
		}
	}

	var users []models.Leetcode
	if err := db.DB.Select("id", "username", "total_solved_dict").Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	// Pre-allocate the full slice: 3 rows per user.
	allRows := make([]models.LeaderboardEntry, 0, len(users)*3)

	for _, u := range users {
		// Direct map copy — no marshal/unmarshal roundtrip.
		solvedDict := make(map[string]any, len(u.TotalSolvedDict))
		maps.Copy(solvedDict, u.TotalSolvedDict)

		dayMap, weekMap, monthMap := lc.CalSolvedIntervals(qRecords, solvedDict)

		allRows = append(allRows,
			models.LeaderboardEntry{
				UserID: u.ID, Interval: models.IntervalDay,
				QuestionsSolved: len(dayMap), EarliestSolvedTimestamp: maxVal(dayMap),
			},
			models.LeaderboardEntry{
				UserID: u.ID, Interval: models.IntervalWeek,
				QuestionsSolved: len(weekMap), EarliestSolvedTimestamp: maxVal(weekMap),
			},
			models.LeaderboardEntry{
				UserID: u.ID, Interval: models.IntervalMonth,
				QuestionsSolved: len(monthMap), EarliestSolvedTimestamp: maxVal(monthMap),
			},
		)
	}

	if len(allRows) == 0 {
		return nil
	}
	if err := db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "interval"}},
		DoUpdates: clause.AssignmentColumns([]string{"questions_solved", "earliest_solved_timestamp"}),
	}).Create(&allRows).Error; err != nil {
		return fmt.Errorf("batch upsert failed: %w", err)
	}

	log.Println("[task] generateLeaderboardEntries: completed")
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// sortRows uses stdlib sort.Slice (intro sort, O(n log n)) instead of the
// previous insertion sort (O(n²)).
func sortRows(rows []userRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Solved != rows[j].Solved {
			return rows[i].Solved > rows[j].Solved // higher solved first
		}
		return rows[i].Timestamp < rows[j].Timestamp // earlier timestamp wins ties
	})
}

func buildLeaderboardMap(rows []userRow, photoMap map[string]string) map[string]LeaderboardEntryData {
	result := make(map[string]LeaderboardEntryData, len(rows))
	for idx, row := range rows {
		lastSolv := ""
		if row.Timestamp > 0 {
			lastSolv = time.Unix(row.Timestamp, 0).Format("2006-01-02 15:04:05")
		}
		result[strconv.Itoa(idx+1)] = LeaderboardEntryData{
			Username: row.Username,
			PhotoURL: photoMap[row.Username],
			QuesSolv: row.Solved,
			LastSolv: lastSolv,
		}
	}
	return result
}

// bulkUpdateAllRanks issues one UPDATE per rank column (3 total).
// Each UPDATE uses a CASE WHEN expression to set every user's rank in one shot.
func bulkUpdateAllRanks(daily, weekly, monthly []userRow) {
	for _, job := range [...]struct {
		rows  []userRow
		field string
	}{
		{daily, "daily_rank"},
		{weekly, "weekly_rank"},
		{monthly, "monthly_rank"},
	} {
		if len(job.rows) == 0 {
			continue
		}
		var caseExpr strings.Builder

		// Pre-size: "CASE " + N×"WHEN username = ? THEN ? " + "END"
		caseExpr.Grow(5 + len(job.rows)*26 + 3)
		caseExpr.WriteString("CASE ")

		args := make([]any, 0, len(job.rows)*2+len(job.rows))
		usernames := make([]any, 0, len(job.rows))

		for idx, row := range job.rows {
			caseExpr.WriteString("WHEN username = ? THEN ?::bigint ")
			args = append(args, row.Username, idx+1)
			usernames = append(usernames, row.Username)
		}
		caseExpr.WriteString("END")

		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(usernames)), ",")
		args = append(args, usernames...)

		sql := fmt.Sprintf("UPDATE leetcodes SET %s = %s WHERE username IN (%s)",
			job.field, caseExpr.String(), placeholders)

		if err := db.DB.Exec(sql, args...).Error; err != nil {
			log.Printf("[task] bulkUpdateAllRanks: failed for %s: %v", job.field, err)
		}
	}
}

// upsertLeaderboard — INSERT ... ON CONFLICT DO UPDATE, one round-trip.
func upsertLeaderboard(lbType models.LeaderboardType, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf("[task] upsertLeaderboard: marshal failed for %s: %v", lbType, err)
		return
	}
	lb := models.Leaderboard{
		LeaderboardType: lbType,
		LeaderboardData: datatypes.JSON(raw),
	}
	if err := db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "leaderboard_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"leaderboard_data"}),
	}).Create(&lb).Error; err != nil {
		log.Printf("[task] upsertLeaderboard: failed for %s: %v", lbType, err)
	}
}

func maxVal(m map[int]int64) int64 {
	var max int64
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}
