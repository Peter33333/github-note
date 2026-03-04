package github

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github-note/internal/config"
	"github-note/internal/domain"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type issueNodeQuery struct {
	ID     githubv4.ID
	Number githubv4.Int
	Title  githubv4.String
	URL    githubv4.URI
	State  githubv4.String
	Labels struct {
		Nodes []struct {
			Name githubv4.String
		}
	} `graphql:"labels(first: 20)"`
	Parent *struct {
		ID githubv4.ID
	} `graphql:"parent"`
}

type issueTreeQuery struct {
	Repository struct {
		Issues struct {
			Nodes    []issueNodeQuery
			PageInfo struct {
				HasNextPage githubv4.Boolean
				EndCursor   githubv4.String
			}
		} `graphql:"issues(first: $first, after: $after, orderBy: {field: CREATED_AT, direction: ASC})"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// GitHubClient implements Client.
type GitHubClient struct {
	cfg        *config.Config
	httpClient *http.Client
	token      *oauth2.Token
}

func New(cfg *config.Config) *GitHubClient {
	return &GitHubClient{cfg: cfg, httpClient: &http.Client{}}
}

func (client *GitHubClient) EnsureToken(ctx context.Context) error {
	token, err := config.LoadToken()
	if err == nil {
		client.token = token
		return nil
	}

	deviceCode, err := requestDeviceCode(ctx, client.httpClient, client.cfg.ClientID)
	if err != nil {
		return err
	}

	fmt.Printf("Open %s and enter code: %s\n", deviceCode.VerificationURI, deviceCode.UserCode)
	fmt.Println("Waiting for authorization...")

	oauthToken, err := pollAccessToken(ctx, client.httpClient, client.cfg.ClientID, deviceCode.DeviceCode, deviceCode.Interval)
	if err != nil {
		return err
	}
	if err := config.SaveToken(oauthToken); err != nil {
		return err
	}
	client.token = oauthToken
	return nil
}

func (client *GitHubClient) LoadIssueTree(ctx context.Context, owner string, repo string) (*domain.IssueTree, error) {
	if client.token == nil {
		return nil, fmt.Errorf("token not ready")
	}
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(client.token))
	graphqlClient := githubv4.NewClient(oauthClient)

	tree := domain.NewIssueTree()
	var after *githubv4.String

	for {
		query := issueTreeQuery{}
		variables := map[string]interface{}{
			"owner": githubv4.String(owner),
			"repo":  githubv4.String(repo),
			"first": githubv4.Int(100),
			"after": after,
		}
		if err := graphqlClient.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("query issues: %w", err)
		}

		for _, n := range query.Repository.Issues.Nodes {
			labels := make([]string, 0, len(n.Labels.Nodes))
			for _, label := range n.Labels.Nodes {
				labels = append(labels, string(label.Name))
			}

			node := &domain.IssueNode{
				ID:     fmt.Sprint(n.ID),
				Number: int(n.Number),
				Title:  string(n.Title),
				Labels: labels,
				URL:    n.URL.String(),
				State:  string(n.State),
			}
			if n.Parent != nil {
				node.ParentID = fmt.Sprint(n.Parent.ID)
			}
			tree.AddNode(node)
		}

		if !bool(query.Repository.Issues.PageInfo.HasNextPage) {
			break
		}
		cursor := query.Repository.Issues.PageInfo.EndCursor
		after = &cursor
	}

	tree.BuildRoots()
	sortTree(tree)
	return tree, nil
}

func sortTree(tree *domain.IssueTree) {
	if tree == nil {
		return
	}
	sort.Slice(tree.Roots, func(i, j int) bool {
		return tree.Roots[i].Number < tree.Roots[j].Number
	})
	for _, root := range tree.Roots {
		sortChildren(root)
	}
}

func sortChildren(node *domain.IssueNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Number < node.Children[j].Number
	})
	for _, child := range node.Children {
		sortChildren(child)
	}
}

var _ Client = (*GitHubClient)(nil)
