package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const junkSettle = 45 * time.Millisecond

type junkTickMsg struct {
	id  int
	buf string
}

func looksLikeTermReply(s string) bool {
	if s == "" {
		return false
	}
	low := strings.ToLower(s)
	if strings.Contains(low, "rgb:") || strings.Contains(low, "11;rgb") || strings.Contains(low, "]11;") {
		return true
	}
	if colorTriple(low) {
		return true
	}
	if strings.Contains(low, "/") && hexSlashOnly(low) && digitCount(low) >= 4 {
		return true
	}
	return false
}

func colorTriple(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if len(p) < 2 || len(p) > 4 || !hexOnly(p) {
			return false
		}
	}
	return true
}

func hexSlashOnly(s string) bool {
	for _, r := range s {
		if r != '/' && r != ':' && r != '#' && !hexRune(r) {
			return false
		}
	}
	return true
}

func hexOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !hexRune(r) {
			return false
		}
	}
	return true
}

func hexRune(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func digitCount(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func dropBuffered(buf string) bool {
	if looksLikeTermReply(buf) {
		return true
	}
	switch strings.ToLower(buf) {
	case "rgb", "rgb:":
		return true
	}
	return false
}

func isJunkStarter(s string) bool {
	switch s {
	case "/", "#", "r", "R":
		return true
	}
	return len(s) == 1 && s[0] >= '0' && s[0] <= '9'
}

func isJunkChar(s string) bool {
	if len(s) != 1 {
		return false
	}
	r := rune(s[0])
	switch r {
	case '/', ':', '#', ';', 'r', 'g', 'b', 'R', 'G', 'B':
		return true
	}
	return hexRune(r)
}

func keyPayload(msg tea.KeyPressMsg) string {
	if t := msg.Key().Text; t != "" {
		return t
	}
	return msg.String()
}

func (m Model) ingestBrowseKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	s := keyPayload(msg)
	if looksLikeTermReply(s) {
		m.junkBuf = ""
		return m, nil
	}
	// KeyExtended / coalesced dump that is not a single binding.
	if len([]rune(s)) > 1 {
		if looksLikeTermReply(s) || hexSlashOnly(strings.ToLower(s)) && digitCount(s) >= 4 {
			m.junkBuf = ""
			return m, nil
		}
	}

	if m.junkBuf != "" || isJunkStarter(s) {
		if isJunkChar(s) {
			m.junkBuf += s
			if dropBuffered(m.junkBuf) {
				m.junkBuf = ""
				return m, nil
			}
			m.junkID++
			id := m.junkID
			buf := m.junkBuf
			return m, tea.Tick(junkSettle, func(time.Time) tea.Msg {
				return junkTickMsg{id: id, buf: buf}
			})
		}
		// Real key after a short prefix (e.g. "/" then "n").
		return m.flushJunkThen(msg)
	}

	return m.updateBrowseKey(msg)
}

func (m Model) flushJunkThen(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	buf := m.junkBuf
	m.junkBuf = ""
	m.junkID++
	if dropBuffered(buf) {
		return m.ingestBrowseKey(msg)
	}
	if buf == "" {
		return m.updateBrowseKey(msg)
	}
	first := string([]rune(buf)[0])
	m, cmd := m.updateBrowseKey(fakeKey(first))
	if m.mode == modeFilter {
		rest := strings.TrimPrefix(buf, first)
		if rest != "" && !looksLikeTermReply(rest) && !looksLikeTermReply(buf) {
			m.input.SetValue(m.input.Value() + rest)
			m.filter = m.input.Value()
			m.rebuildFilter()
		}
		next, more := m.updateFilterKey(msg)
		if nm, ok := next.(Model); ok {
			m = nm
		}
		return m, tea.Batch(cmd, more)
	}
	m2, cmd2 := m.updateBrowseKey(msg)
	return m2, tea.Batch(cmd, cmd2)
}

func (m Model) flushJunkTimeout(buf string) (Model, tea.Cmd) {
	if m.junkBuf != buf {
		return m, nil
	}
	m.junkBuf = ""
	if dropBuffered(buf) || buf == "" {
		return m, nil
	}
	first := string([]rune(buf)[0])
	m, cmd := m.updateBrowseKey(fakeKey(first))
	if m.mode == modeFilter {
		rest := strings.TrimPrefix(buf, first)
		if rest != "" && !looksLikeTermReply(buf) {
			m.input.SetValue(m.input.Value() + rest)
			m.filter = m.input.Value()
			m.rebuildFilter()
		}
	}
	return m, cmd
}

func fakeKey(s string) tea.KeyPressMsg {
	r := '/'
	if rs := []rune(s); len(rs) > 0 {
		r = rs[0]
	}
	return tea.KeyPressMsg{Code: r, Text: s}
}

func isFilterJunk(q string) bool {
	return looksLikeTermReply(strings.TrimSpace(q))
}

func (m Model) abortJunkFilter() (Model, tea.Cmd) {
	m.mode = modeBrowse
	m.input.Blur()
	m.filter = ""
	m.input.SetValue("")
	m.rebuildFilter()
	m.status = ""
	return m, nil
}
