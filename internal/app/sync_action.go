package app

import (
	"context"
	"fmt"

	"github.com/morehao/starman/internal/store"
)

type StarLister interface {
	ListStarred(ctx context.Context, username string) ([]*store.Repository, error)
}

type syncAction struct {
	store    store.Store
	gh       StarLister
	username string
}

func NewSyncAction(s store.Store, gh StarLister, username string) SyncAction {
	return &syncAction{store: s, gh: gh, username: username}
}

func (a *syncAction) Run(ctx context.Context, opts SyncOpts) (*SyncResult, error) {
	repos, err := a.gh.ListStarred(ctx, a.username)
	if err != nil {
		return nil, fmt.Errorf("list starred: %w", err)
	}

	if err := a.store.UpsertReposOnSync(ctx, repos, opts.Full); err != nil {
		return nil, fmt.Errorf("sync to db: %w", err)
	}

	return &SyncResult{
		Fetched: len(repos),
	}, nil
}
