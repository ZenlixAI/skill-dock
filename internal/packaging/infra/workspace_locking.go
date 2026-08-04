package infra

import (
	"context"
	"strings"
	"sync"
)

type workspaceLockEntry struct {
	mu   sync.Mutex
	refs int
}

type workspaceLockContextKey struct{}

type workspaceLocker struct {
	mu    sync.Mutex
	locks map[string]*workspaceLockEntry
}

func newWorkspaceLocker() *workspaceLocker {
	return &workspaceLocker{locks: map[string]*workspaceLockEntry{}}
}

func (l *workspaceLocker) lock(ctx context.Context, workspaceID string, enabled bool) (context.Context, func(), error) {
	if !enabled || strings.TrimSpace(workspaceID) == "" {
		return ctx, func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if current, ok := ctx.Value(workspaceLockContextKey{}).(string); ok && current == workspaceID {
		return ctx, func() {}, nil
	}

	l.mu.Lock()
	entry := l.locks[workspaceID]
	if entry == nil {
		entry = &workspaceLockEntry{}
		l.locks[workspaceID] = entry
	}
	entry.refs++
	l.mu.Unlock()

	entry.mu.Lock()

	release := func() {
		entry.mu.Unlock()

		l.mu.Lock()
		defer l.mu.Unlock()

		entry.refs--
		if entry.refs == 0 && l.locks[workspaceID] == entry {
			delete(l.locks, workspaceID)
		}
	}

	return context.WithValue(ctx, workspaceLockContextKey{}, workspaceID), release, nil
}

func (l *workspaceLocker) Lock(ctx context.Context, workspaceID string, enabled bool) (context.Context, func(), error) {
	return l.lock(ctx, workspaceID, enabled)
}
