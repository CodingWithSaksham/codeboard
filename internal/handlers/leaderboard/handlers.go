package leaderboard

import (
	"errors"
	"time"

	"github.com/creative-computing-society/codeboard/internal/db"
	"github.com/creative-computing-society/codeboard/internal/models"
	"github.com/creative-computing-society/codeboard/internal/tasks"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register links a LeetCode username to the authenticated user.
//
// It validates the request body, creates a new Leetcode record,
// and asynchronously triggers background data fetching.
//
// Returns:
//
//	201: User registered successfully
//	400: Invalid request body, missing username, user registration failed
//
// Route:
//
//	POST /register/
func Register(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var body struct {
		Username string `json:"username"`
	}
	if err := c.BodyParser(&body); err != nil || body.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username is required"})
	}

	acc := models.Leetcode{
		Username: body.Username,
		UserID:   userID,
	}

	err := gorm.G[models.Leetcode](db.DB).
		Create(c.Context(), &acc)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "User registration failed"})
	}

	go tasks.GetUserData(body.Username, acc.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User registered successfully"})
}

// Profile retrieves the authenticated user's linked LeetCode profile.
//
// It loads the associated account from the database and returns
// a serialized representation.
//
// WARN: A 1-1 relation exists between CUser and Leetcode models,
// so GORM throws an error if one user tries to register multiple accounts.
//
// TODO: Check if account exists on LeetCode using their GraphQL API.
//
// Returns:
//
//	200: Profile retrieved successfully
//	404: No linked LeetCode account found
//
// Route:
//
//	GET /user/profile/
func Profile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var acc models.Leetcode
	if err := db.DB.Where("user_id = ?", userID).First(&acc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User does not exist"})
	}
	return c.Status(fiber.StatusOK).JSON(serializeLeetcode(acc))
}

// GetQuestionsForTheDay returns all questions scheduled for the current day,
// including the authenticated user's solve status.
//
// Returns:
//
//	200: Questions retrieved successfully
//	400: No linked LeetCode account found
//	404: No questions found for date range
//	500: Failed to fetch questions
//
// Route:
//
//	GET /questions/today/
func GetQuestionsForTheDay(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	acc, err := gorm.G[models.Leetcode](db.DB).
		Where("user_id = ?", userID).
		First(c.Context())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No linked LeetCode account"})
	}

	solved := buildSolvedSet(&acc)

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24*time.Hour - time.Nanosecond)

	questions, err := gorm.G[models.Question](db.DB).
		Where("question_date BETWEEN ? AND ?", start, end).
		Find(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to fetch questions"})
	}

	if len(questions) == 0 {
		return c.Status(fiber.StatusNotFound).
			JSON(fiber.Map{"error": "No questions found for date range"})
	}

	result := make([]questionResponse, len(questions))
	for i, q := range questions {
		result[i] = serializeQuestion(q, solved)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// GetAllQuestions returns all available questions without user-specific
// solve information.
//
// Returns:
//
//	200: Questions retrieved successfully
//	404: Questions not found
//	500: Database error
//
// Route:
//
//	GET /questions/all/
func GetAllQuestions(c *fiber.Ctx) error {
	questions, err := gorm.G[models.Question](db.DB).
		Find(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "database error"})
	}

	if len(questions) == 0 {
		return c.Status(fiber.StatusNotFound).
			JSON(fiber.Map{"error": "Questions not found"})
	}

	result := make([]questionResponse, len(questions))
	for i, q := range questions {
		result[i] = serializeQuestionPublic(q)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// leaderboardHandler retrieves the specified leaderboard and returns its JSON data.
//
// Params:
//
//	lbType models.LeaderboardType: The type of leaderboard to retrieve
//	(e.g., Daily, Weekly, Monthly)
//
// Returns:
//
//	200: Leaderboard retrieved successfully
//	404: Leaderboard not found
func leaderboardHandler(c *fiber.Ctx, lbType models.LeaderboardType) error {
	lb, err := gorm.G[models.Leaderboard](db.DB).
		Select("leaderboard_data").
		Where("leaderboard_type = ?", lbType).
		First(c.Context())

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": string(lbType) + " leaderboard not found"},
		)
	}

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{"error": string(lbType) + " leaderboard not found"},
		)
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Status(fiber.StatusOK).Send(lb.LeaderboardData)
}

// DailyLeaderboard retrieves the Daily leaderboard and returns its JSON data.
//
// Returns:
//
//	200: Leaderboard retrieved successfully
//	404: Leaderboard not found
//
// Route:
//
//	GET /daily/
func DailyLeaderboard(c *fiber.Ctx) error {
	return leaderboardHandler(c, models.LeaderboardDaily)
}

// WeeklyLeaderboard retrieves the Weekly leaderboard and returns its JSON data.
//
// Returns:
//
//	200: Leaderboard retrieved successfully
//	404: Leaderboard not found
//
// Route:
//
//	GET /weekly/
func WeeklyLeaderboard(c *fiber.Ctx) error {
	return leaderboardHandler(c, models.LeaderboardWeekly)
}

// MonthlyLeaderboard retrieves the Monthly leaderboard and returns its JSON data.
//
// Returns:
//
//	200: Leaderboard retrieved successfully
//	404: Leaderboard not found
//
// Route:
//
//	GET /monthly/
func MonthlyLeaderboard(c *fiber.Ctx) error {
	return leaderboardHandler(c, models.LeaderboardMonthly)
}

// DebugRefreshUserData triggers a background refresh of user data.
//
// Behavior:
//
//	Initiates an asynchronous data refresh task.
//
// Returns:
//
//	200: Data refresh successfully initiated
//
// Route:
//
//	GET /refresh_data/
func DebugRefreshUserData(c *fiber.Ctx) error {
	go tasks.RefreshUserData()
	return c.Status(fiber.StatusOK).JSON(
		fiber.Map{"message": "Data refresh initiated"},
	)
}
