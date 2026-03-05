package tui

import (
	"fmt"
	"math"
	"strings"

	"github-note/internal/domain"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	width      int
	height     int
	quitting   bool
	err        error
	status     string
	collapsed  map[string]bool
	openIssue  func(url string) error
	topBar     lipgloss.Style
	helpBar    lipgloss.Style
	parentLine lipgloss.Style
	leafLine   lipgloss.Style
	focusLine  lipgloss.Style
	panel      lipgloss.Style
	statusBar  lipgloss.Style
	errorBar   lipgloss.Style
	logoLine   lipgloss.Style
}

func New(tree *domain.IssueTree, openIssue func(url string) error) *Model {
	model := &Model{
		tree:      tree,
		openIssue: openIssue,
		collapsed: make(map[string]bool),
		topBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("48")).
			Background(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1),
		helpBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("0")).
			Padding(0, 1),
		parentLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ADD8E6")).
			Background(lipgloss.Color("0")),
		leafLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("0")),
		focusLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("#D2F1FA")).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("35")).
			Background(lipgloss.Color("0")).
			Padding(0, 0),
		statusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("120")).
			Background(lipgloss.Color("0")).
			Padding(0, 1),
		errorBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Background(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1),
		logoLine: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E4FDED")).
			Background(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1),
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
			model.status = fmt.Sprintf("Selected %d/%d", model.cursor+1, len(model.flat))
		case "down", "j":
			if model.cursor < len(model.flat)-1 {
				model.cursor++
			}
			model.ensureCursorVisible()
			model.status = fmt.Sprintf("Selected %d/%d", model.cursor+1, len(model.flat))
		case "left", "h":
			model.collapseCurrent()
		case "right", "l":
			model.expandCurrent()
		case " ":
			model.toggleCurrent()
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
		innerWidth = 60
	}
	if innerHeight <= 0 {
		innerHeight = 20
	}

	if len(model.flat) == 0 {
		header := model.renderWithWidth(model.topBar, "Issue Tree", innerWidth)
		logo := model.renderLogo(innerWidth)
		help := model.renderWithWidth(model.helpBar, "No issues found. Check owner/repo permissions.", innerWidth)
		content := strings.Join([]string{header, logo, help}, "\n")
		return model.panel.Width(innerWidth).Height(innerHeight).Render(content)
	}

	header := model.renderWithWidth(model.topBar, fmt.Sprintf("Issue Tree  Items:%d  Selected:%d/%d", len(model.flat), model.cursor+1, len(model.flat)), innerWidth)
	logo := model.renderLogo(innerWidth)
	help := model.renderWithWidth(model.helpBar, "Move:j/k  Fold:h/left  Unfold:l/right  Toggle:space  Open:enter  Quit:q", innerWidth)

	listRows := max(1, innerHeight-8)
	model.ensureCursorVisibleWithRows(listRows)
	start := clamp(model.offset, 0, max(0, len(model.flat)-1))
	end := min(len(model.flat), start+listRows)

	rows := make([]string, 0, listRows)
	for i := start; i < end; i++ {
		item := model.flat[i]
		line := model.formatIssueLine(item)
		if i == model.cursor {
			rows = append(rows, model.renderWithWidth(model.focusLine, line, innerWidth))
		} else {
			if item.HasKid {
				rows = append(rows, model.renderWithWidth(model.parentLine, line, innerWidth))
			} else {
				rows = append(rows, model.renderWithWidth(model.leafLine, line, innerWidth))
			}
		}
	}
	for len(rows) < listRows {
		rows = append(rows, model.renderWithWidth(model.leafLine, "", innerWidth))
	}

	footer := model.status
	if footer == "" {
		footer = "Ready"
	}
	footerLine := model.renderWithWidth(model.statusBar, footer, innerWidth)
	if model.err != nil {
		footerLine = model.renderWithWidth(model.errorBar, "Error: "+model.err.Error(), innerWidth)
	}

	content := strings.Join([]string{header, logo, help, strings.Join(rows, "\n"), footerLine}, "\n")
	return model.panel.Width(innerWidth).Height(innerHeight).Render(content)
}

func (model *Model) renderLogo(width int) string {
	logo := []string{
		"   ________  ___ ___                  __          ",
		" /  _____/ /   |   \\    ____   _____/  |_  ____  ",
		"/   \\  ___/    ~    \\  /    \\ /  _ \\   __\\/ __ \\ ",
		"\\    \\_\\  \\    Y    / |   |  (  <_> )  | \\  ___/ ",
		" \\______  /\\___|_  /  |___|  /\\____/|__|  \\___  >",
		"        \\/       \\/        \\/                 \\/ ",
	}
	lines := make([]string, 0, len(logo))
	for _, row := range logo {
		lines = append(lines, model.renderWithWidth(model.logoLine, row, width))
	}
	return strings.Join(lines, "\n")
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
	branch := "  "
	if item.HasKid {
		if item.IsCollapsed {
			branch = "+ "
		} else {
			branch = "- "
		}
	}
	indent := strings.Repeat("  ", item.Depth)
	labels := formatLabels(item.Node.Labels)
	if labels == "" {
		return fmt.Sprintf("%s%s%s", indent, branch, item.Node.Title)
	}
	return fmt.Sprintf("%s%s%s %s", indent, branch, item.Node.Title, labels)
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
	return max(20, model.width-2)
}

func (model *Model) innerHeight() int {
	if model.height <= 0 {
		return 0
	}
	return max(8, model.height-1)
}

func (model *Model) ensureCursorVisible() {
	rows := max(1, model.innerHeight()-4)
	model.ensureCursorVisibleWithRows(rows)
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

func (model *Model) renderWithWidth(style lipgloss.Style, content string, width int) string {
	trimmed := truncateToWidth(content, max(1, width))
	return style.Width(width).MaxWidth(width).Render(trimmed)
}

func truncateToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
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
