package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vegarringdal/dotr/internal/gitstatus"
	"github.com/vegarringdal/dotr/internal/preview"
	"github.com/vegarringdal/dotr/internal/scan"
	"github.com/vegarringdal/dotr/internal/secret"
)

func (m Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("dotr…")
		v.AltScreen = true
		return v
	}

	var parts []string
	parts = append(parts, m.renderTitle())
	parts = append(parts, m.renderBody())
	if m.mode == modeFilter || m.mode == modePaste || m.mode == modeNew {
		parts = append(parts, m.styles.status.Width(m.width).Render(m.input.View()))
	}
	parts = append(parts, m.renderStatus())

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	v := tea.NewView(content)
	v.AltScreen = true
	if m.cfg.Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.WindowTitle = "dotr"
	return v
}

func (m Model) renderTitle() string {
	var label string
	switch m.mode {
	case modeIgnores:
		n := 0
		if m.ignores != nil {
			n = len(m.ignores.Patterns)
		}
		label = fmt.Sprintf("dotr  ignores  %d", n)
	case modeRestore:
		label = fmt.Sprintf("dotr  restore  %d", len(m.snapshots))
	case modeStow:
		label = fmt.Sprintf("dotr  stow  %d", len(m.stowPkgs))
	case modeHelp:
		label = "dotr  help"
	case modeFilter:
		label = fmt.Sprintf("dotr  %d/%d  filter", len(m.filtered), len(m.entries))
	case modePaste:
		label = "dotr  paste"
	case modeNew:
		label = "dotr  new"
	case modeConfirm:
		label = "dotr  confirm"
	default:
		label = fmt.Sprintf("dotr  %d/%d", len(m.filtered), len(m.entries))
		if m.filter != "" {
			label += fmt.Sprintf("  /%s", m.filter)
		}
		if m.yankPath != "" {
			label += "  ⎘"
		}
		if m.scanning {
			label = "dotr  scanning…"
		}
	}
	return m.styles.title.Width(m.width).Render(truncate(label, m.width))
}

func (m Model) renderStatus() string {
	var help string
	switch m.mode {
	case modeIgnores:
		help = "j/k  d unignore  e edit  esc"
	case modeFilter:
		help = "type to filter  enter apply  esc clear"
	case modePaste:
		help = "enter paste  esc cancel"
	case modeNew:
		help = "enter create+edit  esc cancel"
	case modeRestore:
		help = "j/k  enter restore  esc back"
	case modeStow:
		help = "j/k  enter/l link  u unlink  R restow  esc"
	case modeConfirm:
		help = "enter confirm  esc cancel"
	case modeHelp:
		help = "any key to close"
	default:
		help = "/ n s y/Y p b R i , ?  q"
	}
	left := m.status
	if left == "" {
		left = help
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(help) - 2
	if gap < 1 {
		return m.styles.status.Width(m.width).Render(truncate(left, m.width-2))
	}
	line := left + strings.Repeat(" ", gap) + help
	return m.styles.status.Width(m.width).Render(truncate(line, m.width))
}

func (m Model) renderBody() string {
	switch m.mode {
	case modeIgnores:
		return m.renderIgnoresBody()
	case modeRestore:
		return m.renderRestoreBody()
	case modeStow:
		return m.renderStowBody()
	case modeHelp:
		return m.renderHelpBody()
	case modeConfirm:
		return m.renderConfirmBody()
	}

	leftW, rightW := m.paneWidths()
	_, h := m.contentSize()

	listStyle := m.styles.panel
	prevStyle := m.styles.panel
	if m.focus == focusList {
		listStyle = m.styles.panelActive
	} else {
		prevStyle = m.styles.panelActive
	}

	listInnerW := max(1, leftW-4)
	listInnerH := max(1, h)
	listContent := fillHeight(m.renderList(listInnerW, listInnerH), listInnerH)

	prevInnerW := max(1, rightW-4)
	prevHeader := m.renderPreviewHeader(prevInnerW)
	prevBody := fillHeight(prevHeader+"\n"+m.viewport.View(), listInnerH)

	left := listStyle.Width(leftW).Render(listContent)
	right := prevStyle.Width(rightW).Render(prevBody)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderConfirmBody() string {
	_, h := m.contentSize()
	innerW := max(1, m.width-4)
	msg := m.status
	if msg == "" {
		msg = "confirm?"
	}
	body := m.styles.secret.Render(msg) + "\n\n" + m.styles.muted.Render(m.confirmPath)
	return m.styles.panelActive.Width(m.width).Render(fillHeight(truncate(body, innerW), h))
}

func (m Model) renderIgnoresBody() string {
	_, h := m.contentSize()
	innerW := max(1, m.width-4)
	return m.styles.panelActive.Width(m.width).Render(
		fillHeight(m.renderIgnoreList(innerW, h), h),
	)
}

func (m Model) renderRestoreBody() string {
	_, h := m.contentSize()
	innerW := max(1, m.width-4)
	return m.styles.panelActive.Width(m.width).Render(
		fillHeight(m.renderSnapList(innerW, h), h),
	)
}

func (m Model) renderStowBody() string {
	_, h := m.contentSize()
	innerW := max(1, m.width-4)
	header := m.styles.muted.Render(truncate(
		fmt.Sprintf("%s → %s", m.stowOpts.Dir, m.stowOpts.Target), innerW))
	body := m.renderStowList(innerW, max(1, h-1))
	return m.styles.panelActive.Width(m.width).Render(
		fillHeight(header+"\n"+body, h),
	)
}

func (m Model) renderHelpBody() string {
	_, h := m.contentSize()
	innerW := max(1, m.width-4)
	return m.styles.panelActive.Width(m.width).Render(
		fillHeight(wrapHelp(helpText, innerW), h),
	)
}

func (m Model) renderIgnoreList(width, height int) string {
	if m.ignores == nil || len(m.ignores.Patterns) == 0 {
		return m.styles.muted.Render("no ignores yet — press i on a config to add one")
	}
	var b strings.Builder
	end := m.ignoreOffset + height
	if end > len(m.ignores.Patterns) {
		end = len(m.ignores.Patterns)
	}
	for i := m.ignoreOffset; i < end; i++ {
		line := m.ignores.Patterns[i]
		if i == m.ignoreCursor {
			b.WriteString(m.styles.listCursor.Width(width).Render(truncate(line, width)))
		} else {
			b.WriteString(m.styles.listNormal.Width(width).Render(truncate(line, width)))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderSnapList(width, height int) string {
	if len(m.snapshots) == 0 {
		return m.styles.muted.Render("no backups — press b on a file to create one")
	}
	var b strings.Builder
	end := m.snapOff + height
	if end > len(m.snapshots) {
		end = len(m.snapshots)
	}
	for i := m.snapOff; i < end; i++ {
		line := m.snapshots[i].DisplayName()
		if i == m.snapCur {
			b.WriteString(m.styles.listCursor.Width(width).Render(truncate(line, width)))
		} else {
			b.WriteString(m.styles.listNormal.Width(width).Render(truncate(line, width)))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderStowList(width, height int) string {
	if len(m.stowPkgs) == 0 {
		return m.styles.muted.Render("no stow packages — set stow_dir or ~/.stowrc")
	}
	var b strings.Builder
	end := m.stowOff + height
	if end > len(m.stowPkgs) {
		end = len(m.stowPkgs)
	}
	for i := m.stowOff; i < end; i++ {
		p := m.stowPkgs[i]
		line := fmt.Sprintf("%s %-16s  %d↑ %d· %d!",
			p.Status.Mark(), p.Name, p.Linked, p.Missing, p.Conflicts)
		if i == m.stowCur {
			b.WriteString(m.styles.listCursor.Width(width).Render(truncate(line, width)))
		} else {
			b.WriteString(m.styles.listNormal.Width(width).Render(truncate(line, width)))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderPreviewHeader(width int) string {
	e, ok := m.selected()
	if !ok {
		return m.styles.muted.Render(truncate("preview", width))
	}
	name := displayName(e)
	meta := m.previewLang
	if meta == "" {
		meta = "?"
	}
	badge := m.previewBadge
	var badgeStyled string
	switch m.previewParse {
	case preview.ParseOK:
		badgeStyled = m.styles.parseOK.Render(badge)
	case preview.ParseFail:
		badgeStyled = m.styles.parseFail.Render(badge)
	}
	parts := []string{truncate(name, max(8, width-24)), meta}
	if badgeStyled != "" {
		parts = append(parts, badgeStyled)
	}
	if secret.Path(e.AbsPath) {
		parts = append(parts, m.styles.secret.Render("🔒"))
	}
	if m.git != nil {
		if g := m.git[e.AbsPath]; g != gitstatus.None && g != gitstatus.Clean {
			parts = append(parts, m.styles.parseFail.Render("git:"+g.String()))
		} else if g == gitstatus.Clean {
			parts = append(parts, m.styles.muted.Render("git:·"))
		}
	}
	parts = append(parts, formatSize(e.Size))
	if e.Symlink {
		parts = append(parts, "→")
	}
	line := strings.Join(parts, "  ")
	return m.styles.muted.Render(truncate(line, width))
}

func displayName(e scan.Entry) string {
	switch e.App {
	case "home", "config":
		return e.RelPath
	default:
		return filepath.Join(e.App, e.RelPath)
	}
}

func (m Model) renderList(width, height int) string {
	if m.scanning {
		return m.styles.muted.Render("scanning…")
	}
	if len(m.filtered) == 0 {
		if m.filter != "" {
			return m.styles.muted.Render("no matches")
		}
		return m.styles.muted.Render("no configs found")
	}

	var b strings.Builder
	end := m.offset + height
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := m.offset; i < end; i++ {
		e := m.entries[m.filtered[i]]
		line := formatEntry(e, m.git, m.stowOwners)
		if i == m.cursor {
			b.WriteString(m.styles.listCursor.Width(width).Render(truncate(line, width)))
		} else {
			b.WriteString(m.styles.listNormal.Width(width).Render(truncate(line, width)))
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func formatEntry(e scan.Entry, git map[string]gitstatus.Kind, owners map[string]string) string {
	mark := " "
	if e.Symlink {
		mark = "↗"
	}
	if owners != nil {
		if _, ok := owners[e.AbsPath]; ok {
			mark = "S"
		}
	}
	if secret.Path(e.AbsPath) {
		mark = "🔒"
	}
	g := " "
	if git != nil {
		if m := git[e.AbsPath].Mark(); m != "" {
			g = m
		}
	}
	return g + mark + " " + displayName(e)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func fillHeight(s string, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func wrapHelp(s string, width int) string {
	if width < 20 {
		width = 20
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if lipgloss.Width(line) <= width {
			out = append(out, line)
			continue
		}
		out = append(out, truncate(line, width))
	}
	return strings.Join(out, "\n")
}
