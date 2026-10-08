// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestEventsTailCommitRetriesAfterStop: the commit retry loop must outlive a
// stop request just as the commit request itself does; otherwise a stop
// that lands on a transiently-failing commit bails out after the first
// attempt and the offset is never committed, so the next run re-delivers an
// already-shown event.
//
// The fake endpoint serves one event on the first poll, then on the first
// commit attempt it cancels the run ctx (simulating Ctrl+C/SIGTERM hitting the
// commit) and answers HTTP 500 (a transient failure per isTransientPollingError);
// any later commit attempt succeeds with 204.
func TestEventsTailCommitRetriesAfterStop(t *testing.T) {
	events := []string{tailEvent(0, tailTestAccountA)}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var (
		mu             sync.Mutex
		commitAttempts int
		commits        []int64
		polls          int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tailTestToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/commit") {
			mu.Lock()
			commitAttempts++
			attempt := commitAttempts
			mu.Unlock()
			var body struct {
				Offset int64 `json:"offset"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if attempt == 1 {
				cancel()
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, `{"code":"transient","detail":"try again"}`)
				return
			}
			mu.Lock()
			commits = append(commits, body.Offset)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mu.Lock()
		polls++
		served := polls
		mu.Unlock()
		if served == 1 {
			_, _ = fmt.Fprintf(w, `{"data":[{"offset":0,"eventType":"charge.event.v1","payload":%s,"timestamp":"2026-09-30T15:10:26Z"}],"done":true}`, events[0])
			return
		}
		_, _ = io.WriteString(w, `{"data":[],"done":true}`)
	}))
	defer server.Close()
	isolateEventsTail(t, server)
	stdout, stderr, err := runEventsTail(t, ctx, "--consumer", "retry-after-stop")
	t.Logf("err=%v commits=%v commitAttempts=%d polls=%d\nstdout=%s\nstderr=%s",
		err, commits, commitAttempts, polls, stdout, stderr)
	if err != nil {
		t.Fatalf("tail returned %v; expected the stop-requested run to succeed after the commit was retried\nstderr: %s", err, stderr)
	}
	if len(commits) == 0 {
		t.Fatalf("commit was never recorded; the offset was not advanced, so a restart would re-show offset 0 (duplicate). commitAttempts=%d stderr=%s", commitAttempts, stderr)
	}
	if commits[0] != 0 {
		t.Fatalf("committed offset %d, want 0", commits[0])
	}
	if commitAttempts < 2 {
		t.Fatalf("commit was attempted only %d time(s); the retry-after-stop invariant requires at least one retry after the transient 500", commitAttempts)
	}
}

// TestEventsTailCommitGivesUpAfterStopBound: a stop that lands on a commit
// endpoint that keeps failing transiently must not hold the first Ctrl+C
// open forever. The commit keeps retrying for the post-stop bound, then the
// command fails with the "shown again" hint and a non-zero exit instead of
// needing a second Ctrl+C that kills the process without it.
func TestEventsTailCommitGivesUpAfterStopBound(t *testing.T) {
	const bound = 50 * time.Millisecond
	saved := eventsStopCommitTimeout
	eventsStopCommitTimeout = bound
	t.Cleanup(func() { eventsStopCommitTimeout = saved })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		mu        sync.Mutex
		polls     int
		stoppedAt time.Time
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/commit") {
			if stoppedAt.IsZero() {
				stoppedAt = time.Now()
				cancel()
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"code":"unavailable"}`)
			return
		}
		polls++
		if polls == 1 {
			_, _ = fmt.Fprintf(w, `{"data":[{"offset":0,"eventType":"charge.event.v1","payload":%s,"timestamp":"2026-09-30T15:10:26Z"}],"done":true}`, tailEvent(0, tailTestAccountA))
			return
		}
		_, _ = io.WriteString(w, `{"data":[],"done":true}`)
	}))
	defer server.Close()
	isolateEventsTail(t, server)

	type result struct {
		stdout, stderr string
		err            error
	}
	finished := make(chan result, 1)
	go func() {
		stdout, stderr, err := runEventsTail(t, ctx, "--consumer", "commit-outage-after-stop", "--json")
		finished <- result{stdout, stderr, err}
	}()
	var got result
	select {
	case got = <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("tail did not return after a stop while the commit kept failing; the first Ctrl+C never exits")
	}
	mu.Lock()
	sinceStop := time.Since(stoppedAt)
	mu.Unlock()

	if got.err == nil || !strings.Contains(got.err.Error(), "commit of offset 0") || !strings.Contains(got.err.Error(), "shown again on the next run") {
		t.Fatalf("tail error = %v, want the failed commit of offset 0 with the shown-again hint\nstderr: %s", got.err, got.stderr)
	}
	if code := ExitCode(got.err); code != 5 {
		t.Errorf("exit code = %d, want 5 (%v)", code, got.err)
	}
	if sinceStop < bound {
		t.Errorf("tail gave up %s after the stop, before the %s post-stop commit bound", sinceStop, bound)
	}
	if !strings.Contains(got.stdout, `"offset":0`) {
		t.Errorf("stdout = %q, want offset 0 shown before the commit", got.stdout)
	}
}
