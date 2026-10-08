// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
//
// events tail reads Straddle notifications from the notification polling
// endpoint shown in the Straddle dashboard (a Svix polling endpoint). It is
// not a Straddle API operation: its requests carry only the polling token,
// never the API key or Straddle-Account-Id.
//
// Wire contract, observed against Sandbox on 2026-09-30:
//
//	GET  <url>?limit=N&starting_position=earliest|latest
//	     200 {"data":[{"offset":0,"id":"msg_…","eventType":"…","payload":{…},"timestamp":"…"}],"done":true}
//	     423 while an earlier batch for the consumer is leased and uncommitted
//	POST <url>/commit {"offset":N} -> 204
//
// starting_position applies only to a consumer with no position; its first
// poll pins one. Committing a batch's last offset releases the lease at once.
// Committing an earlier offset advances the position but keeps the lease
// until it expires (about five minutes), so a restart after a mid-batch stop
// sees 423 until then. A commit made after the lease expired still counts.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/config"
	"github.com/straddle-build/straddle-cli/internal/store"
)

const (
	defaultEventsConsumer = "straddle-cli"
	consumerIDPlaceholder = "{consumer_id}"
	// eventsBatchSize bounds one lease so a batch is delivered and committed
	// well inside the endpoint's five-minute lease.
	eventsBatchSize = 100
	// maxPollResponseBytes bounds memory for one batch of eventsBatchSize
	// events; a larger body fails to decode instead of growing unbounded.
	maxPollResponseBytes = 32 << 20
	eventsMaxRetryDelay  = 30 * time.Second
)

// eventsStopCommitTimeout bounds how long a commit keeps retrying after a
// stop request, so a failing commit endpoint cannot hold the first Ctrl+C
// open. It is a var so tests can shorten it.
var eventsStopCommitTimeout = time.Minute

// consumerIDPattern is the endpoint's own rule for consumer IDs.
var consumerIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

var errUnexpectedPollResponse = errors.New("unexpected poll response")

func newEventsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Read Straddle notifications from your polling endpoint",
		RunE:  parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newEventsTailCmd(flags))
	return cmd
}

type eventsTailOptions struct {
	consumer   string
	fromNow    bool
	accountID  string
	forwardTo  string
	pollingURL string
	interval   time.Duration
}

func newEventsTailCmd(flags *rootFlags) *cobra.Command {
	var opts eventsTailOptions
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Print notifications in order as they arrive, until stopped",
		Long: `Tail reads your notification polling endpoint and prints each event in order
until you stop it (Ctrl+C). It commits the consumer's position only after an
event is printed, or forwarded with --forward-to, so a restart resumes after
the last event shown with no gaps and no duplicates.

A new consumer starts at the earliest retained event and replays history for
every account on the endpoint; --from-now starts it at the newest event
instead. A consumer that already has a position always resumes from it.

Configure the endpoint shown in the Straddle dashboard: STRADDLE_POLLING_URL
(or --polling-url) is its URL including {consumer_id}, and
STRADDLE_POLLING_TOKEN is its token. Both can instead be saved as polling_url
and polling_token in config.toml. The token is never printed.

Output is one line per event in a terminal, and NDJSON when piped or with
--json/--agent: {"offset","timestamp","event_type","payload"} plus
"forward_status" when forwarding. payload is the Straddle event exactly as a
webhook would deliver it.`,
		Example: `  # Watch Sandbox notifications from now on
  straddle events tail --from-now

  # Only one embedded account's events, as NDJSON
  straddle events tail --account-id <account_id> --json

  # Re-send each event to a local handler, in order
  straddle events tail --forward-to http://localhost:3000/webhooks`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.validate(); err != nil {
				return usageErr(err)
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			endpoint, err := resolvePollingEndpoint(cfg, opts.pollingURL, opts.consumer, flags.timeout)
			if err != nil {
				return configErr(err)
			}
			tail := &eventsTail{
				endpoint:  endpoint,
				start:     "earliest",
				accountID: opts.accountID,
				interval:  opts.interval,
				out:       cmd.OutOrStdout(),
				warn:      cmd.ErrOrStderr(),
				asJSON:    straddleWantsJSON(cmd, flags),
			}
			tail.enc = json.NewEncoder(tail.out)
			tail.enc.SetEscapeHTML(false)
			if opts.fromNow {
				tail.start = "latest"
			}
			if opts.forwardTo != "" {
				tail.forward = &eventForwarder{url: opts.forwardTo, http: &http.Client{Timeout: flags.timeout}}
			}
			if flags.dryRun {
				return tail.printDryRun(opts.consumer)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			// After the first signal the final commit still runs, for at most
			// eventsStopCommitTimeout; restoring default handling lets a
			// second Ctrl+C exit at once.
			context.AfterFunc(ctx, stop)
			if !tail.asJSON {
				tail.printBanner(opts.consumer)
			}
			if err := tail.run(ctx); err != nil {
				return err
			}
			if !tail.asJSON {
				fmt.Fprintf(tail.warn, "Stopped; consumer %q resumes after the last event shown.\n", opts.consumer)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.consumer, "consumer", defaultEventsConsumer, "Consumer ID that keeps this reader's position; use a different one per terminal or teammate (letters, digits, '.', '_', '-'; at most 64)")
	cmd.Flags().BoolVar(&opts.fromNow, "from-now", false, "Start a new consumer at the newest event instead of replaying retained history (a consumer with a position always resumes)")
	cmd.Flags().StringVar(&opts.accountID, "account-id", "", "Only show events for this account ID; the consumer still moves past other accounts' events")
	cmd.Flags().StringVar(&opts.forwardTo, "forward-to", "", "POST each event's payload to this URL in order before showing it; a failing URL is retried and nothing after it is committed")
	cmd.Flags().StringVar(&opts.pollingURL, "polling-url", "", "Polling endpoint URL with {consumer_id} (overrides STRADDLE_POLLING_URL and polling_url in config.toml)")
	cmd.Flags().DurationVar(&opts.interval, "interval", 2*time.Second, "Wait between polls once caught up; failed requests retry after this, doubling up to 30s")
	return cmd
}

func (o eventsTailOptions) validate() error {
	if !consumerIDPattern.MatchString(o.consumer) {
		return fmt.Errorf("--consumer %q must be 1-64 letters, digits, '.', '_' or '-'", o.consumer)
	}
	if o.accountID != "" && !store.IsUUID(o.accountID) {
		return fmt.Errorf("--account-id %q is not an account ID (UUID)", o.accountID)
	}
	if o.interval <= 0 {
		return errors.New("--interval must be greater than zero")
	}
	if o.forwardTo == "" {
		return nil
	}
	u, err := url.Parse(o.forwardTo)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("--forward-to %q must be an absolute http or https URL", o.forwardTo)
	}
	return nil
}

// pollingEndpoint is one consumer's view of the notification polling endpoint.
type pollingEndpoint struct {
	consumerURL *url.URL // endpoint URL with the consumer ID filled in
	token       string
	http        *http.Client
}

func resolvePollingEndpoint(cfg *config.Config, flagURL, consumer string, timeout time.Duration) (*pollingEndpoint, error) {
	rawURL, token := cfg.Polling()
	if flagURL != "" {
		rawURL = flagURL
	}
	const hint = "\nhint: copy the polling endpoint URL and token from the Straddle dashboard, then export STRADDLE_POLLING_URL=<url> and STRADDLE_POLLING_TOKEN=<token>, or save polling_url and polling_token in config.toml"
	if rawURL == "" {
		return nil, errors.New("no notification polling URL configured" + hint)
	}
	if token == "" {
		return nil, errors.New("no notification polling token configured" + hint)
	}
	if !strings.Contains(rawURL, consumerIDPlaceholder) {
		return nil, fmt.Errorf("polling URL must contain %s, as shown in the Straddle dashboard", consumerIDPlaceholder)
	}
	filled := strings.ReplaceAll(rawURL, consumerIDPlaceholder, consumer)
	if err := config.RequireSecureURL("the polling URL", filled); err != nil {
		return nil, err
	}
	u, err := url.Parse(filled)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("polling URL %q is not an absolute URL", rawURL)
	}
	return &pollingEndpoint{
		consumerURL: u,
		token:       token,
		http: &http.Client{
			Timeout: timeout,
			// The token must reach only the configured endpoint, so a
			// redirect is reported as an error instead of being followed.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (p *pollingEndpoint) pollURL(start string) string {
	u := *p.consumerURL
	q := u.Query()
	q.Set("limit", strconv.Itoa(eventsBatchSize))
	q.Set("starting_position", start)
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *pollingEndpoint) commitURL() string {
	return p.consumerURL.JoinPath("commit").String()
}

// poll leases the consumer's next batch.
func (p *pollingEndpoint) poll(ctx context.Context, start string) ([]polledMessage, bool, error) {
	body, err := p.do(ctx, http.MethodGet, p.pollURL(start), nil)
	if err != nil {
		return nil, false, err
	}
	return decodePollResponse(body)
}

// commit moves the consumer's position to offset.
func (p *pollingEndpoint) commit(ctx context.Context, offset int64) error {
	_, err := p.do(ctx, http.MethodPost, p.commitURL(), []byte(`{"offset":`+strconv.FormatInt(offset, 10)+`}`))
	return err
}

func (p *pollingEndpoint) do(ctx context.Context, method, target string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", client.UserAgent(Version(), "events"))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPollResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &pollingStatusError{status: resp.StatusCode, body: p.scrub(data)}
	}
	return data, nil
}

// scrub prepares a response body for an error message: bounded, and never
// carrying the token even if the endpoint echoed it.
func (p *pollingEndpoint) scrub(body []byte) string {
	text := strings.ReplaceAll(string(body), p.token, "[REDACTED]")
	return strings.TrimSpace(truncate(text, 300))
}

// pollingStatusError is a non-2xx answer from the polling endpoint.
type pollingStatusError struct {
	status int
	body   string
}

func (e *pollingStatusError) Error() string {
	if e.status == http.StatusLocked {
		return "HTTP 423: this consumer holds a leased batch that was never committed (another process may be using the same --consumer, or an earlier run stopped mid-batch); the lease expires after about 5 minutes"
	}
	return fmt.Sprintf("HTTP %d: %s", e.status, e.body)
}

// isTransientPollingError reports whether a poll or commit is worth retrying:
// a held lease, throttling, a server error, or a transport failure. An
// unusable response or any other client error is permanent.
func isTransientPollingError(err error) bool {
	var se *pollingStatusError
	if errors.As(err, &se) {
		return se.status == http.StatusLocked || se.status == http.StatusTooManyRequests || se.status >= 500
	}
	return !errors.Is(err, errUnexpectedPollResponse)
}

// pollingFailure maps a permanent polling error to the CLI's exit codes.
func pollingFailure(action string, err error) error {
	wrapped := fmt.Errorf("%s: %w", action, err)
	var se *pollingStatusError
	if !errors.As(err, &se) {
		return apiErr(wrapped)
	}
	switch se.status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return authErr(fmt.Errorf("%w\nhint: check STRADDLE_POLLING_TOKEN (or polling_token in config.toml) against the token shown with the polling endpoint in the Straddle dashboard", wrapped))
	case http.StatusNotFound:
		return notFoundErr(fmt.Errorf("%w\nhint: check STRADDLE_POLLING_URL (or --polling-url) against the polling endpoint URL in the Straddle dashboard", wrapped))
	}
	return apiErr(wrapped)
}

// polledMessage is one validated message from a poll response.
type polledMessage struct {
	offset    int64
	timestamp string
	eventType string
	payload   json.RawMessage // the Straddle event, exactly as delivered
	accountID string
	// dataID and dataStatus summarize the event's resource for human output.
	dataID     string
	dataStatus string
}

func decodePollResponse(body []byte) ([]polledMessage, bool, error) {
	var wire struct {
		Data *[]struct {
			Offset    *int64          `json:"offset"`
			Timestamp string          `json:"timestamp"`
			EventType string          `json:"eventType"`
			Payload   json.RawMessage `json:"payload"`
		} `json:"data"`
		Done *bool `json:"done"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, false, fmt.Errorf("%w: %w", errUnexpectedPollResponse, err)
	}
	if wire.Data == nil || wire.Done == nil {
		return nil, false, fmt.Errorf("%w: no data array or done flag", errUnexpectedPollResponse)
	}
	msgs := make([]polledMessage, 0, len(*wire.Data))
	for i, item := range *wire.Data {
		if item.Offset == nil || *item.Offset < 0 {
			return nil, false, fmt.Errorf("%w: message %d has no offset", errUnexpectedPollResponse, i)
		}
		if i > 0 && *item.Offset <= msgs[i-1].offset {
			return nil, false, fmt.Errorf("%w: offset %d follows offset %d", errUnexpectedPollResponse, *item.Offset, msgs[i-1].offset)
		}
		var event struct {
			AccountID string          `json:"account_id"`
			Data      json.RawMessage `json:"data"`
		}
		if len(item.Payload) == 0 || item.Payload[0] != '{' || json.Unmarshal(item.Payload, &event) != nil {
			return nil, false, fmt.Errorf("%w: message at offset %d has an unusable event payload", errUnexpectedPollResponse, *item.Offset)
		}
		var data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		// Display only: an event whose data is absent or not an object
		// is still delivered, just without the id/status summary.
		_ = json.Unmarshal(event.Data, &data)
		msgs = append(msgs, polledMessage{
			offset:     *item.Offset,
			timestamp:  item.Timestamp,
			eventType:  item.EventType,
			payload:    item.Payload,
			accountID:  event.AccountID,
			dataID:     data.ID,
			dataStatus: data.Status,
		})
	}
	return msgs, *wire.Done, nil
}

// eventForwarder re-sends event payloads to a developer's local handler.
type eventForwarder struct {
	url  string
	http *http.Client
}

// send POSTs one event payload; any answer other than 2xx is a failure.
func (f *eventForwarder) send(ctx context.Context, payload json.RawMessage) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", client.UserAgent(Version(), "events"))
	resp, err := f.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// tailedEvent is one NDJSON output line. Field order is the contract.
type tailedEvent struct {
	Offset        int64           `json:"offset"`
	Timestamp     string          `json:"timestamp"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	ForwardStatus int             `json:"forward_status,omitempty"`
}

type eventsTail struct {
	endpoint  *pollingEndpoint
	start     string // starting_position for a new consumer: earliest or latest
	accountID string
	forward   *eventForwarder // nil unless --forward-to
	interval  time.Duration
	out       io.Writer
	warn      io.Writer
	asJSON    bool
	enc       *json.Encoder // NDJSON writer on out; HTML escaping off so payloads stay byte-faithful
}

// run polls, delivers and commits until ctx ends (Ctrl+C, SIGTERM) or a
// permanent failure. A requested stop returns nil.
func (t *eventsTail) run(ctx context.Context) error {
	for {
		var msgs []polledMessage
		var done bool
		err := t.retry(ctx, "poll", isTransientPollingError, func() error {
			var err error
			msgs, done, err = t.endpoint.poll(ctx, t.start)
			return err
		})
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return pollingFailure("poll", err)
		}

		last, delivered, deliverErr := t.deliver(ctx, msgs)
		if delivered {
			if err := t.commit(ctx, last); err != nil {
				return pollingFailure(fmt.Sprintf("commit of offset %d", last), fmt.Errorf("%w\nhint: events shown since the last commit will be shown again on the next run", err))
			}
		}
		if deliverErr != nil {
			return deliverErr
		}
		if ctx.Err() != nil {
			return nil
		}
		// An empty batch without done still waits, so the loop never spins.
		if (done || len(msgs) == 0) && !sleepCtx(ctx, t.interval) {
			return nil
		}
	}
}

// commit moves the consumer's position to last, retrying transient
// failures. It outlives a stop request, since an event already shown must be
// committed or a restart would show it again, but once ctx ends retries stop
// after eventsStopCommitTimeout. Each request stays uncancellable (bounded by
// --timeout), so the error returned is the endpoint's own failure.
func (t *eventsTail) commit(ctx context.Context, last int64) error {
	requestCtx := context.WithoutCancel(ctx)
	retryCtx, cancel := context.WithCancel(requestCtx)
	defer cancel()
	defer context.AfterFunc(ctx, func() {
		sleepCtx(retryCtx, eventsStopCommitTimeout)
		cancel()
	})()
	return t.retry(retryCtx, fmt.Sprintf("commit of offset %d", last), isTransientPollingError, func() error {
		return t.endpoint.commit(requestCtx, last)
	})
}

// deliver shows (and forwards) messages in order and returns the offset of
// the last one handled. Events filtered out by --account-id count as handled
// so the consumer moves past them, and so does a forwarded event whose
// printing then fails, since committing it is what stops a restart from
// forwarding it again. Delivery stops at the first event that cannot be
// delivered, so nothing after it is committed.
func (t *eventsTail) deliver(ctx context.Context, msgs []polledMessage) (last int64, handled bool, err error) {
	for _, m := range msgs {
		if t.accountID != "" && !strings.EqualFold(m.accountID, t.accountID) {
			last, handled = m.offset, true
			continue
		}
		status := 0
		if t.forward != nil {
			what := fmt.Sprintf("forward of offset %d to %s", m.offset, t.forward.url)
			err := t.retry(ctx, what, func(error) bool { return true }, func() error {
				var err error
				status, err = t.forward.send(ctx, m.payload)
				return err
			})
			if err != nil {
				// Forwarding retries until it succeeds or the tail stops.
				return last, handled, nil
			}
			last, handled = m.offset, true
		}
		if err := t.show(m, status); err != nil {
			return last, handled, fmt.Errorf("writing event at offset %d: %w", m.offset, err)
		}
		last, handled = m.offset, true
	}
	return last, handled, nil
}

// retry runs op until it succeeds, fails permanently, or ctx ends. Waits
// start at the poll interval and double up to eventsMaxRetryDelay. op
// always runs at least once, so a stop request still gets one attempt.
func (t *eventsTail) retry(ctx context.Context, what string, transient func(error) bool, op func() error) error {
	delay := t.interval
	for {
		err := op()
		if err == nil || !transient(err) || ctx.Err() != nil {
			return err
		}
		fmt.Fprintf(t.warn, "warning: %s failed: %v; retrying in %s\n", what, err, delay)
		if !sleepCtx(ctx, delay) {
			return err
		}
		delay = min(2*delay, eventsMaxRetryDelay)
	}
}

func (t *eventsTail) show(m polledMessage, forwardStatus int) error {
	if t.asJSON {
		return t.enc.Encode(tailedEvent{
			Offset:        m.offset,
			Timestamp:     m.timestamp,
			EventType:     m.eventType,
			Payload:       m.payload,
			ForwardStatus: forwardStatus,
		})
	}
	fields := []string{eventTime(m.timestamp), m.eventType}
	if m.dataID != "" {
		fields = append(fields, "id="+m.dataID)
	}
	if m.dataStatus != "" {
		fields = append(fields, "status="+m.dataStatus)
	}
	if m.accountID != "" {
		fields = append(fields, "account="+m.accountID)
	}
	if forwardStatus != 0 {
		fields = append(fields, "forwarded="+strconv.Itoa(forwardStatus))
	}
	_, err := fmt.Fprintln(t.out, strings.Join(fields, "  "))
	return err
}

func eventTime(raw string) string {
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return ts.UTC().Format(time.RFC3339)
}

func (t *eventsTail) printBanner(consumer string) {
	origin := "replays retained history"
	if t.start == "latest" {
		origin = "starts from now"
	}
	fmt.Fprintf(t.warn, "Tailing events as consumer %q: it resumes its position, or %s if it is new. Ctrl+C to stop.\n", consumer, origin)
	if t.accountID != "" {
		fmt.Fprintf(t.warn, "Showing only account %s.\n", t.accountID)
	}
	if t.forward != nil {
		fmt.Fprintf(t.warn, "Forwarding each event to %s.\n", t.forward.url)
	}
}

// printDryRun describes the requests tail would make without sending any.
// The token is shown only as a mask.
func (t *eventsTail) printDryRun(consumer string) error {
	fmt.Fprintf(t.warn, "GET %s\n  Authorization: Bearer ****\n", t.endpoint.pollURL(t.start))
	fmt.Fprintf(t.warn, "POST %s (after each batch is shown)\n  Authorization: Bearer ****\n", t.endpoint.commitURL())
	forwardTo := ""
	if t.forward != nil {
		forwardTo = t.forward.url
		fmt.Fprintf(t.warn, "POST %s (each event, in order)\n", forwardTo)
	}
	fmt.Fprintf(t.warn, "\n(dry run - no request sent)\n")
	if !t.asJSON {
		return nil
	}
	return t.enc.Encode(struct {
		DryRun    bool   `json:"dry_run"`
		Consumer  string `json:"consumer"`
		Start     string `json:"starting_position"`
		PollURL   string `json:"poll_url"`
		CommitURL string `json:"commit_url"`
		AccountID string `json:"account_id,omitempty"`
		ForwardTo string `json:"forward_to,omitempty"`
	}{true, consumer, t.start, t.endpoint.pollURL(t.start), t.endpoint.commitURL(), t.accountID, forwardTo})
}

// sleepCtx waits d or until ctx ends, and reports whether the wait finished.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
