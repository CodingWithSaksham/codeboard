package scheduler

import (
	"testing"
	"time"
)

// Tests that Start initialises the cron scheduler without panicking
// and that Stop shuts it down cleanly.
func TestStartAndStop_NoPanic(t *testing.T) {
	// Start and Stop should not panic or return errors.
	Start()
	time.Sleep(50 * time.Millisecond) // let the cron goroutine boot
	Stop()
}

// Tests that calling Stop before Start (c == nil) does not panic,
// covering the nil guard in Stop.
func TestStop_BeforeStart_NoPanic(t *testing.T) {
	c = nil // reset global
	Stop()  // must not panic
}

// Tests that calling Start twice and then Stop does not panic,
// verifying the scheduler can be replaced safely.
func TestStart_CalledTwice_NoPanic(t *testing.T) {
	Start()
	Start() // second call replaces the old cron
	Stop()
}
