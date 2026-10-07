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

// TestEventsTailCommitRetriesAfterStop reproduces the mismatch between the
// commit's HTTP request context (commitCtx, detached from the run ctx via
// context.WithoutCancel so it outlives a stop request) and the surrounding
// retry loop's context. The retry loop must watch the same uncancellable
// context the commit request uses; otherwise a stop request that lands on a
// transiently-failing commit bails out after the first attempt and the offset
// is never committed, so the next run re-delivers an already-shown event.
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
