package usecase

import (
	"context"
	"errors"
	"sync"

	"github.com/tfnick/go-svelte-starter/api/db"
)

var ErrNoActiveAppTx = errors.New("no active app transaction")

type AfterCommitFunc func(context.Context)

type appTxHooksContextKey struct{}

type appTxHooks struct {
	mu          sync.Mutex
	active      bool
	afterCommit []AfterCommitFunc
}

// WithAppTx runs fn with an app transaction attached to the usecase context.
func WithAppTx(ctx Context, fn func(Context) error) error {
	if hooks := appHooksFromContext(ctx.Std()); hooks != nil && hooks.isActive() {
		return db.WithTx(ctx.Std(), "app", func(txCtxStd context.Context) error {
			return fn(ctx.WithStd(txCtxStd))
		})
	}

	hooks := &appTxHooks{active: true}
	stdWithHooks := context.WithValue(ctx.Std(), appTxHooksContextKey{}, hooks)
	txCtx := ctx.WithStd(stdWithHooks)

	if err := db.WithTx(stdWithHooks, "app", func(txCtxStd context.Context) error {
		return fn(txCtx.WithStd(txCtxStd))
	}); err != nil {
		hooks.deactivate()
		return err
	}

	hooks.runAfterCommit(ctx.Std())
	return nil
}

func InAppTx(ctx Context) bool {
	hooks := appHooksFromContext(ctx.Std())
	return hooks != nil && hooks.isActive()
}

func RegisterAfterCommit(ctx Context, fn AfterCommitFunc) error {
	if fn == nil {
		return nil
	}

	hooks := appHooksFromContext(ctx.Std())
	if hooks == nil || !hooks.isActive() {
		return ErrNoActiveAppTx
	}

	hooks.addAfterCommit(fn)
	return nil
}

func appHooksFromContext(ctx context.Context) *appTxHooks {
	if ctx == nil {
		return nil
	}
	hooks, _ := ctx.Value(appTxHooksContextKey{}).(*appTxHooks)
	return hooks
}

func (h *appTxHooks) addAfterCommit(fn AfterCommitFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.afterCommit = append(h.afterCommit, fn)
}

func (h *appTxHooks) runAfterCommit(ctx context.Context) {
	callbacks := h.deactivateAndDrainAfterCommit()
	for _, fn := range callbacks {
		fn(ctx)
	}
}

func (h *appTxHooks) isActive() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.active
}

func (h *appTxHooks) deactivate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.active = false
	h.afterCommit = nil
}

func (h *appTxHooks) deactivateAndDrainAfterCommit() []AfterCommitFunc {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.active = false
	callbacks := h.afterCommit
	h.afterCommit = nil
	if len(callbacks) == 0 {
		return nil
	}

	next := make([]AfterCommitFunc, len(callbacks))
	copy(next, callbacks)
	return next
}
