package github

import (
	"context"

	"github-note/internal/domain"
)

// Client is the abstraction for GitHub API operations used by the app.
type Client interface {
	EnsureToken(ctx context.Context) error
	LoadIssueTree(ctx context.Context, owner string, repo string) (*domain.IssueTree, error)
}
