package tui

import tea "github.com/charmbracelet/bubbletea"

func Run(model *Model) error {
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}
