// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/config"
	"github.com/straddle-build/straddle-cli/internal/store"
	"github.com/straddle-build/straddle-cli/internal/straddleacct"
)

// Local store scope is the API environment plus the selected platform
// acting account. It is deliberately separate from the Straddle-Account-Id
// header: a marketplace fetches customers without the header, yet rows it
// captured while acting as one account must not appear under another.

const storeScopeAnnotation = "straddle:store-scope"

// markStoreScoped lets a local-store command accept --account as its store
// context. These commands send no account header of their own; sync
// applies the header policy per request instead.
func markStoreScoped(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[storeScopeAnnotation] = "true"
	return cmd
}

type storeSelection struct {
	configPath     string
	account        string
	accountChanged bool
	// platform is the sticky context read once when the command started;
	// it is never reloaded, so use-account in another process cannot
	// move this command's rows.
	platform straddleacct.Context
	recorded bool
}

type (
	storeSelectionKey struct{}
	requestScopeKey   struct{}
)

// recordStoreSelection keeps the command's config path, --account choice
// and platform context on its context so every store opened for it
// resolves one scope.
func recordStoreSelection(cmd *cobra.Command, f *rootFlags, platform straddleacct.Context) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, storeSelectionKey{}, storeSelection{
		configPath:     f.configPath,
		account:        f.straddleAccount,
		accountChanged: cmd.Flags().Changed("account"),
		platform:       platform,
		recorded:       true,
	}))
}

func selectionFrom(ctx context.Context) storeSelection {
	if ctx == nil {
		return storeSelection{}
	}
	selection, _ := ctx.Value(storeSelectionKey{}).(storeSelection)
	return selection
}

// platformContext returns the command's recorded platform context, or
// reads it when no command recorded one.
func platformContext(ctx context.Context) (straddleacct.Context, error) {
	if selection := selectionFrom(ctx); selection.recorded {
		return selection.platform, nil
	}
	return straddleacct.LoadContext()
}

// withRequestScope pins the store scope for one live read, resolved from
// the client that sends it, so its write-through and any offline
// fallback use the request's origin and acting account.
func withRequestScope(ctx context.Context, c *client.Client) (context.Context, store.Scope, error) {
	var templateVars map[string]string
	if c.Config != nil {
		templateVars = c.Config.TemplateVars
	}
	scope, err := storeScopeFor(ctx, c.BaseURL, templateVars)
	if err != nil {
		return ctx, store.Scope{}, err
	}
	return context.WithValue(ctx, requestScopeKey{}, scope), scope, nil
}

// localStoreScope resolves the store scope for the running command from
// its config and platform context.
func localStoreScope(ctx context.Context) (store.Scope, error) {
	if ctx != nil {
		if scope, ok := ctx.Value(requestScopeKey{}).(store.Scope); ok {
			return scope, nil
		}
	}
	cfg, err := config.Load(selectionFrom(ctx).configPath)
	if err != nil {
		return store.Scope{}, configErr(err)
	}
	return storeScopeFor(ctx, cfg.BaseURL, cfg.TemplateVars)
}

// storeScopeFor resolves the scope for an explicit API base URL, used by
// sync after --path-context has adjusted the client's template values.
func storeScopeFor(ctx context.Context, baseURL string, templateVars map[string]string) (store.Scope, error) {
	environment, err := apiEnvironment(baseURL, templateVars)
	if err != nil {
		return store.Scope{}, err
	}
	platform, err := platformContext(ctx)
	if err != nil {
		return store.Scope{}, err
	}
	selection := selectionFrom(ctx)
	if platform.IntegrationType == straddleacct.TypeAccount {
		// A direct account has no acting-account context; reuse the
		// policy's rejection of an explicit --account.
		if _, _, err := straddleacct.Resolve(straddleacct.Forbid, selection.account, selection.accountChanged, ""); err != nil {
			return store.Scope{}, usageErr(err)
		}
		return store.Scope{Environment: environment}, nil
	}
	account := platform.CurrentAccount
	if selection.accountChanged {
		account = selection.account
	}
	return store.Scope{Environment: environment, Account: account}, nil
}

// apiEnvironment is the lowercase origin of the resolved API base URL, so
// sandbox, production and any local server never share rows.
func apiEnvironment(baseURL string, templateVars map[string]string) (string, error) {
	resolved := baseURL
	for name, value := range templateVars {
		resolved = strings.ReplaceAll(resolved, "{"+name+"}", value)
	}
	parsed, err := url.Parse(resolved)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || strings.ContainsAny(parsed.Host, "{}") {
		return "", configErr(fmt.Errorf("cannot determine the API environment for the local store from base URL %q", baseURL))
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), nil
}

// runtimeContext is the selection a command acts under: the API
// environment and acting account of its local store scope, plus the
// configured integration type. Null means not selected. It reports
// selection only; whether a request carries Straddle-Account-Id is still
// decided per operation by straddleacct.
type runtimeContext struct {
	Environment     *string `json:"environment"`
	IntegrationType *string `json:"integration_type"`
	ActingAccount   *string `json:"acting_account"`
	Error           string  `json:"error,omitempty"`
}

// resolveRuntimeContext resolves the running command's runtime context
// the same way its local store scope is resolved, without opening the
// store.
func resolveRuntimeContext(ctx context.Context) runtimeContext {
	var rc runtimeContext
	platform, err := platformContext(ctx)
	if err != nil {
		rc.Error = err.Error()
		return rc
	}
	if platform.IntegrationType != "" {
		rc.IntegrationType = &platform.IntegrationType
	}
	scope, err := localStoreScope(ctx)
	if err != nil {
		rc.Error = err.Error()
		return rc
	}
	rc.Environment = &scope.Environment
	if scope.Account != "" {
		rc.ActingAccount = &scope.Account
	}
	if platform.IntegrationType != "" && !straddleacct.ValidIntegrationType(platform.IntegrationType) {
		rc.Error = fmt.Sprintf("invalid integration type %q in %s: must be %s, %s, or %s; run 'straddle setup --type ...'",
			platform.IntegrationType, straddleacct.ContextPath(), straddleacct.TypeAccount, straddleacct.TypeSaaS, straddleacct.TypeMarketplace)
	}
	return rc
}

// openScopedStore opens the local store in the command's scope.
func openScopedStore(ctx context.Context, dbPath string) (*store.Store, error) {
	scope, err := localStoreScope(ctx)
	if err != nil {
		return nil, err
	}
	return store.OpenWithContext(ctx, dbPath, scope)
}

// hiddenLegacyRecords reports resources stored before scoping, or zero
// when the count cannot be read.
func hiddenLegacyRecords(db *store.Store) int {
	count, err := db.HiddenLegacyCount()
	if err != nil {
		return 0
	}
	return count
}

// syncGetter applies the existing Straddle-Account-Id policy to each sync
// request: the acting account is sent only where the operation accepts
// it for the configured integration type, never on forbidden operations.
type syncGetter struct {
	client          *client.Client
	integrationType string
	account         string
}

func newSyncGetter(ctx context.Context, c *client.Client, scope store.Scope) (syncGetter, error) {
	platform, err := platformContext(ctx)
	if err != nil {
		return syncGetter{}, err
	}
	return syncGetter{client: c, integrationType: platform.IntegrationType, account: scope.Account}, nil
}

func (g syncGetter) Get(path string, params map[string]string) (json.RawMessage, error) {
	var headers map[string]string
	decision := straddleacct.Classify(path, "GET", g.integrationType, acceptsAccountHeader(path, "GET"))
	if decision != straddleacct.Forbid && g.account != "" {
		headers = map[string]string{straddleacct.Header: g.account}
	}
	return g.client.GetWithHeaders(path, params, headers)
}

func (g syncGetter) RateLimit() float64 {
	return g.client.RateLimit()
}

// scopeLiveRead pins the read's store scope and turns off the HTTP
// response cache when the account header sent differs from that scope's
// account. The cache key includes the header, so skipping both cache
// reads and writes in that case means every cached response was fetched
// under the scope that reads it; a marketplace response cached while
// acting as one account can then never be replayed, and written through,
// while acting as another.
func scopeLiveRead(ctx context.Context, c *client.Client, headers map[string]string) context.Context {
	scoped, scope, err := withRequestScope(ctx, c)
	if c.NoCache {
		return scoped
	}
	sent := headers[straddleacct.Header]
	if sent == "" && c.Config != nil {
		sent = c.Config.Headers[straddleacct.Header]
	}
	if err != nil || scope.Account != sent {
		c.NoCache = true
	}
	return scoped
}
