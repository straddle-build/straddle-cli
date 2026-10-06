// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AdaptiveLimiter shares one request pace across callers. It backs off on 429
// and recovers after consecutive successes, never exceeding the configured rate.
// Per-session only, not persisted. Methods are safe to call on a nil receiver.
type AdaptiveLimiter struct {
	admission   chan struct{} // serialize admissions without blocking rate feedback
	mu          sync.Mutex
	rate        float64
	maximum     float64
	ceiling     float64
	successes   int
	rampAfter   int
	lastRequest time.Time // zero-value: first Wait() returns immediately
}

// NewAdaptiveLimiter returns a limiter starting at ratePerSec. Nonpositive
// values return nil for compatibility and disable limiting. Methods on the
// nil limiter no-op.
func NewAdaptiveLimiter(ratePerSec float64) *AdaptiveLimiter {
	if ratePerSec <= 0 {
		return nil
	}
	return &AdaptiveLimiter{
		admission: make(chan struct{}, 1),
		rate:      ratePerSec,
		maximum:   ratePerSec,
		rampAfter: 10,
	}
}

// ValidateRateLimit checks the CLI-facing rate limit before a client is built.
// Zero disables limiting; positive intervals must fit in time.Duration and
// retain at least one nanosecond of resolution.
func ValidateRateLimit(ratePerSec float64) error {
	if ratePerSec < 0 {
		return fmt.Errorf("rate limit must be non-negative")
	}
	if ratePerSec == 0 {
		return nil
	}
	if math.IsNaN(ratePerSec) || math.IsInf(ratePerSec, 0) {
		return fmt.Errorf("rate limit must be finite")
	}
	interval := float64(time.Second) / ratePerSec
	maxDuration := float64(time.Duration(1<<63 - 1))
	if interval < 1 || interval >= maxDuration {
		return fmt.Errorf("rate limit is outside the representable pacing range")
	}
	return nil
}

func (l *AdaptiveLimiter) Wait() {
	if l == nil {
		return
	}
	l.admission <- struct{}{}
	defer func() { <-l.admission }()
	for {
		l.mu.Lock()
		delay := time.Duration(math.Ceil(float64(time.Second) / l.rate))
		remaining := delay - time.Since(l.lastRequest)
		if l.lastRequest.IsZero() || remaining <= 0 {
			l.lastRequest = time.Now()
			l.mu.Unlock()
			return
		}
		l.mu.Unlock()
		time.Sleep(remaining)
		// Recheck the current rate: a 429 may have extended the wait.
	}
}

func (l *AdaptiveLimiter) OnSuccess() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.successes++
	if l.successes >= l.rampAfter {
		newRate := l.rate * 1.25
		capRate := l.maximum * 0.9
		if l.ceiling > l.rate {
			capRate = l.ceiling * 0.9
		}
		if newRate > capRate {
			newRate = capRate
		}
		l.rate = min(l.maximum, max(l.rate, newRate))
		l.successes = 0
	}
}

func (l *AdaptiveLimiter) OnRateLimit() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ceiling = l.rate
	l.rate = min(l.rate, max(0.5, l.rate/2))
	l.successes = 0
}

func (l *AdaptiveLimiter) Rate() float64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rate
}

// MaxRetryWait caps the wait derived from a Retry-After header so a buggy
// or hostile upstream cannot pin a CLI for hours.
const MaxRetryWait = 60 * time.Second

const (
	defaultRetryWait               = 5 * time.Second
	unixEpochSecondsThreshold      = 1_000_000_000
	unixEpochMillisecondsThreshold = 1_000_000_000_000
)

// RetryAfter parses an HTTP Retry-After header (RFC 7231: delta-seconds or
// HTTP-date), plus common Unix epoch seconds/milliseconds variants emitted by
// some APIs. Waits are capped at MaxRetryWait. Returns 5s when missing or
// unparseable.
func RetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return defaultRetryWait
	}
	header := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if header == "" {
		return defaultRetryWait
	}
	if value, err := strconv.ParseInt(header, 10, 64); err == nil {
		return retryAfterFromNumber(value)
	}
	if t, err := http.ParseTime(header); err == nil {
		wait := time.Until(t)
		if wait > MaxRetryWait {
			return MaxRetryWait
		}
		if wait > 0 {
			return wait
		}
	}
	return defaultRetryWait
}

func retryAfterFromNumber(value int64) time.Duration {
	if value <= 0 {
		return defaultRetryWait
	}
	if value > int64(MaxRetryWait/time.Second) {
		if wait := retryAfterEpochWait(value); wait > 0 {
			if wait > MaxRetryWait {
				return MaxRetryWait
			}
			return wait
		}
		return MaxRetryWait
	}
	return time.Duration(value) * time.Second
}

func retryAfterEpochWait(value int64) time.Duration {
	switch {
	case value >= unixEpochMillisecondsThreshold:
		return time.Until(time.UnixMilli(value))
	case value >= unixEpochSecondsThreshold:
		return time.Until(time.Unix(value, 0))
	default:
		return 0
	}
}
