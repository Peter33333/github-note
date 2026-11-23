package domain

import "testing"

func TestBuildRootsResolvesParentAcrossBatches(t *testing.T) {
	tree := NewIssueTree()
	// Simulate page 2 arriving before its parent from page 1.
	tree.AddNode(&IssueNode{ID: "child", Number: 120, ParentID: "parent"})
	tree.AddNode(&IssueNode{ID: "parent", Number: 10})
	tree.AddNode(&IssueNode{ID: "root", Number: 5})

	tree.BuildRoots()

	if len(tree.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree.Roots))
	}
	parent := tree.Nodes["parent"]
	if len(parent.Children) != 1 {
		t.Fatalf("expected parent to have 1 child, got %d", len(parent.Children))
	}
	if parent.Children[0].ID != "child" {
		t.Fatalf("unexpected child: %s", parent.Children[0].ID)
	}
}

func TestBuildRootsPromotesOrphanToRoot(t *testing.T) {
	tree := NewIssueTree()
	tree.AddNode(&IssueNode{ID: "orphan", Number: 3, ParentID: "missing"})

	tree.BuildRoots()

	if len(tree.Roots) != 1 || tree.Roots[0].ID != "orphan" {
		t.Fatalf("orphan should be promoted to root, roots=%+v", tree.Roots)
	}
}
