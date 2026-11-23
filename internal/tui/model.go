package tui

import (
	"fmt"
	"math"
	"strings"

	"github-note/internal/domain"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const headerASCIIArt = `______ _ _   _____     _      _____     _       
|   __|_| |_|  |  |_ _| |_   |   | |___| |_ ___ 
|  |  | |  _|     | | | . |  | | | | . |  _| -_|
|_____|_|_| |__|__|___|___|  |_|___|___|_| |___|`

type flatNode struct {
	Node        *domain.IssueNode
	Depth       int
	HasKid      bool
	IsCollapsed bool
}

type Model struct {
	tree       *domain.IssueTree
	flat       []flatNode
	cursor     int
	offset     int
	totalCount int
	width      int
	height     int
	quitting   bool
	err        error
	status     string
	showHelp   bool
	collapsed  map[string]bool
	openIssue  func(url string) error
	loadTree   func() (*domain.IssueTree, int, error)

	root        lipgloss.Style
	headerBar   lipgloss.Style
	headerTitle lipgloss.Style
	headerMeta  lipgloss.Style

	pane         lipgloss.Style
	paneTitle    lipgloss.Style
	paneBody     lipgloss.Style
	paneBodyMute lipgloss.Style

	treeParent lipgloss.Style
	treeLeaf   lipgloss.Style
	treeFocus  lipgloss.Style

	detailKey   lipgloss.Style
	detailValue lipgloss.Style
	detailHint  lipgloss.Style

	footerBar  lipgloss.Style
	footerInfo lipgloss.Style
	footerHint lipgloss.Style
	footerErr  lipgloss.Style
}

func New(tree *domain.IssueTree, openIssue func(url string) error, totalCount int, loadTree func() (*domain.IssueTree, int, error)) *Model {
	model := &Model{
		tree:       tree,
		openIssue:  openIssue,
		collapsed:  make(map[string]bool),
		totalCount: totalCount,
		loadTree:   loadTree,
		status:     loadStatus(tree, totalCount),
		root: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4A5158")).
			Background(lipgloss.Color("#111417")).
			Padding(0, 0),
		headerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E7E2D8")).
			Background(lipgloss.Color("#1A1E23")).
			Padding(0, 1),
		headerTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F1ECE3")).
			Bold(true),
		headerMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#98A0A8")),
		pane: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#353C44")).
			Background(lipgloss.Color("#15191D")),
		paneTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C6CED6")).
			Background(lipgloss.Color("#1D2228")).
			Bold(true).
			Padding(0, 1),
		paneBody: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D8D3CB")).
			Background(lipgloss.Color("#15191D")),
		paneBodyMute: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#838A91")).
			Background(lipgloss.Color("#15191D")),
		treeParent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B9C2CB")).
			Background(lipgloss.Color("#15191D")),
		treeLeaf: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C6CED6")).
			Background(lipgloss.Color("#15191D")),
		treeFocus: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#14181C")).
			Background(lipgloss.Color("#B7C0C8")).
			Bold(true),
		detailKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9BA5AF")).
			Bold(true),
		detailValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDD7CE")),
		detailHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#858C93")),
		footerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E7E2D8")).
			Background(lipgloss.Color("#1A1E23")).
			Padding(0, 1),
		footerInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BCC5CD")),
		footerHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#848B92")),
		footerErr: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D39A8E")).
			Bold(true),
	}

	initCollapsedState(tree, model.collapsed)
	model.rebuildFlat()
	return model
}

func (model *Model) Init() tea.Cmd {
	return nil
}

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		model.width = m.Width
		model.height = m.Height
		model.ensureCursorVisible()
	case tea.KeyMsg:
		switch m.String() {
		case "q", "ctrl+c":
			model.quitting = true
			return model, tea.Quit
		case "up", "k":
			if model.cursor > 0 {
				model.cursor--
			}
			model.ensureCursorVisible()
			model.status = model.selectionStatus()
		case "down", "j":
			if model.cursor < len(model.flat)-1 {
				model.cursor++
			}
			model.ensureCursorVisible()
			model.status = model.selectionStatus()
		case "g", "home":
			if len(model.flat) > 0 {
				model.cursor = 0
				model.ensureCursorVisible()
				model.status = "Moved to first issue"
			}
		case "G", "end":
			if len(model.flat) > 0 {
				model.cursor = len(model.flat) - 1
				model.ensureCursorVisible()
				model.status = "Moved to last issue"
			}
		case "pgup", "ctrl+u":
			step := model.visibleTreeRows() - 1
			if step < 1 {
				step = 1
			}
			model.cursor = clamp(model.cursor-step, 0, max(0, len(model.flat)-1))
			model.ensureCursorVisible()
			model.status = model.selectionStatus()
		case "pgdown", "ctrl+d":
			step := model.visibleTreeRows() - 1
			if step < 1 {
				step = 1
			}
			model.cursor = clamp(model.cursor+step, 0, max(0, len(model.flat)-1))
			model.ensureCursorVisible()
			model.status = model.selectionStatus()
		case "left", "h":
			model.collapseCurrent()
		case "right", "l":
			model.expandCurrent()
		case " ":
			model.toggleCurrent()
		case "?":
			model.showHelp = !model.showHelp
			if model.showHelp {
				model.status = "Help expanded"
			} else {
				model.status = "Help collapsed"
			}
		case "r":
			model.reloadTree()
		case "enter":
			if len(model.flat) == 0 || model.openIssue == nil {
				model.status = "No issue to open"
				return model, nil
			}
			selected := model.flat[model.cursor].Node
			if err := model.openIssue(selected.URL); err != nil {
				model.err = err
				model.status = "Open issue failed"
			} else {
				model.err = nil
				model.status = fmt.Sprintf("Opened #%d", selected.Number)
			}
		}
	}
	return model, nil
}

func (model *Model) View() string {
	if model.quitting {
		return "bye\n"
	}

	innerWidth := model.innerWidth()
	innerHeight := model.innerHeight()
	if innerWidth <= 0 {
		innerWidth = 1
	}
	if innerHeight <= 0 {
		innerHeight = 1
	}

	footerLines := model.renderFooterLines(innerWidth)
	header := model.renderHeader(innerWidth)
	headerHeight := strings.Count(header, "\n") + 1
	bodyHeight := max(5, innerHeight-headerHeight-len(footerLines))
	body := model.renderBody(innerWidth, bodyHeight)
	footer := strings.Join(footerLines, "\n")
	content := strings.Join([]string{header, body, footer}, "\n")

	return model.root.Width(innerWidth).Height(innerHeight).Render(content)
}

func (model *Model) renderHeader(width int) string {
	innerWidth := max(1, width-model.headerBar.GetHorizontalFrameSize())
	titleLines := strings.Split(headerASCIIArt, "\n")
	rows := make([]string, 0, len(titleLines)+1)
	for _, line := range titleLines {
		styled := model.headerTitle.Render(truncateToWidth(line, innerWidth))
		rows = append(rows, model.headerBar.Width(innerWidth).MaxWidth(innerWidth).Render(styled))
	}
	meta := model.headerMeta.Render(fmt.Sprintf("issues:%d  total:%d  selected:%d/%d", len(model.flat), model.totalCount, model.cursor+1, max(1, len(model.flat))))
	rows = append(rows, model.headerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(meta, innerWidth)))
	return strings.Join(rows, "\n")
}

func (model *Model) renderBody(width int, height int) string {
	if width < 70 {
		treeHeight := max(3, int(float64(height)*0.65))
		detailHeight := max(2, height-treeHeight)
		tree := model.renderTreePane(width, treeHeight)
		detail := model.renderDetailPane(width, detailHeight)
		return lipgloss.JoinVertical(lipgloss.Left, tree, detail)
	}

	gap := 1
	available := max(1, width-gap)
	leftWidth := int(float64(available) * 0.64)
	leftWidth = clamp(leftWidth, 20, available-16)
	rightWidth := available - leftWidth
	if rightWidth < 16 {
		rightWidth = 16
		leftWidth = available - rightWidth
	}

	left := model.renderTreePane(leftWidth, height)
	right := model.renderDetailPane(rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (model *Model) renderTreePane(width int, height int) string {
	paneInnerWidth := max(1, width-model.pane.GetHorizontalFrameSize())
	paneInnerHeight := max(1, height-model.pane.GetVerticalFrameSize())
	title := model.paneTitle.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("ISSUES")
	rowsHeight := max(1, paneInnerHeight-1)
	model.ensureCursorVisibleWithRows(rowsHeight)

	rows := make([]string, 0, rowsHeight)
	if len(model.flat) == 0 {
		empty := model.paneBodyMute.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("No issues found.")
		rows = append(rows, empty)
	}

	if len(model.flat) > 0 {
		start := clamp(model.offset, 0, max(0, len(model.flat)-1))
		end := min(len(model.flat), start+rowsHeight)
		for i := start; i < end; i++ {
			item := model.flat[i]
			line := truncateToWidth(model.formatIssueLine(item), paneInnerWidth)
			if i == model.cursor {
				rows = append(rows, model.treeFocus.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(line))
				continue
			}
			if item.HasKid {
				rows = append(rows, model.treeParent.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(line))
			} else {
				rows = append(rows, model.treeLeaf.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(line))
			}
		}
	}

	for len(rows) < rowsHeight {
		rows = append(rows, model.paneBody.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(""))
	}

	body := strings.Join(rows, "\n")
	return model.pane.Width(paneInnerWidth).Height(paneInnerHeight).Render(strings.Join([]string{title, body}, "\n"))
}

func (model *Model) renderDetailPane(width int, height int) string {
	paneInnerWidth := max(1, width-model.pane.GetHorizontalFrameSize())
	paneInnerHeight := max(1, height-model.pane.GetVerticalFrameSize())
	title := model.paneTitle.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("DETAIL")
	rowsHeight := max(1, paneInnerHeight-1)
	rows := make([]string, 0, rowsHeight)

	if len(model.flat) == 0 {
		rows = append(rows, model.detailHint.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("Select an issue to see details."))
	} else {
		selected := model.flat[model.cursor].Node
		labels := formatLabels(selected.Labels)
		if labels == "" {
			labels = "-"
		}
		rows = append(rows, model.detailRow(paneInnerWidth, "Number", fmt.Sprintf("#%d", selected.Number)))
		rows = append(rows, model.detailRow(paneInnerWidth, "State", selected.State))
		rows = append(rows, model.detailRow(paneInnerWidth, "Children", fmt.Sprintf("%d", len(selected.Children))))
		if strings.TrimSpace(selected.ParentID) == "" {
			rows = append(rows, model.detailRow(paneInnerWidth, "Parent", "ROOT"))
		} else {
			rows = append(rows, model.detailRow(paneInnerWidth, "Parent", "HAS_PARENT"))
		}
		rows = append(rows, model.detailHint.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("Title"))
		rows = append(rows, model.detailValue.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(truncateToWidth(selected.Title, paneInnerWidth)))
		rows = append(rows, model.detailHint.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("Labels"))
		rows = append(rows, model.detailValue.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(truncateToWidth(labels, paneInnerWidth)))
		rows = append(rows, model.detailHint.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render("URL"))
		rows = append(rows, model.detailValue.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(truncateToWidth(selected.URL, paneInnerWidth)))
	}

	for len(rows) < rowsHeight {
		rows = append(rows, model.paneBody.Width(paneInnerWidth).MaxWidth(paneInnerWidth).Render(""))
	}

	body := strings.Join(rows[:rowsHeight], "\n")
	return model.pane.Width(paneInnerWidth).Height(paneInnerHeight).Render(strings.Join([]string{title, body}, "\n"))
}

func (model *Model) detailRow(width int, key string, value string) string {
	line := fmt.Sprintf("%s: %s", model.detailKey.Render(key), model.detailValue.Render(value))
	return truncateToWidth(line, width)
}

func (model *Model) renderFooterLines(width int) []string {
	innerWidth := max(1, width-model.footerBar.GetHorizontalFrameSize())
	statusLine := model.footerInfo.Render(model.selectionStatus())
	if model.status != "" {
		statusLine = model.footerInfo.Render(model.status)
	}
	if model.err != nil {
		statusLine = model.footerErr.Render("Error: " + model.err.Error())
	}

	hintLine := model.footerHint.Render("j/k:move  h/l:fold  space:toggle  r:reload all  enter:open  g/G:top/bottom  pgup/pgdown:scroll  ?:help  q:quit")
	lines := []string{
		model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(statusLine, innerWidth)),
		model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(hintLine, innerWidth)),
	}
	if model.showHelp {
		extra := model.footerHint.Render("Tip: parent nodes show ▸[n]/▾[n]. Press r to reload all issues from GitHub. Right panel shows issue metadata.")
		lines = append(lines, model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(extra, innerWidth)))
	}
	return lines
}

func (model *Model) selectionStatus() string {
	if len(model.flat) == 0 {
		return fmt.Sprintf("No issues loaded (total %d)", model.totalCount)
	}
	node := model.flat[model.cursor].Node
	return fmt.Sprintf("Selected #%d (%d/%d)  total %d", node.Number, model.cursor+1, len(model.flat), model.totalCount)
}

func (model *Model) reloadTree() {
	if model.loadTree == nil {
		model.status = "Reload is not available"
		return
	}

	selectedID := ""
	if model.cursor >= 0 && model.cursor < len(model.flat) {
		selectedID = model.flat[model.cursor].Node.ID
	}
	previousTree := model.tree
	previousCollapsed := cloneCollapsedMap(model.collapsed)

	tree, totalCount, err := model.loadTree()
	if err != nil {
		model.err = err
		model.status = "Reload failed"
		return
	}

	model.tree = tree
	model.totalCount = totalCount
	model.offset = 0
	model.collapsed = make(map[string]bool)
	initCollapsedState(model.tree, model.collapsed)
	restoreCollapsedState(previousTree, previousCollapsed, model.tree, model.collapsed)
	model.rebuildFlat()
	if selectedID != "" {
		model.restoreSelectionByID(selectedID)
	}

	model.err = nil
	model.status = loadStatus(model.tree, model.totalCount)
	if loaded := loadedCount(model.tree); totalCount > 0 && loaded != totalCount {
		model.err = fmt.Errorf("loaded %d issues but GitHub reports %d", loaded, totalCount)
	}
}

func loadedCount(tree *domain.IssueTree) int {
	if tree == nil {
		return 0
	}
	return len(tree.Nodes)
}

func loadStatus(tree *domain.IssueTree, totalCount int) string {
	return fmt.Sprintf("Loaded %d/%d issues", loadedCount(tree), totalCount)
}

func (model *Model) collapseCurrent() {
	if len(model.flat) == 0 {
		return
	}
	node := model.flat[model.cursor]
	if !node.HasKid {
		return
	}
	model.collapsed[node.Node.ID] = true
	model.rebuildFlat()
	model.status = fmt.Sprintf("Collapsed #%d", node.Node.Number)
}

func (model *Model) expandCurrent() {
	if len(model.flat) == 0 {
		return
	}
	node := model.flat[model.cursor]
	if !node.HasKid {
		return
	}
	delete(model.collapsed, node.Node.ID)
	model.rebuildFlat()
	model.status = fmt.Sprintf("Expanded #%d", node.Node.Number)
}

func (model *Model) toggleCurrent() {
	if len(model.flat) == 0 {
		return
	}
	node := model.flat[model.cursor]
	if !node.HasKid {
		return
	}
	if model.collapsed[node.Node.ID] {
		delete(model.collapsed, node.Node.ID)
		model.status = fmt.Sprintf("Expanded #%d", node.Node.Number)
	} else {
		model.collapsed[node.Node.ID] = true
		model.status = fmt.Sprintf("Collapsed #%d", node.Node.Number)
	}
	model.rebuildFlat()
}

func (model *Model) rebuildFlat() {
	selectedID := ""
	if model.cursor >= 0 && model.cursor < len(model.flat) {
		selectedID = model.flat[model.cursor].Node.ID
	}
	model.flat = flattenTree(model.tree, model.collapsed)
	if len(model.flat) == 0 {
		model.cursor = 0
		model.offset = 0
		return
	}
	if selectedID == "" {
		model.cursor = clamp(model.cursor, 0, len(model.flat)-1)
		model.ensureCursorVisible()
		return
	}
	for index, item := range model.flat {
		if item.Node.ID == selectedID {
			model.cursor = index
			model.ensureCursorVisible()
			return
		}
	}
	model.cursor = clamp(model.cursor, 0, len(model.flat)-1)
	model.ensureCursorVisible()
}

func (model *Model) restoreSelectionByID(selectedID string) {
	for index, item := range model.flat {
		if item.Node.ID == selectedID {
			model.cursor = index
			model.ensureCursorVisible()
			return
		}
	}
	model.cursor = clamp(model.cursor, 0, max(0, len(model.flat)-1))
	model.ensureCursorVisible()
}

func cloneCollapsedMap(source map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

// initCollapsedState folds every node that has children, so the tree starts collapsed.
func initCollapsedState(tree *domain.IssueTree, collapsed map[string]bool) {
	if tree == nil {
		return
	}
	for _, node := range tree.Nodes {
		if len(node.Children) > 0 {
			collapsed[node.ID] = true
		}
	}
}

// restoreCollapsedState keeps the fold choice of the user for issues that already existed
// before a reload. Nodes that appear for the first time stay collapsed.
func restoreCollapsedState(previousTree *domain.IssueTree, previous map[string]bool, current *domain.IssueTree, target map[string]bool) {
	if previousTree == nil || current == nil {
		return
	}
	for id := range current.Nodes {
		if _, existed := previousTree.Nodes[id]; !existed {
			continue
		}
		target[id] = previous[id]
	}
}

func flattenTree(tree *domain.IssueTree, collapsed map[string]bool) []flatNode {
	if tree == nil {
		return nil
	}
	result := make([]flatNode, 0)
	for _, root := range tree.Roots {
		walkNode(root, 0, collapsed, &result)
	}
	return result
}

func walkNode(node *domain.IssueNode, depth int, collapsed map[string]bool, out *[]flatNode) {
	isCollapsed := collapsed[node.ID]
	*out = append(*out, flatNode{Node: node, Depth: depth, HasKid: len(node.Children) > 0, IsCollapsed: isCollapsed})
	if isCollapsed {
		return
	}
	for _, child := range node.Children {
		walkNode(child, depth+1, collapsed, out)
	}
}

func (model *Model) formatIssueLine(item flatNode) string {
	marker := "•"
	if item.HasKid {
		childCount := fmt.Sprintf("[%d]", len(item.Node.Children))
		if item.IsCollapsed {
			marker = "▸" + childCount
		} else {
			marker = "▾" + childCount
		}
	}
	indent := strings.Repeat("  ", item.Depth)
	labels := formatLabels(item.Node.Labels)
	if labels == "" {
		return fmt.Sprintf("%s%s #%d %s", indent, marker, item.Node.Number, item.Node.Title)
	}
	return fmt.Sprintf("%s%s #%d %s %s", indent, marker, item.Node.Number, item.Node.Title, labels)
}

func formatLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("[%s]", label))
	}
	return strings.Join(parts, " ")
}

func (model *Model) innerWidth() int {
	if model.width <= 0 {
		return 0
	}
	return max(1, model.width-model.root.GetHorizontalFrameSize())
}

func (model *Model) innerHeight() int {
	if model.height <= 0 {
		return 0
	}
	return max(1, model.height-model.root.GetVerticalFrameSize())
}

func (model *Model) visibleTreeRows() int {
	innerHeight := model.innerHeight()
	footerCount := len(model.renderFooterLines(max(1, model.innerWidth())))
	headerHeight := strings.Count(model.renderHeader(max(1, model.innerWidth())), "\n") + 1
	bodyHeight := max(5, innerHeight-headerHeight-footerCount)
	return max(1, bodyHeight-1)
}

func (model *Model) ensureCursorVisible() {
	model.ensureCursorVisibleWithRows(model.visibleTreeRows())
}

func (model *Model) ensureCursorVisibleWithRows(rows int) {
	if len(model.flat) == 0 {
		model.cursor = 0
		model.offset = 0
		return
	}
	model.cursor = clamp(model.cursor, 0, len(model.flat)-1)
	if model.cursor < model.offset {
		model.offset = model.cursor
	}
	if model.cursor >= model.offset+rows {
		model.offset = model.cursor - rows + 1
	}
	maxOffset := max(0, len(model.flat)-rows)
	model.offset = clamp(model.offset, 0, maxOffset)
}

func truncateToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(value, width, "…")
}

func min(a int, b int) int {
	return int(math.Min(float64(a), float64(b)))
}

func max(a int, b int) int {
	return int(math.Max(float64(a), float64(b)))
}

func clamp(value int, low int, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
