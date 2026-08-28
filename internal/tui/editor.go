package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/editor"
)

type editorFinishedMsg struct {
	path string
	err  error
}

func openEditor(path string) tea.Cmd {
	cmd, err := editor.Cmd("dotr", path)
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{path: path, err: err}
		}
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{path: path, err: err}
	})
}

func formatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	}
}
