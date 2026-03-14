package cache

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupTestCache replaces the global Client with one backed by
// miniredis so tests run without a real Redis instance.
func setupTestCache(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { mr.Close() })
	return mr
}

// Tests that Set stores a value in Redis and Get retrieves the
// same value for the given key.
func TestSetAndGet_RoundTrip(t *testing.T) {
	setupTestCache(t)

	if err := Set("hello", "world", time.Minute); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	got, err := Get("hello")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != "world" {
		t.Errorf("expected %q, got %q", "world", got)
	}
}

// Tests that Get returns a redis.Nil error when the key
// does not exist in the cache.
func TestGet_MissingKey_ReturnsError(t *testing.T) {
	setupTestCache(t)

	_, err := Get("nonexistent_key_xyz")
	if err == nil {
		t.Error("expected error for missing key, got nil")
	}
}

// Tests that Del removes a key from Redis so a subsequent Get
// returns an error.
func TestDel_RemovesKey(t *testing.T) {
	setupTestCache(t)

	Set("to-delete", "value", time.Minute)
	if err := Del("to-delete"); err != nil {
		t.Fatalf("Del error: %v", err)
	}
	_, err := Get("to-delete")
	if err == nil {
		t.Error("expected error after Del, got nil")
	}
}

// Tests that Set with a short TTL causes the key to expire and
// become unreachable after the TTL elapses.
func TestSet_TTLExpiry(t *testing.T) {
	mr := setupTestCache(t)

	Set("expiring", "soon", 1*time.Second)
	mr.FastForward(2 * time.Second) // advance miniredis clock

	_, err := Get("expiring")
	if err == nil {
		t.Error("expected key to be expired, but Get succeeded")
	}
}

// Tests that Set correctly overwrites an existing key with
// a new value.
func TestSet_Overwrite(t *testing.T) {
	setupTestCache(t)

	Set("key", "first", time.Minute)
	Set("key", "second", time.Minute)

	got, err := Get("key")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != "second" {
		t.Errorf("expected %q after overwrite, got %q", "second", got)
	}
}

// Tests that Del on a non-existent key does not return an error
// (idempotent deletion).
func TestDel_NonExistentKey_NoError(t *testing.T) {
	setupTestCache(t)

	if err := Del("does-not-exist"); err != nil {
		t.Errorf("expected no error deleting non-existent key, got: %v", err)
	}
}
