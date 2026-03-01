package scheduler

import (
	"log"

	"github.com/creative-computing-society/codeboard/internal/tasks"
	"github.com/robfig/cron/v3"
)

var c *cron.Cron

// Start initialises and starts the cron scheduler.
// Matches the original Celery Beat setup: refresh_user_data runs periodically.
func Start() {
	c = cron.New()

	// Refresh all user data and recalculate leaderboards every hour.
	_, err := c.AddFunc("0 * * * *", func() {
		log.Println("[scheduler] Running scheduled RefreshUserData")
		tasks.RefreshUserData()
	})
	if err != nil {
		log.Fatalf("[scheduler] Failed to add cron job: %v", err)
	}

	c.Start()
	log.Println("[scheduler] Cron scheduler started (refresh every hour)")
}

func Stop() {
	if c != nil {
		c.Stop()
	}
}
