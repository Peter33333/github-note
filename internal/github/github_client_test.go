package github

import (
	"net/url"
	"testing"

	"github-note/internal/domain"

	"github.com/shurcooL/githubv4"
)

func TestSortTreeOrdersRootsAndChildrenByNumber(t *testing.T) {
	tree := domain.NewIssueTree()
	tree.AddNode(&domain.IssueNode{ID: "a2", Number: 2})
	tree.AddNode(&domain.IssueNode{ID: "a1", Number: 1})
	tree.AddNode(&domain.IssueNode{ID: "c22", Number: 22, ParentID: "a2"})
	tree.AddNode(&domain.IssueNode{ID: "c21", Number: 21, ParentID: "a2"})
	tree.BuildRoots()
	sortTree(tree)

	if len(tree.Roots) != 2 || tree.Roots[0].Number != 1 || tree.Roots[1].Number != 2 {
		t.Fatalf("unexpected root order: %+v", tree.Roots)
	}
	children := tree.Nodes["a2"].Children
	if len(children) != 2 || children[0].Number != 21 || children[1].Number != 22 {
		t.Fatalf("unexpected child order: %+v", children)
	}
}

func TestExtractNextLink(t *testing.T) {
	header := `<https://api.github.com/repositories/1/issues?page=2>; rel="next", ` +
		`<https://api.github.com/repositories/1/issues?page=5>; rel="last"`
	want := "https://api.github.com/repositories/1/issues?page=2"
	if got := extractNextLink(header); got != want {
		t.Fatalf("unexpected next link: got %q want %q", got, want)
	}
	if got := extractNextLink(""); got != "" {
		t.Fatalf("expected empty link, got %q", got)
	}
}

func TestIssueNodeFromQueryMapsFields(t *testing.T) {
	issueURL, err := url.Parse("https://github.com/owner/repo/issues/7")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	query := issueNodeQuery{
		ID:     githubv4.ID("I_7"),
		Number: githubv4.Int(7),
		Title:  githubv4.String("child issue"),
		URL:    githubv4.URI{URL: issueURL},
		State:  githubv4.String("OPEN"),
	}
	query.Labels.Nodes = []struct {
		Name githubv4.String
	}{{Name: githubv4.String("note")}}
	query.Parent = &struct{ ID githubv4.ID }{ID: githubv4.ID("I_1")}

	node := issueNodeFromQuery(query)

	if node.ID != "I_7" || node.Number != 7 || node.Title != "child issue" {
		t.Fatalf("unexpected node: %+v", node)
	}
	if node.URL != issueURL.String() || node.State != "OPEN" {
		t.Fatalf("unexpected node url/state: %+v", node)
	}
	if len(node.Labels) != 1 || node.Labels[0] != "note" {
		t.Fatalf("unexpected labels: %+v", node.Labels)
	}
	if node.ParentID != "I_1" {
		t.Fatalf("unexpected parent id: %q", node.ParentID)
	}
}

func TestIssueNodeFromQueryWithoutParent(t *testing.T) {
	issueURL, err := url.Parse("https://github.com/owner/repo/issues/1")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	node := issueNodeFromQuery(issueNodeQuery{
		ID:     githubv4.ID("I_1"),
		Number: githubv4.Int(1),
		URL:    githubv4.URI{URL: issueURL},
	})

	if node.ParentID != "" {
		t.Fatalf("expected root node, got parent %q", node.ParentID)
	}
}
