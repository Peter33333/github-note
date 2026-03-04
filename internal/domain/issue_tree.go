package domain

// IssueNode represents one issue in a tree relationship.
type IssueNode struct {
	ID       string
	Number   int
	Title    string
	Labels   []string
	URL      string
	State    string
	ParentID string
	Children []*IssueNode
}

// IssueTree contains issue nodes and root references.
type IssueTree struct {
	Nodes map[string]*IssueNode
	Roots []*IssueNode
}

func NewIssueTree() *IssueTree {
	return &IssueTree{Nodes: make(map[string]*IssueNode)}
}

func (tree *IssueTree) AddNode(node *IssueNode) {
	if node == nil {
		return
	}
	tree.Nodes[node.ID] = node
}

func (tree *IssueTree) BuildRoots() {
	roots := make([]*IssueNode, 0)
	for _, node := range tree.Nodes {
		if node.ParentID == "" {
			roots = append(roots, node)
			continue
		}
		parent, ok := tree.Nodes[node.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	tree.Roots = roots
}
