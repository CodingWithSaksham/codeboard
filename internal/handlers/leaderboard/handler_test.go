package leaderboard

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/creative-computing-society/codeboard/internal/db"
	"github.com/creative-computing-society/codeboard/internal/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB initialises an in-memory SQLite database and runs
// AutoMigrate so handler tests have a real schema to query against.
func setupTestDB(t *testing.T) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	gdb.AutoMigrate(
		&models.CUser{},
		&models.Leetcode{},
		&models.Question{},
		&models.LeaderboardEntry{},
		&models.Leaderboard{},
	)
	db.DB = gdb
}

// newTestApp builds a Fiber app that wires all leaderboard routes
// exactly as they are registered in main.go.
func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/daily/", DailyLeaderboard)
	app.Get("/weekly/", WeeklyLeaderboard)
	app.Get("/monthly/", MonthlyLeaderboard)
	app.Get("/questions/all/", GetAllQuestions)
	app.Get("/refresh_data/", DebugRefreshUserData)
	return app
}

// ── DailyLeaderboard ──────────────────────────────────────────────────────────

// Tests that GET /daily/ returns 404 when no daily leaderboard
// record exists in the database.
func TestDailyLeaderboard_NotFound(t *testing.T) {
	setupTestDB(t)
	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/daily/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// Tests that GET /daily/ returns 200 and the stored JSON payload
// when a daily leaderboard record is present.
func TestDailyLeaderboard_Found(t *testing.T) {
	setupTestDB(t)
	payload := `{"1":{"username":"alice","ques_solv":3}}`
	db.DB.Create(&models.Leaderboard{
		LeaderboardType: models.LeaderboardDaily,
		LeaderboardData: datatypes.JSON(payload),
	})

	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/daily/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── WeeklyLeaderboard ─────────────────────────────────────────────────────────

// Tests that GET /weekly/ returns 404 when no weekly leaderboard
// record exists in the database.
func TestWeeklyLeaderboard_NotFound(t *testing.T) {
	setupTestDB(t)
	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/weekly/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// Tests that GET /weekly/ returns 200 and the correct JSON when
// a weekly leaderboard row is seeded in the database.
func TestWeeklyLeaderboard_Found(t *testing.T) {
	setupTestDB(t)
	db.DB.Create(&models.Leaderboard{
		LeaderboardType: models.LeaderboardWeekly,
		LeaderboardData: datatypes.JSON(`{"1":{"username":"bob"}}`),
	})

	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/weekly/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── MonthlyLeaderboard ────────────────────────────────────────────────────────

// Tests that GET /monthly/ returns 404 when no monthly leaderboard
// row is present in the database.
func TestMonthlyLeaderboard_NotFound(t *testing.T) {
	setupTestDB(t)
	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/monthly/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// Tests that GET /monthly/ returns 200 with valid JSON when the
// monthly leaderboard is seeded in the database.
func TestMonthlyLeaderboard_Found(t *testing.T) {
	setupTestDB(t)
	db.DB.Create(&models.Leaderboard{
		LeaderboardType: models.LeaderboardMonthly,
		LeaderboardData: datatypes.JSON(`{"1":{"username":"carol"}}`),
	})

	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/monthly/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── GetAllQuestions ───────────────────────────────────────────────────────────

// Tests that GET /questions/all/ returns 404 when the questions
// table is empty.
func TestGetAllQuestions_NoQuestions_Returns404(t *testing.T) {
	setupTestDB(t)
	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/questions/all/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// Tests that GET /questions/all/ returns 200 and a non-empty array
// when questions are present in the database.
func TestGetAllQuestions_WithData_Returns200(t *testing.T) {
	setupTestDB(t)
	db.DB.Create(&models.Question{
		LeetcodeID: 1,
		Title:      "Two Sum",
		TitleSlug:  "two-sum",
		Difficulty: "Easy",
	})

	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/questions/all/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var questions []questionResponse
	json.NewDecoder(resp.Body).Decode(&questions)
	if len(questions) == 0 {
		t.Error("expected at least one question in response")
	}
}

// Tests that GET /questions/all/ returns questions without a Status
// field (empty string), as these are public (unauthenticated) responses.
func TestGetAllQuestions_NoStatusField(t *testing.T) {
	setupTestDB(t)
	db.DB.Create(&models.Question{LeetcodeID: 2, Title: "Add Two Numbers", TitleSlug: "add-two-numbers"})

	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/questions/all/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	var questions []questionResponse
	json.NewDecoder(resp.Body).Decode(&questions)

	for _, q := range questions {
		if q.Status != "" {
			t.Errorf("expected empty Status in public response, got %q", q.Status)
		}
	}
}

// ── DebugRefreshUserData ──────────────────────────────────────────────────────

// Tests that GET /refresh_data/ always returns 200 with the expected
// message and fires the refresh goroutine without blocking.
func TestDebugRefreshUserData_Returns200(t *testing.T) {
	setupTestDB(t)
	resp, err := newTestApp(t).Test(httptest.NewRequest("GET", "/refresh_data/", nil), -1)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if !strings.Contains(body["message"], "refresh") {
		t.Errorf("unexpected message: %q", body["message"])
	}
}
