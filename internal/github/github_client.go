package github

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github-note/internal/config"
	"github-note/internal/domain"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"golang.org/x/term"
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

type issuePageQuery struct {
	Repository struct {
		Issues struct {
			TotalCount githubv4.Int
			Nodes      []issueNodeQuery
			PageInfo   struct {
				HasNextPage githubv4.Boolean
				EndCursor   githubv4.String
			}
		} `graphql:"issues(first: $first, after: $after, orderBy: {field: CREATED_AT, direction: ASC})"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

type restIssue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	HTMLURL     string `json:"html_url"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request,omitempty"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// GitHubClient implements Client.
type GitHubClient struct {
	token *oauth2.Token
}

func New(cfg *config.Config) *GitHubClient {
	_ = cfg
	return &GitHubClient{}
}

func (client *GitHubClient) EnsureToken(ctx context.Context) error {
	token, err := config.LoadToken()
	if err == nil {
		if err := client.validateToken(ctx, token.AccessToken); err == nil {
			client.token = token
			return nil
		}
	}

	if envToken := strings.TrimSpace(os.Getenv("GH_TOKEN")); envToken != "" {
		if err := client.validateToken(ctx, envToken); err != nil {
			return fmt.Errorf("invalid GH_TOKEN: %w", err)
		}
		oauthToken := &oauth2.Token{AccessToken: envToken, TokenType: "Bearer"}
		if err := config.SaveToken(oauthToken); err != nil {
			return err
		}
		client.token = oauthToken
		return nil
	}

	inputToken, err := promptPersonalAccessToken()
	if err != nil {
		return err
	}
	if inputToken != "" {
		if err := client.validateToken(ctx, inputToken); err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}
		oauthToken := &oauth2.Token{AccessToken: inputToken, TokenType: "Bearer"}
		if err := config.SaveToken(oauthToken); err != nil {
			return err
		}
		client.token = oauthToken
		return nil
	}

	client.token = nil
	return nil
}

func promptPersonalAccessToken() (string, error) {
	fmt.Println("Paste GitHub Personal Access Token (recommended scope: repo).")
	fmt.Println("Generate one at: https://github.com/settings/tokens (classic token with `repo` scope).")
	fmt.Print("Personal Access Token (optional, press Enter for public repositories only): ")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		secret, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read token input: %w", err)
		}
		return strings.TrimSpace(string(secret)), nil
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, os.ErrClosed) {
			return "", nil
		}
		return "", fmt.Errorf("read token input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func (client *GitHubClient) validateToken(ctx context.Context, accessToken string) error {
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("empty token")
	}
	oauthToken := &oauth2.Token{AccessToken: accessToken, TokenType: "Bearer"}
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(oauthToken))
	graphqlClient := githubv4.NewClient(oauthClient)

	query := struct {
		Viewer struct {
			Login githubv4.String
		}
	}{}
	if err := graphqlClient.Query(ctx, &query, nil); err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}
	if strings.TrimSpace(string(query.Viewer.Login)) == "" {
		return errors.New("token verification failed: empty viewer login")
	}
	return nil
}

const (
	defaultIssueBatchSize = 100
	maxIssuePages         = 100
)

// LoadIssueTree fetches every issue page and returns one merged tree.
// The second return value is the total issue count reported by GitHub.
func (client *GitHubClient) LoadIssueTree(ctx context.Context, owner string, repo string) (*domain.IssueTree, int, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, 0, errors.New("owner and repo are required")
	}
	if client.token == nil {
		return client.loadIssueTreePublic(ctx, owner, repo)
	}
	return client.loadIssueTreeWithToken(ctx, owner, repo)
}

func (client *GitHubClient) loadIssueTreeWithToken(ctx context.Context, owner string, repo string) (*domain.IssueTree, int, error) {
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(client.token))
	graphqlClient := githubv4.NewClient(oauthClient)

	tree := domain.NewIssueTree()
	totalCount := 0
	var after *githubv4.String

	// Iterate every page so parent nodes can be resolved across page boundaries.
	for page := 1; page <= maxIssuePages; page++ {
		query := issuePageQuery{}
		variables := map[string]interface{}{
			"owner": githubv4.String(owner),
			"repo":  githubv4.String(repo),
			"first": githubv4.Int(defaultIssueBatchSize),
			"after": after,
		}

		if err := queryWithRetry(ctx, graphqlClient, &query, variables); err != nil {
			return nil, 0, fmt.Errorf("query issues page %d: %w", page, err)
		}

		totalCount = int(query.Repository.Issues.TotalCount)
		for _, n := range query.Repository.Issues.Nodes {
			tree.AddNode(issueNodeFromQuery(n))
		}

		hasNext := bool(query.Repository.Issues.PageInfo.HasNextPage)
		cursor := string(query.Repository.Issues.PageInfo.EndCursor)
		if !hasNext || strings.TrimSpace(cursor) == "" {
			break
		}
		if totalCount > 0 && len(tree.Nodes) >= totalCount {
			break
		}

		cursorValue := githubv4.String(cursor)
		after = &cursorValue
	}

	tree.BuildRoots()
	sortTree(tree)
	return tree, totalCount, nil
}

func queryWithRetry(ctx context.Context, graphqlClient *githubv4.Client, query interface{}, variables map[string]interface{}) error {
	var queryErr error
	for attempt := 1; attempt <= githubRetryMaxAttempts; attempt++ {
		queryErr = graphqlClient.Query(ctx, query, variables)
		if queryErr == nil {
			return nil
		}
		if attempt == githubRetryMaxAttempts || !isRetryableGitHubError(queryErr) {
			return queryErr
		}
		if err := waitRetryBackoff(ctx, attempt); err != nil {
			return queryErr
		}
	}
	return queryErr
}

func issueNodeFromQuery(n issueNodeQuery) *domain.IssueNode {
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
	return node
}

func (client *GitHubClient) loadIssueTreePublic(ctx context.Context, owner string, repo string) (*domain.IssueTree, int, error) {
	tree := domain.NewIssueTree()
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=all&per_page=%d&page=1", owner, repo, defaultIssueBatchSize)

	// Follow the Link header until the last page so no issue is skipped.
	for page := 1; page <= maxIssuePages; page++ {
		issues, next, err := client.fetchPublicIssuesPage(ctx, owner, repo, endpoint)
		if err != nil {
			return nil, 0, err
		}
		for _, issue := range issues {
			if issue.PullRequest != nil {
				continue
			}
			labels := make([]string, 0, len(issue.Labels))
			for _, label := range issue.Labels {
				labels = append(labels, label.Name)
			}
			tree.AddNode(&domain.IssueNode{
				ID:     fmt.Sprintf("public-%d", issue.Number),
				Number: issue.Number,
				Title:  issue.Title,
				Labels: labels,
				URL:    issue.HTMLURL,
				State:  issue.State,
			})
		}
		if strings.TrimSpace(next) == "" {
			break
		}
		endpoint = next
	}

	tree.BuildRoots()
	sortTree(tree)
	return tree, len(tree.Nodes), nil
}

func (client *GitHubClient) fetchPublicIssuesPage(ctx context.Context, owner string, repo string, endpoint string) ([]restIssue, string, error) {
	httpClient := &http.Client{}

	var (
		resp *http.Response
		err  error
	)

	for attempt := 1; attempt <= githubRetryMaxAttempts; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if reqErr != nil {
			return nil, "", fmt.Errorf("create public issues request: %w", reqErr)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "ghnote")

		resp, err = httpClient.Do(req)
		if err != nil {
			if attempt == githubRetryMaxAttempts || !isRetryableGitHubError(err) {
				return nil, "", fmt.Errorf("request public issues: %w", err)
			}
			if waitErr := waitRetryBackoff(ctx, attempt); waitErr != nil {
				return nil, "", fmt.Errorf("request public issues: %w", err)
			}
			continue
		}

		if isRetryableGitHubStatus(resp.StatusCode) {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			err = fmt.Errorf("transient status code: %d body: %q", resp.StatusCode, strings.TrimSpace(string(body)))
			if attempt == githubRetryMaxAttempts {
				return nil, "", fmt.Errorf("request public issues: %w", err)
			}
			if waitErr := waitRetryBackoff(ctx, attempt); waitErr != nil {
				return nil, "", fmt.Errorf("request public issues: %w", err)
			}
			continue
		}

		break
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", fmt.Errorf("repository not found or private: %s/%s (provide PAT to access private repositories)", owner, repo)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, "", fmt.Errorf("public api access denied for %s/%s (provide PAT for private repositories or higher rate limits)", owner, repo)
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", fmt.Errorf("request public issues failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	issues := make([]restIssue, 0)
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		resp.Body.Close()
		return nil, "", fmt.Errorf("decode public issues response: %w", err)
	}
	next := extractNextLink(resp.Header.Get("Link"))
	resp.Body.Close()
	return issues, next, nil
}

func extractNextLink(linkHeader string) string {
	if strings.TrimSpace(linkHeader) == "" {
		return ""
	}
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if !strings.Contains(segment, `rel="next"`) {
			continue
		}
		start := strings.Index(segment, "<")
		end := strings.Index(segment, ">")
		if start == -1 || end == -1 || end <= start+1 {
			continue
		}
		return strings.TrimSpace(segment[start+1 : end])
	}
	return ""
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

const (
	githubRetryMaxAttempts = 4
	githubRetryBaseDelay   = 400 * time.Millisecond
	githubRetryMaxDelay    = 3 * time.Second
)

func waitRetryBackoff(ctx context.Context, attempt int) error {
	delay := githubRetryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= githubRetryMaxDelay {
			delay = githubRetryMaxDelay
			break
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableGitHubStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func isRetryableGitHubError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	msg := strings.ToLower(err.Error())
	retryHints := []string{
		"status code: 429",
		"status code: 500",
		"status code: 502",
		"status code: 503",
		"status code: 504",
		"bad gateway",
		"service unavailable",
		"temporarily unavailable",
		"connection reset",
		"tls handshake timeout",
		"timeout",
		"eof",
	}
	for _, hint := range retryHints {
		if strings.Contains(msg, hint) {
			return true
		}
	}

	return false
}

var _ Client = (*GitHubClient)(nil)
