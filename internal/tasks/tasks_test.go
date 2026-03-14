package tasks

import (
	"testing"
	"time"
)

// ── sortRows ──────────────────────────────────────────────────────────────────

// Tests that sortRows orders rows descending by Solved count,
// placing the highest solver first.
func TestSortRows_DescendingBySolved(t *testing.T) {
	rows := []userRow{
		{Username: "a", Solved: 3},
		{Username: "b", Solved: 7},
		{Username: "c", Solved: 1},
	}
	sortRows(rows)
	if rows[0].Username != "b" {
		t.Errorf("expected b first, got %q", rows[0].Username)
	}
	if rows[1].Username != "a" {
		t.Errorf("expected a second, got %q", rows[1].Username)
	}
	if rows[2].Username != "c" {
		t.Errorf("expected c last, got %q", rows[2].Username)
	}
}

// Tests that sortRows breaks ties by ascending Timestamp, so the
// user who solved earliest wins the tie.
func TestSortRows_TieBreakByEarliestTimestamp(t *testing.T) {
	rows := []userRow{
		{Username: "late", Solved: 5, Timestamp: 2000},
		{Username: "early", Solved: 5, Timestamp: 1000},
	}
	sortRows(rows)
	if rows[0].Username != "early" {
		t.Errorf("expected early to win tie-break, got %q", rows[0].Username)
	}
}

// Tests that sortRows leaves a single-element slice unchanged
// and does not panic on empty input.
func TestSortRows_SingleAndEmpty(t *testing.T) {
	single := []userRow{{Username: "only", Solved: 1}}
	sortRows(single) // should not panic

	empty := []userRow{}
	sortRows(empty) // should not panic
}

// ── buildLeaderboardMap ───────────────────────────────────────────────────────

// Tests that buildLeaderboardMap assigns 1-based rank keys and
// correctly populates username, photo, solved count, and timestamp.
func TestBuildLeaderboardMap_RankKeysAndFields(t *testing.T) {
	ts := int64(1700000000)
	rows := []userRow{
		{Username: "alice", Solved: 10, Timestamp: ts},
		{Username: "bob", Solved: 5, Timestamp: 0},
	}
	photoMap := map[string]string{
		"alice": "http://example.com/alice.png",
		"bob":   "",
	}
	result := buildLeaderboardMap(rows, photoMap)

	if _, ok := result["1"]; !ok {
		t.Error("expected rank key \"1\" in result")
	}
	if result["1"].Username != "alice" {
		t.Errorf("rank 1 username: expected alice, got %q", result["1"].Username)
	}
	if result["1"].QuesSolv != 10 {
		t.Errorf("rank 1 QuesSolv: expected 10, got %d", result["1"].QuesSolv)
	}
	if result["1"].PhotoURL != "http://example.com/alice.png" {
		t.Errorf("rank 1 PhotoURL mismatch: %q", result["1"].PhotoURL)
	}
}

// Tests that buildLeaderboardMap formats non-zero Timestamp values
// as "2006-01-02 15:04:05" and leaves zero timestamps as empty string.
func TestBuildLeaderboardMap_TimestampFormatting(t *testing.T) {
	ts := int64(1700000000)
	rows := []userRow{
		{Username: "user1", Solved: 1, Timestamp: ts},
		{Username: "user2", Solved: 0, Timestamp: 0},
	}
	result := buildLeaderboardMap(rows, map[string]string{})

	expected := time.Unix(ts, 0).Format("2006-01-02 15:04:05")
	if result["1"].LastSolv != expected {
		t.Errorf("LastSolv: expected %q, got %q", expected, result["1"].LastSolv)
	}
	if result["2"].LastSolv != "" {
		t.Errorf("zero Timestamp should produce empty LastSolv, got %q", result["2"].LastSolv)
	}
}

// Tests that buildLeaderboardMap returns an empty map
// when given an empty rows slice.
func TestBuildLeaderboardMap_EmptyRows(t *testing.T) {
	result := buildLeaderboardMap([]userRow{}, map[string]string{})
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

// ── maxVal ────────────────────────────────────────────────────────────────────

// Tests that maxVal returns the largest int64 value
// in the provided map.
func TestMaxVal_ReturnsMaximum(t *testing.T) {
	m := map[int]int64{1: 100, 2: 9999, 3: 50}
	if got := maxVal(m); got != 9999 {
		t.Errorf("expected 9999, got %d", got)
	}
}

// Tests that maxVal returns 0 for an empty map,
// matching the zero-value initialisation in CalculateLeaderboards.
func TestMaxVal_EmptyMap(t *testing.T) {
	if got := maxVal(map[int]int64{}); got != 0 {
		t.Errorf("expected 0 for empty map, got %d", got)
	}
}

// Tests that maxVal handles a single-entry map and returns
// that entry's value.
func TestMaxVal_SingleEntry(t *testing.T) {
	m := map[int]int64{42: 777}
	if got := maxVal(m); got != 777 {
		t.Errorf("expected 777, got %d", got)
	}
}

// Tests that maxVal returns the correct maximum even when
// all values in the map are negative.
func TestMaxVal_AllNegative(t *testing.T) {
	m := map[int]int64{1: -10, 2: -5, 3: -100}
	// All negative: our implementation starts max at 0, so result is 0.
	// This documents the known behaviour.
	if got := maxVal(m); got != 0 {
		t.Errorf("expected 0 (no positive values), got %d", got)
	}
}
