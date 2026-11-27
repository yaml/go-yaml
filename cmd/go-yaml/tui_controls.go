package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	tuiModes     = []string{"-y", "-Y", "-j", "-J", "-t", "-T", "-e", "-E", "-n", "-N"}
	tuiModeNames = []string{"YAML", "YAML preserved", "JSON compact", "JSON pretty", "Tokens", "Tokens detailed", "Events", "Events detailed", "Nodes", "Nodes detailed"}
	tuiLayouts   = []string{"Columns", "Stacked", "Rows", "Focus"}
	tuiBooleans  = []string{"compact-seq-indent", "unicode", "unique-keys", "canonical", "explicit-start", "explicit-end", "flow-simple-coll", "known-fields", "single-document", "all-documents", "stream-nodes"}
)

type tuiRect struct{ x, y, w, h int }

func (r tuiRect) contains(x, y int) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type tuiControls struct {
	layout, selected, mode int
	dropdown               bool
	rects                  [3]tuiRect
	split                  float64
	dragging               bool
}

func (m *tuiModel) syncControls() {
	s, err := m.session()
	if err != nil {
		return
	}
	m.controls.split = s.Split
	m.controls.layout = s.Layout
	if m.controls.layout < 0 || m.controls.layout > 3 {
		m.controls.layout = 0
	}
	for _, a := range s.Args {
		for i, v := range tuiModes {
			if a == v {
				m.controls.mode = i
			}
		}
	}
}
func tuiResetIndex() int   { return 3 + len(tuiBooleans) }
func tuiControlCount() int { return tuiResetIndex() + 1 }

func (m *tuiModel) resetConfig() {
	data, err := encodeTUISession(tuiSession{Version: 1, Live: true, Args: []string{"-y"}})
	if err != nil {
		m.status = err.Error()
		return
	}
	m.config.SetValue(string(data))
	m.controls.dropdown = false
	m.syncControls()
	m.resize()
	if err = m.save(); err != nil {
		m.status = err.Error()
		return
	}
	m.status = "Configuration reset"
}

func (m *tuiModel) changeControl(delta int) {
	if m.controls.selected == tuiResetIndex() {
		m.resetConfig()
		return
	}
	s, err := m.session()
	if err != nil {
		m.status = err.Error()
		return
	}
	switch m.controls.selected {
	case 0:
		m.controls.layout = (m.controls.layout + delta + 4) % 4
		s.Layout = m.controls.layout
	case 1:
		m.controls.mode = (m.controls.mode + delta + len(tuiModes)) % len(tuiModes)
		var args []string
		for _, a := range s.Args {
			found := false
			for _, v := range tuiModes {
				if a == v {
					found = true
				}
			}
			if !found {
				args = append(args, a)
			}
		}
		s.Args = append([]string{tuiModes[m.controls.mode]}, args...)
	case 2:
		s.Live = !s.Live
	default:
		index := m.controls.selected - 3
		if index >= len(tuiBooleans) {
			return
		}
		name := tuiBooleans[index]
		state := 0
		var args []string
		for i := 0; i < len(s.Args); i++ {
			if s.Args[i] == "-o" && i+1 < len(s.Args) && (s.Args[i+1] == name || s.Args[i+1] == "no-"+name) {
				if s.Args[i+1] == name {
					state = 1
				} else {
					state = 2
				}
				i++
				continue
			}
			args = append(args, s.Args[i])
		}
		state = (state + delta + 3) % 3
		if state == 1 {
			args = append(args, "-o", name)
		}
		if state == 2 {
			args = append(args, "-o", "no-"+name)
		}
		s.Args = args
	}
	data, err := encodeTUISession(s)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.config.SetValue(string(data))
	m.resize()
	if err = m.save(); err != nil {
		m.status = err.Error()
	}
}

func (m *tuiModel) controlView() string {
	s, _ := m.session()
	live := "[-]"
	if s.Live {
		live = "[+]"
	}
	rows := []string{"Layout: " + tuiLayouts[m.controls.layout] + " ▾", "Output: " + tuiModeNames[m.controls.mode] + " ▾", live + " Live run"}
	for _, name := range tuiBooleans {
		state := "[ ]"
		for _, a := range s.Args {
			if a == name {
				state = "[+]"
			}
			if a == "no-"+name {
				state = "[-]"
			}
		}
		rows = append(rows, state+" "+name)
	}
	for i := range rows {
		if i == m.controls.selected {
			rows[i] = "› " + rows[i]
		} else {
			rows[i] = "  " + rows[i]
		}
	}
	rows = append(rows[:2], append([]string{"", "  Options:"}, rows[2:]...)...)
	reset := "  [Reset]"
	if m.controls.selected == tuiResetIndex() {
		reset = "› [Reset]"
	}
	rows = append(rows, "", reset, "", "[+] true  [-] false [ ] default", "Enter/Space: change", tuiKeyLabel(tuiEditKey)+": Edit config in Vim", "Esc: Close config panel")
	if m.controls.dropdown {
		items := tuiLayouts
		if m.controls.selected == 1 {
			items = tuiModeNames
		}
		rows = []string{"Choose (arrows, Enter, Esc):"}
		for _, v := range items {
			rows = append(rows, "  "+v)
		}
	}
	return strings.Join(rows, "\n")
}

func (m *tuiModel) geometry() {
	w, h := m.width, m.height-5
	if w < 8 {
		w = 8
	}
	if h < 6 {
		h = 6
	}
	layout := m.controls.layout
	if w < 70 || h < 12 {
		layout = 3
	}
	split := m.controls.split
	if split < 0.2 || split > 0.8 {
		split = 0.7
	}
	if !(m.configVisible || m.helpVisible) {
		m.controls.rects[2] = tuiRect{}
		switch layout {
		case 0:
			m.controls.rects[0] = tuiRect{0, 1, w / 2, h}
			m.controls.rects[1] = tuiRect{w / 2, 1, w - w/2, h}
		case 3:
			m.controls.rects[0] = tuiRect{0, 1, w, h}
			m.controls.rects[1] = tuiRect{0, 1, w, h}
		default:
			m.controls.rects[0] = tuiRect{0, 1, w, h / 2}
			m.controls.rects[1] = tuiRect{0, 1 + h/2, w, h - h/2}
		}
		return
	}
	switch layout {
	case 0:
		workspace := int(float64(w) * split)
		sidebar := w - workspace
		a := workspace / 2
		m.controls.rects = [3]tuiRect{{sidebar, 1, a, h}, {sidebar + a, 1, workspace - a, h}, {0, 1, sidebar, h}}
	case 1:
		workspace := int(float64(w) * split)
		sidebar := w - workspace
		a := h / 2
		m.controls.rects = [3]tuiRect{{sidebar, 1, workspace, a}, {sidebar, 1 + a, workspace, h - a}, {0, 1, sidebar, h}}
	case 2:
		a := h / 3
		top := h - 2*a
		m.controls.rects = [3]tuiRect{{0, 1 + top, w, a}, {0, 1 + top + a, w, a}, {0, 1, w, top}}
	case 3:
		for i := range m.controls.rects {
			m.controls.rects[i] = tuiRect{0, 1, w, h}
		}
	}
}

func (m *tuiModel) panelsView() string {
	m.geometry()
	content := []string{m.input.View(), m.output.View(), m.controlView()}
	titles := []string{"INPUT", "OUTPUT", "CONFIG"}
	if m.helpVisible {
		titles[2] = "HELP"
		content[2] = m.helpView()
	}
	panels := make([]string, 3)
	for i, r := range m.controls.rects {
		if i == 2 && !(m.configVisible || m.helpVisible) {
			continue
		}
		color := "240"
		if i == m.focus || i == 1 && m.focus == outputButton {
			color = "45"
		}
		title := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(titles[i])
		if i == 0 && r.w-2 >= len("INPUT  ")+len(tuiClearLabel) {
			title += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render(tuiClearLabel)
		}
		if i == 1 {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
			if m.outputMenu {
				style = style.Bold(true).Foreground(lipgloss.Color("114"))
			}
			if m.focus == outputButton {
				style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24"))
			}
			title += "  " + style.Render(m.outputSelectorLabel())
		}
		panels[i] = lipgloss.NewStyle().Width(r.w - 2).Height(r.h - 2).MaxWidth(r.w).MaxHeight(r.h).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(color)).Render(title + "\n" + content[i])
	}
	switch {
	case m.controls.layout == 3 || m.width < 70 || m.height < 17:
		focus := m.focus
		if focus >= configButton {
			focus = m.lastWorkspace
		}
		return panels[focus]
	case m.controls.layout == 1 && !(m.configVisible || m.helpVisible):
		return lipgloss.JoinVertical(lipgloss.Left, panels[0], panels[1])
	case m.controls.layout == 1:
		return lipgloss.JoinHorizontal(lipgloss.Top, panels[2], lipgloss.JoinVertical(lipgloss.Left, panels[0], panels[1]))
	case m.controls.layout == 2:
		if !(m.configVisible || m.helpVisible) {
			return lipgloss.JoinVertical(lipgloss.Left, panels[0], panels[1])
		}
		return lipgloss.JoinVertical(lipgloss.Left, panels[2], panels[0], panels[1])
	default:
		return lipgloss.JoinHorizontal(lipgloss.Top, panels[2], panels[0], panels[1])
	}
}

func (m *tuiModel) mouse(v tea.MouseMsg) {
	if v.Action == tea.MouseActionRelease {
		if m.controls.dragging {
			s, err := m.session()
			if err == nil {
				s.Split = m.controls.split
				data, _ := encodeTUISession(s)
				m.config.SetValue(string(data))
				if err = m.save(); err != nil {
					m.status = err.Error()
				}
			}
		}
		m.controls.dragging = false
		return
	}
	if m.controls.dragging && v.Action == tea.MouseActionMotion {
		m.controls.split = 1 - float64(v.X+1)/float64(m.width)
		if m.controls.split < 0.2 {
			m.controls.split = 0.2
		}
		if m.controls.split > 0.8 {
			m.controls.split = 0.8
		}
		m.resize()
		return
	}
	for i, r := range m.controls.rects {
		visibleFocus := m.focus
		if visibleFocus >= configButton {
			visibleFocus = m.lastWorkspace
		}
		if (m.controls.layout == 3 || m.width < 70 || m.height < 17) && i != visibleFocus {
			continue
		}
		if !r.contains(v.X, v.Y) {
			continue
		}
		m.setFocus(i)
		m.input.Blur()
		if i == 0 {
			m.input.Focus()
		}
		if v.Button == tea.MouseButtonLeft && v.X == r.x+r.w-1 && i == 2 && m.controls.layout < 2 && m.width >= 70 && m.height >= 17 {
			m.controls.dragging = true
			return
		}
		if i == 2 && m.helpVisible {
			return
		}
		if i == 2 {
			if v.Action != tea.MouseActionPress || v.Button != tea.MouseButtonLeft {
				return
			}
			if m.controls.dropdown {
				row := v.Y - r.y - 3
				limit := 4
				if m.controls.selected == 1 {
					limit = len(tuiModes)
				}
				if row >= 0 && row < limit {
					current := m.controls.layout
					if m.controls.selected == 1 {
						current = m.controls.mode
					}
					m.changeControl(row - current)
					m.controls.dropdown = false
				}
				return
			}
			row := tuiControlAtRow(v.Y - r.y - 2)
			if row >= 0 && row < tuiControlCount() {
				m.controls.selected = row
				if row < 2 {
					m.controls.dropdown = true
				} else {
					m.changeControl(1)
				}
			}
			return
		}
		if i == 1 {
			m.output, _ = m.output.Update(v)
		}
		if i == 0 && v.Button == tea.MouseButtonLeft {
			m.input.SetCursor(v.X - r.x - 4)
		}
		return
	}
}

func (m *tuiModel) controlsKey(key string) (bool, tea.Cmd) {
	if m.controls.dropdown {
		switch key {
		case "esc", "enter":
			m.controls.dropdown = false
		case "up", "left":
			m.changeControl(-1)
		case "down", "right":
			m.changeControl(1)
		}
		return true, nil
	}
	if key == tuiLayoutKey {
		old := m.controls.selected
		m.controls.selected = 0
		m.changeControl(1)
		m.controls.selected = old
		return true, nil
	}
	if m.focus != 2 || m.helpVisible {
		return false, nil
	}
	switch key {
	case "up":
		m.controls.selected = (m.controls.selected + tuiControlCount() - 1) % tuiControlCount()
		return true, nil
	case "down":
		m.controls.selected = (m.controls.selected + 1) % tuiControlCount()
		return true, nil
	case "enter", " ", "right", "left":
		delta := 1
		if key == "left" {
			delta = -1
		}
		m.changeControl(delta)

		return true, nil
	}
	return false, nil
}

// tuiControlAtRow maps rendered content rows to selectable controls.
// The blank separator and Options heading are not selectable.
func tuiControlAtRow(row int) int {
	if row == tuiResetIndex()+3 {
		return tuiResetIndex()
	}
	if row < 0 {
		return -1
	}
	if row < 2 {
		return row
	}
	if row < 4 {
		return -1
	}
	index := row - 2
	if index >= 3+len(tuiBooleans) {
		return -1
	}
	return index
}
