package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *tuiModel) outputSelectorLabel() string { return "[" + tuiModeNames[m.controls.mode] + " ▾]" }

func (m *tuiModel) outputSelectorRect() tuiRect {
	r := m.controls.rects[1]
	return tuiRect{x: r.x + 1 + len("OUTPUT  "), y: r.y + 1, w: lipgloss.Width(m.outputSelectorLabel()), h: 1}
}

func (m *tuiModel) outputSelectorClicked(v tea.MouseMsg) bool {
	if v.Action != tea.MouseActionPress || v.Button != tea.MouseButtonLeft {
		return false
	}
	focus := m.focus
	if focus >= configButton {
		focus = m.lastWorkspace
	}
	if (m.controls.layout == 3 || m.width < 70 || m.height < 17) && focus != 1 {
		return false
	}
	r := m.controls.rects[1]
	return m.outputSelectorRect().contains(v.X, v.Y) && v.X < r.x+r.w-1
}

func (m *tuiModel) outputMenuRect() tuiRect {
	selector := m.outputSelectorRect()
	width := 32
	if m.width < width {
		width = m.width
	}
	x := selector.x
	if x+width > m.width {
		x = m.width - width
	}
	height := len(tuiModes) + 2
	y := selector.y + 1
	if y+height > m.height-4 {
		y = m.height - 4 - height
	}
	if y < 1 {
		y = 1
	}
	return tuiRect{x: x, y: y, w: width, h: height}
}

func (m *tuiModel) openOutputMenu() {
	m.setFocus(outputButton)
	m.outputMenu = true
	m.outputChoice = m.controls.mode
}

func (m *tuiModel) outputMenuView() string {
	rows := []string{"Select output:"}
	for i, name := range tuiModeNames {
		prefix := "  "
		if i == m.outputChoice {
			prefix = "› "
		}
		rows = append(rows, prefix+name)
	}
	rows = append(rows, "Enter: select  Esc: cancel")
	background := strings.Split(m.panelsView(), "\n")
	rect := m.outputMenuRect()
	for len(background) < rect.y-1+len(rows) {
		background = append(background, "")
	}
	for i, line := range rows {
		row := rect.y - 1 + i
		menu := lipgloss.NewStyle().Width(rect.w).MaxWidth(rect.w).Background(lipgloss.Color("235")).Render(line)
		prefix := ansi.Truncate(background[row], rect.x, "")
		prefix += strings.Repeat(" ", maxTUI(0, rect.x-lipgloss.Width(prefix)))
		background[row] = prefix + menu + ansi.TruncateLeft(background[row], rect.x+rect.w, "")
	}

	return strings.Join(background, "\n")
}

func (m *tuiModel) chooseOutput() {
	old := m.controls.selected
	m.controls.selected = 1
	m.changeControl(m.outputChoice - m.controls.mode)
	m.controls.selected = old
	m.outputMenu = false
}

func (m *tuiModel) outputMenuKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.outputMenu = false
	case "up":
		m.outputChoice = (m.outputChoice + len(tuiModes) - 1) % len(tuiModes)
	case "down":
		m.outputChoice = (m.outputChoice + 1) % len(tuiModes)
	case "enter", " ":
		m.chooseOutput()
	}
	return nil
}

func (m *tuiModel) outputMenuMouse(v tea.MouseMsg) tea.Cmd {
	if v.Action == tea.MouseActionPress && v.Button == tea.MouseButtonLeft {
		rect := m.outputMenuRect()
		index := v.Y - rect.y - 1
		if rect.contains(v.X, v.Y) && index >= 0 && index < len(tuiModes) {
			m.outputChoice = index
			m.chooseOutput()
		} else {
			m.outputMenu = false
		}
	}
	return nil
}
