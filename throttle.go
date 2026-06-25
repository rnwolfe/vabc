package vabc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// throttle is a persistent, cross-process politeness gate for the undocumented
// Virginia ABC endpoints. An agent invokes a fresh process per call, so an
// in-memory timer is a no-op — state (last request time, circuit-breaker
// blocked-until) is persisted to a small JSON file in the state dir.
//
// Behavior:
//   - Enforces a minimum interval between requests (short sleep; serializes bursts).
//   - On a detected block (HTTP 429 / WAF challenge), opens a circuit breaker until
//     blockedUntil (honoring Retry-After). While open, requests FAIL FAST by default
//     so an agent loop never deadlocks; pass wait=true (--wait) to wait it out, up to
//     maxWait. The error carries RetryAfter so the caller can schedule a retry.
//
// This is courtesy/respect for the provider, not evasion. There is no UA spoofing,
// proxy rotation, or challenge solving anywhere in this client.
type throttle struct {
	statePath   string
	minInterval time.Duration
	wait        bool
	maxWait     time.Duration
}

type throttleState struct {
	LastRequestMillis  int64 `json:"lastRequestMillis"`
	BlockedUntilMillis int64 `json:"blockedUntilMillis"`
}

func newThrottle(statePath string, minInterval time.Duration, wait bool, maxWait time.Duration) *throttle {
	return &throttle{statePath: statePath, minInterval: minInterval, wait: wait, maxWait: maxWait}
}

// defaultStatePath resolves the throttle state file location.
// Override the directory with VABC_STATE_DIR.
func defaultStatePath() string {
	if d := os.Getenv("VABC_STATE_DIR"); d != "" {
		return filepath.Join(d, "throttle.json")
	}
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "vabc", "throttle.json")
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "vabc", "throttle.json")
	}
	return ""
}

func (t *throttle) load() throttleState {
	var s throttleState
	if t.statePath == "" {
		return s
	}
	b, err := os.ReadFile(t.statePath)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func (t *throttle) save(s throttleState) {
	if t.statePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(t.statePath), 0o700); err != nil {
		return
	}
	b, err := json.Marshal(s)
	if err != nil {
		return
	}
	// Best-effort write. The race window between processes is tiny and the only
	// consequence is a marginally-too-fast request — acceptable for a courtesy gate.
	_ = os.WriteFile(t.statePath, b, 0o600)
}

// acquire blocks until it is polite to make a request, or returns a rate-limited
// APIError when the circuit breaker is open and waiting is disabled.
func (t *throttle) acquire(ctx context.Context) error {
	s := t.load()
	now := time.Now()

	// Circuit breaker open?
	if s.BlockedUntilMillis > 0 {
		blockedUntil := time.UnixMilli(s.BlockedUntilMillis)
		if remaining := time.Until(blockedUntil); remaining > 0 {
			if !t.wait || remaining > t.maxWait {
				return rateLimited(0, "throttle circuit breaker open (recent block)", remaining)
			}
			if err := sleepCtx(ctx, remaining); err != nil {
				return err
			}
			now = time.Now()
		}
	}

	// Minimum spacing between requests.
	if t.minInterval > 0 && s.LastRequestMillis > 0 {
		since := now.Sub(time.UnixMilli(s.LastRequestMillis))
		if wait := t.minInterval - since; wait > 0 {
			if wait > t.maxWait && !t.wait {
				// Shouldn't happen for a sub-second minInterval, but stay safe.
				wait = t.maxWait
			}
			if err := sleepCtx(ctx, wait); err != nil {
				return err
			}
			now = time.Now()
		}
	}

	s.LastRequestMillis = now.UnixMilli()
	t.save(s)
	return nil
}

// observe records a block so subsequent processes back off (circuit breaker).
func (t *throttle) observe(retryAfter time.Duration) {
	if retryAfter <= 0 {
		retryAfter = 60 * time.Second
	}
	s := t.load()
	s.BlockedUntilMillis = time.Now().Add(retryAfter).UnixMilli()
	t.save(s)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
