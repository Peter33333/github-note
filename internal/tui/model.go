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

type flatNode struct {
	Node        *domain.IssueNode
	Depth       int
	HasKid      bool
	IsCollapsed bool
}

type Model struct {
	tree      *domain.IssueTree
	flat      []flatNode
	cursor    int
	offset    int
	width     int
	height    int
	quitting  bool
	err       error
	status    string
	showHelp  bool
	collapsed map[string]bool
	openIssue func(url string) error

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

func New(tree *domain.IssueTree, openIssue func(url string) error) *Model {
	model := &Model{
		tree:      tree,
		openIssue: openIssue,
		collapsed: make(map[string]bool),
		status:    "Ready",
		root: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3D4A66")).
			Background(lipgloss.Color("#0B1020")).
			Padding(0, 0),
		headerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDE4FF")).
			Background(lipgloss.Color("#121A2B")).
			Padding(0, 1),
		headerTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E8EEFF")).
			Bold(true),
		headerMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A3C7")),
		pane: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#2E3A57")).
			Background(lipgloss.Color("#0D1426")),
		paneTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9FB4FF")).
			Background(lipgloss.Color("#131E36")).
			Bold(true).
			Padding(0, 1),
		paneBody: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D8E1FF")).
			Background(lipgloss.Color("#0D1426")),
		paneBodyMute: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D8BAD")).
			Background(lipgloss.Color("#0D1426")),
		treeParent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B5C7FF")).
			Background(lipgloss.Color("#0D1426")),
		treeLeaf: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CFD8F6")).
			Background(lipgloss.Color("#0D1426")),
		treeFocus: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0B1020")).
			Background(lipgloss.Color("#8FB4FF")).
			Bold(true),
		detailKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8EA3DB")).
			Bold(true),
		detailValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D9E2FF")),
		detailHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D8BAD")),
		footerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDE4FF")).
			Background(lipgloss.Color("#121A2B")).
			Padding(0, 1),
		footerInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9FB4FF")),
		footerHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D8BAD")),
		footerErr: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8AA4")).
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
	bodyHeight := max(5, innerHeight-1-len(footerLines))

	header := model.renderHeader(innerWidth)
	body := model.renderBody(innerWidth, bodyHeight)
	footer := strings.Join(footerLines, "\n")
	content := strings.Join([]string{header, body, footer}, "\n")

	return model.root.Width(innerWidth).Height(innerHeight).Render(content)
}

func (model *Model) renderHeader(width int) string {
	innerWidth := max(1, width-model.headerBar.GetHorizontalFrameSize())
	title := model.headerTitle.Render("GHNOTE")
	meta := model.headerMeta.Render(fmt.Sprintf("items:%d  selected:%d/%d", len(model.flat), model.cursor+1, max(1, len(model.flat))))
	line := truncateToWidth(fmt.Sprintf("%s  %s", title, meta), innerWidth)
	return model.headerBar.Width(innerWidth).MaxWidth(innerWidth).Render(line)
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

	hintLine := model.footerHint.Render("j/k:move  h/l:fold  space:toggle  enter:open  g/G:top/bottom  pgup/pgdown:page  ?:help  q:quit")
	lines := []string{
		model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(statusLine, innerWidth)),
		model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(hintLine, innerWidth)),
	}
	if model.showHelp {
		extra := model.footerHint.Render("Tip: parent nodes show ▸/▾. Focus line is highlighted. Right panel shows issue metadata.")
		lines = append(lines, model.footerBar.Width(innerWidth).MaxWidth(innerWidth).Render(truncateToWidth(extra, innerWidth)))
	}
	return lines
}

func (model *Model) selectionStatus() string {
	if len(model.flat) == 0 {
		return "No issues"
	}
	node := model.flat[model.cursor].Node
	return fmt.Sprintf("Selected #%d (%d/%d)", node.Number, model.cursor+1, len(model.flat))
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
		if item.IsCollapsed {
			marker = "▸"
		} else {
			marker = "▾"
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
	bodyHeight := max(5, innerHeight-1-footerCount)
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
