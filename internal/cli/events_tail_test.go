// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	tailTestToken    = "sk_poll_fixture-secret.us"
	tailTestAccountA = "c04f633b-8b7d-4072-ac37-979d5af4e64a"
	tailTestAccountB = "857c679a-8e80-47f1-9334-39dd08ea801f"
)

// tailEvent returns a Straddle event payload. The description carries
// characters json.Marshal would HTML-escape, to prove payloads pass through
// byte for byte.
func tailEvent(n int, account string) string {
	return fmt.Sprintf(`{"account_id":"%s","data":{"id":"ch-%d","status":"paid","description":"a<b&c"},"event_id":"evt-%d","event_type":"charge.event.v1"}`, account, n, n)
}

func tailEvents(from, to int, account string) []string {
	var out []string
	for n := from; n < to; n++ {
		out = append(out, tailEvent(n, account))
	}
	return out
}

// fakePoller mimics the polling endpoint's consumer contract: a poll leases
// the next batch, a leased consumer gets 423 until it commits the batch's
// last offset (or the lease expires), and a commit moves the position.
type fakePoller struct {
	mu       sync.Mutex
	events   []string         // payload per offset
	arriving []string         // appended once the first poll has been answered
	position map[string]int64 // consumer -> next offset to serve
	leaseEnd map[string]int64 // consumer -> last offset of its uncommitted batch
	failures []int            // statuses answered to the next polls, in order
	// expireOnLocked makes a 423 also expire the lease, standing in for
	// the five minutes a real lease lasts.
	expireOnLocked bool
	starts         []string
	commits        []int64
	polls          int
	stop           context.CancelFunc
	stopped        bool
}

func newFakePoller(events []string) *fakePoller {
	return &fakePoller{events: events, position: map[string]int64{}, leaseEnd: map[string]int64{}}
}

// caughtUp is the default stop condition: every event, including the ones
// still to arrive, has been committed by the consumer.
func (f *fakePoller) caughtUp(consumer string) bool {
	pos, known := f.position[consumer]
	return known && len(f.arriving) == 0 && pos == int64(len(f.events))
}

func (f *fakePoller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+tailTestToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/app/app_x/polling-endpoint/poll_x/consumer/")
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if consumer, isCommit := strings.CutSuffix(rest, "/commit"); isCommit {
		var body struct {
			Offset int64 `json:"offset"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&body) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.commits = append(f.commits, body.Offset)
		f.position[consumer] = body.Offset + 1
		if end, leased := f.leaseEnd[consumer]; leased && end == body.Offset {
			delete(f.leaseEnd, consumer)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	consumer := rest
	f.polls++
	if f.caughtUp(consumer) {
		f.stopped = true
		f.stop()
		_, _ = io.WriteString(w, `{"data":[],"done":true}`)
		return
	}
	if len(f.failures) > 0 {
		status := f.failures[0]
		f.failures = f.failures[1:]
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"code":"scripted","detail":"scripted failure"}`)
		return
	}
	if _, leased := f.leaseEnd[consumer]; leased {
		if f.expireOnLocked {
			delete(f.leaseEnd, consumer)
		}
		w.WriteHeader(http.StatusLocked)
		_, _ = io.WriteString(w, `{"code":"no_available_lease","detail":"No available lease"}`)
		return
	}
	start := r.URL.Query().Get("starting_position")
	f.starts = append(f.starts, start)
	pos, known := f.position[consumer]
	if !known {
		if start == "latest" {
			pos = int64(len(f.events))
		}
		f.position[consumer] = pos
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	end := min(pos+int64(limit), int64(len(f.events)))
	items := make([]string, 0, end-pos)
	for off := pos; off < end; off++ {
		items = append(items, fmt.Sprintf(`{"offset":%d,"eventId":null,"eventType":"charge.event.v1","payload":%s,"channels":null,"id":"msg_%d","timestamp":"2026-09-30T15:10:%02d.760923964Z"}`, off, f.events[off], off, off%60))
	}
	if end > pos {
		f.leaseEnd[consumer] = end - 1
	}
	done := end == int64(len(f.events))
	f.events = append(f.events, f.arriving...)
	f.arriving = nil
	_, _ = fmt.Fprintf(w, `{"data":[%s],"done":%t}`, strings.Join(items, ","), done)
}

// fakeHandler is a developer's local webhook handler. It fails the first
// failFirst[payload] deliveries of a payload, and can stop the tail after
// a number of failed attempts.
type fakeHandler struct {
	mu        sync.Mutex
	failFirst map[string]int
	delivered []string // payloads it accepted, in order
	failed    int
	stopAfter int // stop the tail after this many failures (0: never)
	stop      context.CancelFunc
}

func (h *fakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}
	if h.failFirst[string(body)] > 0 {
		h.failFirst[string(body)]--
		h.failed++
		if h.stopAfter > 0 && h.failed >= h.stopAfter {
			h.stop()
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.delivered = append(h.delivered, string(body))
	w.WriteHeader(http.StatusOK)
}

// isolateEventsTail points config, platform context and polling settings
// at test-owned values and returns the polling URL for server.
func isolateEventsTail(t *testing.T, server *httptest.Server) {
	t.Helper()
	isolateAPIConfig(t)
	t.Setenv("STRADDLE_API_KEY", "straddle-api-key-must-not-be-sent")
	t.Setenv("STRADDLE_POLLING_URL", server.URL+"/api/v1/app/app_x/polling-endpoint/poll_x/consumer/{consumer_id}")
	t.Setenv("STRADDLE_POLLING_TOKEN", tailTestToken)
}

func runEventsTail(t *testing.T, ctx context.Context, args ...string) (string, string, error) {
	t.Helper()
	cmd := RootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"events", "tail", "--interval", "1ms"}, args...))
	err := cmd.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

type shownEvent struct {
	Offset        int64           `json:"offset"`
	Timestamp     string          `json:"timestamp"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	ForwardStatus int             `json:"forward_status"`
}

func decodeShown(t *testing.T, stdout string) []shownEvent {
	t.Helper()
	var out []shownEvent
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		var e shownEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("output line is not an event: %v\n%s", err, line)
		}
		out = append(out, e)
	}
	return out
}

func offsetsOf(events []shownEvent) []int64 {
	out := []int64{}
	for _, e := range events {
		out = append(out, e.Offset)
	}
	return out
}

func span(from, to int64) []int64 {
	out := []int64{}
	for n := from; n < to; n++ {
		out = append(out, n)
	}
	return out
}

func TestEventsTailDeliversInOrderAndCommitsAfterShowing(t *testing.T) {
	mixed := []string{tailEvent(0, tailTestAccountA), tailEvent(1, tailTestAccountB), tailEvent(2, tailTestAccountA), tailEvent(3, tailTestAccountB)}
	cases := []struct {
		name         string
		args         []string
		events       []string
		arriving     []string
		pollFailures []int
		failForward  map[int]int // event index -> failed deliveries before success
		wantShown    []int64
		wantCommits  []int64
		wantStart    string
		wantWarning  string
	}{
		{
			name:        "replays retained history in order",
			events:      tailEvents(0, 3, tailTestAccountA),
			wantShown:   span(0, 3),
			wantCommits: []int64{2},
			wantStart:   "earliest",
		},
		{
			name:        "commits each batch after showing it",
			events:      tailEvents(0, 150, tailTestAccountA),
			wantShown:   span(0, 150),
			wantCommits: []int64{99, 149},
			wantStart:   "earliest",
		},
		{
			name:        "from now skips history and shows new events",
			args:        []string{"--from-now"},
			events:      tailEvents(0, 3, tailTestAccountA),
			arriving:    tailEvents(3, 5, tailTestAccountA),
			wantShown:   span(3, 5),
			wantCommits: []int64{4},
			wantStart:   "latest",
		},
		{
			name:        "account filter shows one account and commits past the others",
			args:        []string{"--account-id", strings.ToUpper(tailTestAccountA)},
			events:      mixed,
			wantShown:   []int64{0, 2},
			wantCommits: []int64{3},
			wantStart:   "earliest",
		},
		{
			name:         "transient poll failures are retried",
			events:       tailEvents(0, 2, tailTestAccountA),
			pollFailures: []int{http.StatusInternalServerError, http.StatusTooManyRequests},
			wantShown:    span(0, 2),
			wantCommits:  []int64{1},
			wantStart:    "earliest",
			wantWarning:  "warning: poll failed: HTTP 500",
		},
		{
			name:         "a held lease is waited out",
			events:       tailEvents(0, 2, tailTestAccountA),
			pollFailures: []int{http.StatusLocked},
			wantShown:    span(0, 2),
			wantCommits:  []int64{1},
			wantStart:    "earliest",
			wantWarning:  "HTTP 423: this consumer holds a leased batch",
		},
		{
			name:        "forwarding delivers each event in order",
			args:        []string{"--forward-to"},
			events:      tailEvents(0, 3, tailTestAccountA),
			wantShown:   span(0, 3),
			wantCommits: []int64{2},
			wantStart:   "earliest",
		},
		{
			name:        "a failing local URL is retried in place",
			args:        []string{"--forward-to"},
			events:      tailEvents(0, 3, tailTestAccountA),
			failForward: map[int]int{1: 2},
			wantShown:   span(0, 3),
			wantCommits: []int64{2},
			wantStart:   "earliest",
			wantWarning: "warning: forward of offset 1 to ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			poller := newFakePoller(append([]string(nil), tc.events...))
			poller.arriving = tc.arriving
			poller.failures = tc.pollFailures
			poller.stop = cancel
			server := httptest.NewServer(poller)
			defer server.Close()
			isolateEventsTail(t, server)

			args := append([]string(nil), tc.args...)
			var handler *fakeHandler
			if len(args) > 0 && args[len(args)-1] == "--forward-to" {
				handler = &fakeHandler{failFirst: map[string]int{}}
				for i, n := range tc.failForward {
					handler.failFirst[tc.events[i]] = n
				}
				local := httptest.NewServer(handler)
				defer local.Close()
				args = append(args, local.URL+"/webhooks")
			}

			stdout, stderr, err := runEventsTail(t, ctx, args...)
			if err != nil {
				t.Fatalf("tail returned %v\nstderr: %s", err, stderr)
			}
			if !poller.stopped {
				t.Fatalf("tail never caught up; commits %v\nstderr: %s", poller.commits, stderr)
			}
			shown := decodeShown(t, stdout)
			if got := offsetsOf(shown); fmt.Sprint(got) != fmt.Sprint(tc.wantShown) {
				t.Errorf("shown offsets = %v, want %v", got, tc.wantShown)
			}
			all := append(tc.events, tc.arriving...)
			for _, e := range shown {
				if string(e.Payload) != all[e.Offset] {
					t.Errorf("offset %d payload = %s, want the delivered bytes %s", e.Offset, e.Payload, all[e.Offset])
				}
				if e.EventType != "charge.event.v1" || !strings.HasPrefix(e.Timestamp, "2026-09-30T15:10:") {
					t.Errorf("offset %d envelope = %+v", e.Offset, e)
				}
			}
			if fmt.Sprint(poller.commits) != fmt.Sprint(tc.wantCommits) {
				t.Errorf("commits = %v, want %v", poller.commits, tc.wantCommits)
			}
			if poller.starts[0] != tc.wantStart {
				t.Errorf("starting_position = %q, want %q", poller.starts[0], tc.wantStart)
			}
			if tc.wantWarning != "" && !strings.Contains(stderr, tc.wantWarning) {
				t.Errorf("stderr lacks %q:\n%s", tc.wantWarning, stderr)
			}
			if handler == nil {
				return
			}
			var wantDelivered []string
			for _, off := range tc.wantShown {
				wantDelivered = append(wantDelivered, all[off])
			}
			if fmt.Sprint(handler.delivered) != fmt.Sprint(wantDelivered) {
				t.Errorf("local handler received %v, want %v", handler.delivered, wantDelivered)
			}
			for _, e := range shown {
				if e.ForwardStatus != http.StatusOK {
					t.Errorf("offset %d forward_status = %d, want 200", e.Offset, e.ForwardStatus)
				}
			}
		})
	}
}

// A restart with the same consumer resumes after the last event shown: no
// event is shown twice and none is skipped, including when the first run
// stopped mid-batch because the local handler was failing.
func TestEventsTailRestartResumesWithoutGapsOrDuplicates(t *testing.T) {
	cases := []struct {
		name         string
		forward      bool
		wantFirst    []int64
		wantSecond   []int64
		wantCommits  []int64
		wantRestart  string
		secondEvents []string
	}{
		{
			name:         "stopped between batches",
			wantFirst:    span(0, 3),
			secondEvents: tailEvents(3, 5, tailTestAccountA),
			wantSecond:   span(3, 5),
			wantCommits:  []int64{2, 4},
		},
		{
			name:        "stopped while the local handler failed mid-batch",
			forward:     true,
			wantFirst:   []int64{0},
			wantSecond:  span(1, 3),
			wantCommits: []int64{0, 2},
			wantRestart: "HTTP 423",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			poller := newFakePoller(tailEvents(0, 3, tailTestAccountA))
			poller.expireOnLocked = true
			server := httptest.NewServer(poller)
			defer server.Close()
			isolateEventsTail(t, server)

			handler := &fakeHandler{failFirst: map[string]int{}}
			local := httptest.NewServer(handler)
			defer local.Close()
			args := []string{"--consumer", "restart-test"}
			if tc.forward {
				args = append(args, "--forward-to", local.URL)
			}

			// First run: in the forwarding case the handler fails event 1
			// until the developer stops the tail.
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			poller.stop = cancel
			if tc.forward {
				handler.failFirst[poller.events[1]] = 1000
				handler.stopAfter = 2
				handler.stop = cancel
			}
			first, stderr, err := runEventsTail(t, ctx, args...)
			if err != nil {
				t.Fatalf("first run: %v\n%s", err, stderr)
			}

			// Second run with the same consumer and a healthy handler.
			poller.mu.Lock()
			poller.events = append(poller.events, tc.secondEvents...)
			poller.stopped = false
			poller.mu.Unlock()
			handler.mu.Lock()
			handler.failFirst = map[string]int{}
			handler.mu.Unlock()
			ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel2()
			poller.stop = cancel2
			second, stderr2, err := runEventsTail(t, ctx2, args...)
			if err != nil {
				t.Fatalf("second run: %v\n%s", err, stderr2)
			}
			if !poller.stopped {
				t.Fatalf("second run never caught up; commits %v\n%s", poller.commits, stderr2)
			}

			if got := offsetsOf(decodeShown(t, first)); fmt.Sprint(got) != fmt.Sprint(tc.wantFirst) {
				t.Errorf("first run showed %v, want %v", got, tc.wantFirst)
			}
			if got := offsetsOf(decodeShown(t, second)); fmt.Sprint(got) != fmt.Sprint(tc.wantSecond) {
				t.Errorf("second run showed %v, want %v", got, tc.wantSecond)
			}
			if fmt.Sprint(poller.commits) != fmt.Sprint(tc.wantCommits) {
				t.Errorf("commits = %v, want %v", poller.commits, tc.wantCommits)
			}
			if tc.wantRestart != "" && !strings.Contains(stderr2, tc.wantRestart) {
				t.Errorf("second run stderr lacks %q:\n%s", tc.wantRestart, stderr2)
			}
			if tc.forward && fmt.Sprint(handler.delivered) != fmt.Sprint(poller.events) {
				t.Errorf("local handler accepted %v, want each event once in order %v", handler.delivered, poller.events)
			}
		})
	}
}

// brokenStdout is stdout piped to a reader that goes away, like
// `straddle events tail | head -n 1`: it accepts writes, then fails.
type brokenStdout struct{ writes int }

func (w *brokenStdout) Write(p []byte) (int, error) {
	if w.writes == 0 {
		return 0, io.ErrClosedPipe
	}
	w.writes--
	return len(p), nil
}

// An event the local handler accepted is delivered even when printing it
// fails, so a restart does not forward it a second time.
func TestEventsTailForwardedEventIsNotForwardedAgainAfterStdoutBreaks(t *testing.T) {
	poller := newFakePoller(tailEvents(0, 2, tailTestAccountA))
	poller.expireOnLocked = true
	server := httptest.NewServer(poller)
	defer server.Close()
	isolateEventsTail(t, server)
	handler := &fakeHandler{failFirst: map[string]int{}}
	local := httptest.NewServer(handler)
	defer local.Close()
	args := []string{"--consumer", "broken-stdout", "--forward-to", local.URL}

	// First run: stdout breaks after event 0 is printed, so event 1 is
	// forwarded but cannot be printed.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	poller.stop = cancel
	cmd := RootCmd()
	var stderr bytes.Buffer
	cmd.SetOut(&brokenStdout{writes: 1})
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"events", "tail", "--interval", "1ms"}, args...))
	if err := cmd.ExecuteContext(ctx); err == nil || !strings.Contains(err.Error(), "writing event at offset 1") {
		t.Fatalf("first run error = %v, want the failed write of offset 1\n%s", err, stderr.String())
	}

	// Second run with the same consumer and a working stdout.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel2()
	poller.mu.Lock()
	poller.stop = cancel2
	poller.mu.Unlock()
	if _, stderr2, err := runEventsTail(t, ctx2, args...); err != nil {
		t.Fatalf("second run: %v\n%s", err, stderr2)
	}
	if !poller.stopped {
		t.Fatalf("second run never caught up; commits %v", poller.commits)
	}
	if fmt.Sprint(handler.delivered) != fmt.Sprint(poller.events) {
		t.Errorf("local handler accepted %v, want each event once in order %v", handler.delivered, poller.events)
	}
}

// Permanent failures exit with the CLI's exit codes, commit nothing past
// what was shown, and never print the polling token, even when the endpoint
// echoes it back.
func TestEventsTailPermanentFailures(t *testing.T) {
	okBatch := fmt.Sprintf(`{"data":[{"offset":0,"eventType":"charge.event.v1","payload":%s,"timestamp":"2026-09-30T15:10:26Z"}],"done":true}`, tailEvent(0, tailTestAccountA))
	cases := []struct {
		name         string
		pollStatus   int
		pollBody     string
		commitStatus int
		wantExit     int
		wantErr      string
		wantShown    int
		wantCommits  int
	}{
		{name: "rejected token", pollStatus: 401, pollBody: `{"detail":"bad token ` + tailTestToken + `"}`, wantExit: 4, wantErr: "poll: HTTP 401"},
		{name: "unknown endpoint", pollStatus: 404, pollBody: `{"detail":"not found"}`, wantExit: 3, wantErr: "STRADDLE_POLLING_URL"},
		{name: "invalid request", pollStatus: 400, pollBody: `{"detail":"consumer group name must only contain alphanumeric characters"}`, wantExit: 5, wantErr: "alphanumeric"},
		{name: "a redirect is not followed", pollStatus: 307, pollBody: ``, wantExit: 5, wantErr: "HTTP 307"},
		{name: "response without a data array", pollStatus: 200, pollBody: `{"done":true}`, wantExit: 5, wantErr: "unexpected poll response"},
		{name: "offsets out of order", pollStatus: 200, pollBody: `{"data":[{"offset":5,"payload":{}},{"offset":4,"payload":{}}],"done":true}`, wantExit: 5, wantErr: "offset 4 follows offset 5"},
		{name: "message without an event", pollStatus: 200, pollBody: `{"data":[{"offset":0,"payload":null}],"done":true}`, wantExit: 5, wantErr: "unusable event payload"},
		{name: "rejected commit", pollStatus: 200, pollBody: okBatch, commitStatus: 403, wantExit: 4, wantErr: "shown again on the next run", wantShown: 1, wantCommits: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var commits int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/commit") {
					commits++
					w.WriteHeader(tc.commitStatus)
					return
				}
				if tc.pollStatus == 307 {
					w.Header().Set("Location", "http://127.0.0.1:1/elsewhere")
				}
				w.WriteHeader(tc.pollStatus)
				_, _ = io.WriteString(w, tc.pollBody)
			}))
			defer server.Close()
			isolateEventsTail(t, server)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			stdout, stderr, err := runEventsTail(t, ctx)
			if err == nil {
				t.Fatal("tail succeeded, want a permanent failure")
			}
			if got := ExitCode(err); got != tc.wantExit {
				t.Errorf("exit code = %d, want %d (%v)", got, tc.wantExit, err)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q lacks %q", err, tc.wantErr)
			}
			for name, text := range map[string]string{"error": err.Error(), "stdout": stdout, "stderr": stderr} {
				if strings.Contains(text, tailTestToken) {
					t.Errorf("%s printed the polling token: %s", name, text)
				}
			}
			if got := len(decodeShown(t, stdout)); got != tc.wantShown {
				t.Errorf("shown %d events, want %d", got, tc.wantShown)
			}
			if commits != tc.wantCommits {
				t.Errorf("commits = %d, want %d", commits, tc.wantCommits)
			}
		})
	}
}

// Invalid flags and missing or unsafe polling settings fail before any
// request is sent.
func TestEventsTailRejectsInvalidSettings(t *testing.T) {
	cases := []struct {
		name     string
		url      string // STRADDLE_POLLING_URL; "server" means the fake's URL
		token    string
		args     []string
		wantExit int
		wantErr  string
	}{
		{name: "no polling URL", token: tailTestToken, wantExit: 10, wantErr: "no notification polling URL"},
		{name: "no polling token", url: "server", wantExit: 10, wantErr: "no notification polling token"},
		{name: "URL without consumer placeholder", url: "https://api.us.svix.com/api/v1/app/app_x/polling-endpoint/poll_x/consumer/me", token: tailTestToken, wantExit: 10, wantErr: "{consumer_id}"},
		{name: "plaintext URL off loopback", url: "http://api.us.svix.com/consumer/{consumer_id}", token: tailTestToken, wantExit: 10, wantErr: "must use https"},
		{name: "invalid consumer ID", url: "server", token: tailTestToken, args: []string{"--consumer", "has space"}, wantExit: 2, wantErr: "--consumer"},
		{name: "consumer ID over 64 bytes", url: "server", token: tailTestToken, args: []string{"--consumer", strings.Repeat("x", 65)}, wantExit: 2, wantErr: "--consumer"},
		{name: "account ID that is not a UUID", url: "server", token: tailTestToken, args: []string{"--account-id", "acct_123"}, wantExit: 2, wantErr: "--account-id"},
		{name: "relative forward URL", url: "server", token: tailTestToken, args: []string{"--forward-to", "localhost:3000/hooks"}, wantExit: 2, wantErr: "--forward-to"},
		{name: "zero interval", url: "server", token: tailTestToken, args: []string{"--interval", "0s"}, wantExit: 2, wantErr: "--interval"},
		{name: "account header flag", url: "server", token: tailTestToken, args: []string{"--account", tailTestAccountA}, wantExit: 2, wantErr: "remove --account"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				w.WriteHeader(http.StatusTeapot)
			}))
			defer server.Close()
			isolateEventsTail(t, server)
			pollingURL := tc.url
			if pollingURL == "server" {
				pollingURL = server.URL + "/consumer/{consumer_id}"
			}
			t.Setenv("STRADDLE_POLLING_URL", pollingURL)
			t.Setenv("STRADDLE_POLLING_TOKEN", tc.token)

			stdout, stderr, err := runEventsTail(t, context.Background(), tc.args...)
			if err == nil {
				t.Fatal("tail accepted invalid settings")
			}
			if got := ExitCode(err); got != tc.wantExit {
				t.Errorf("exit code = %d, want %d (%v)", got, tc.wantExit, err)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q lacks %q", err, tc.wantErr)
			}
			if requests != 0 {
				t.Errorf("sent %d requests before validation failed", requests)
			}
			if strings.Contains(stdout+stderr+err.Error(), tailTestToken) {
				t.Error("printed the polling token")
			}
		})
	}
}

// A dry run describes the resolved requests as a stable JSON plan, masks
// the token, and sends nothing. --polling-url wins over the environment,
// which wins over config.toml.
func TestEventsTailDryRun(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
	}))
	defer server.Close()
	isolateEventsTail(t, server)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("polling_url = 'https://config.example/consumer/{consumer_id}'\npolling_token = 'config-token'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STRADDLE_CONFIG", configPath)

	cases := []struct {
		name string
		env  string
		args []string
		want string
	}{
		{
			name: "config file",
			want: `{"dry_run":true,"consumer":"straddle-cli","starting_position":"earliest","poll_url":"https://config.example/consumer/straddle-cli?limit=100&starting_position=earliest","commit_url":"https://config.example/consumer/straddle-cli/commit"}`,
		},
		{
			name: "environment over config file",
			env:  "https://env.example/consumer/{consumer_id}",
			args: []string{"--consumer", "ci-1", "--from-now", "--account-id", tailTestAccountA, "--forward-to", "http://localhost:3000/hooks?a=1&b=2"},
			want: `{"dry_run":true,"consumer":"ci-1","starting_position":"latest","poll_url":"https://env.example/consumer/ci-1?limit=100&starting_position=latest","commit_url":"https://env.example/consumer/ci-1/commit","account_id":"` + tailTestAccountA + `","forward_to":"http://localhost:3000/hooks?a=1&b=2"}`,
		},
		{
			name: "flag over environment",
			env:  "https://env.example/consumer/{consumer_id}",
			args: []string{"--polling-url", "https://flag.example/c/{consumer_id}"},
			want: `{"dry_run":true,"consumer":"straddle-cli","starting_position":"earliest","poll_url":"https://flag.example/c/straddle-cli?limit=100&starting_position=earliest","commit_url":"https://flag.example/c/straddle-cli/commit"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_POLLING_URL", tc.env)
			t.Setenv("STRADDLE_POLLING_TOKEN", "")
			stdout, stderr, err := runEventsTail(t, context.Background(), append([]string{"--dry-run", "--agent"}, tc.args...)...)
			if err != nil {
				t.Fatalf("dry run: %v\n%s", err, stderr)
			}
			if stdout != tc.want+"\n" {
				t.Errorf("dry-run plan =\n%s\nwant\n%s", stdout, tc.want)
			}
			if !strings.Contains(stderr, "Authorization: Bearer ****") || strings.Contains(stderr, "config-token") {
				t.Errorf("dry run must mask the token:\n%s", stderr)
			}
		})
	}
	if requests != 0 {
		t.Errorf("dry run sent %d requests", requests)
	}
}

// In a terminal each event is one readable line; progress goes to stderr.
func TestEventsTailHumanOutput(t *testing.T) {
	terminal := true
	isTerminalOverride = &terminal
	defer func() { isTerminalOverride = nil }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	poller := newFakePoller(tailEvents(0, 2, tailTestAccountA))
	poller.stop = cancel
	server := httptest.NewServer(poller)
	defer server.Close()
	isolateEventsTail(t, server)

	stdout, stderr, err := runEventsTail(t, ctx, "--consumer", "human")
	if err != nil {
		t.Fatalf("tail: %v\n%s", err, stderr)
	}
	want := "2026-09-30T15:10:00Z  charge.event.v1  id=ch-0  status=paid  account=" + tailTestAccountA + "\n" +
		"2026-09-30T15:10:01Z  charge.event.v1  id=ch-1  status=paid  account=" + tailTestAccountA + "\n"
	if stdout != want {
		t.Errorf("human output =\n%s\nwant\n%s", stdout, want)
	}
	for _, line := range []string{`Tailing events as consumer "human"`, `Stopped; consumer "human" resumes after the last event shown.`} {
		if !strings.Contains(stderr, line) {
			t.Errorf("stderr lacks %q:\n%s", line, stderr)
		}
	}
}
