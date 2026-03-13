package github

import (
	"context"

	"github-note/internal/domain"
)

// Client is the abstraction for GitHub API operations used by the app.
type Client interface {
	EnsureToken(ctx context.Context) error
	LoadIssuePage(ctx context.Context, owner string, repo string, page int, pageSize int) (*domain.IssueTree, bool, error)
	RefreshIssuePage(ctx context.Context, owner string, repo string, page int, pageSize int) (*domain.IssueTree, bool, error)
}
