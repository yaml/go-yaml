// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tuiTitle     = "go-yaml  "
	configButton = 3
	helpButton   = 4
	outputButton = 5
)

type tuiNavItem struct {
	id       int
	label    string
	active   bool
	x, width int
}

func (m *tuiModel) navItems() []tuiNavItem {
	config, help := "Config", "Help"
	if m.configVisible {
		config += " +"
	}
	if m.helpVisible {
		help += " +"
	}
	items := []tuiNavItem{
		{id: configButton, label: config, active: m.configVisible},
		{id: helpButton, label: help, active: m.helpVisible},
	}
	x := lipgloss.Width(tuiTitle)
	for i := range items {
		items[i].label = "[" + items[i].label + "]"
		items[i].x = x
		items[i].width = lipgloss.Width(items[i].label)
		x += items[i].width + 2
	}
	return items
}

func (m *tuiModel) toolbar() string {
	var bar strings.Builder
	bar.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")).Render(tuiTitle))
	for i, item := range m.navItems() {
		if i > 0 {
			bar.WriteString("  ")
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
		if item.active {
			style = style.Bold(true).Foreground(lipgloss.Color("114"))
		}
		if m.focus == item.id {
			style = style.Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24"))
		}
		bar.WriteString(style.Render(item.label))
	}
	return bar.String()
}

func (m *tuiModel) setFocus(focus int) tea.Cmd {
	m.focus = focus
	m.input.Blur()
	m.config.Blur()
	if focus == outputButton {
		m.lastWorkspace = 1
	}
	if focus < 2 {
		m.lastWorkspace = focus
	}
	if focus == 0 {
		return m.input.Focus()
	}
	return nil
}

func (m *tuiModel) toggleConfig() tea.Cmd {
	m.helpVisible = false
	m.configVisible = !m.configVisible
	focus := m.lastWorkspace
	if m.configVisible {
		focus = 2
	} else {
		m.controls.dropdown = false
	}
	cmd := m.setFocus(focus)
	m.resize()
	return cmd
}

func (m *tuiModel) showHelp() {
	m.helpVisible = !m.helpVisible
	m.configVisible = false
	m.controls.dropdown = false
	if m.helpVisible {
		m.setFocus(2)
	} else {
		m.setFocus(m.lastWorkspace)
	}
	m.resize()
}

func (m *tuiModel) helpView() string {
	return tuiShortcutHelp("\n") + "\nCtrl-C twice: save and quit\n\nTab / Shift-Tab: navigate panels and buttons\nEnter / Space: activate a button\n[+] true   [-] false   [ ] default\nMouse: click controls; wheel scrolls output\nEnd: end of line   Ctrl-V: paste\nEsc: Close help panel"
}

func (m *tuiModel) toolbarMouse(v tea.MouseMsg) bool {
	if v.Action != tea.MouseActionPress || v.Button != tea.MouseButtonLeft {
		return false
	}
	if m.modal != "" {
		lines := strings.Split(m.modal, "\n")
		for i, line := range lines {
			if line == "[ Close ]" && v.Y == i+1 && v.X >= 2 && v.X < 11 {
				m.modal = ""
				return true
			}
		}
		return true
	}
	if v.Y != 0 {
		return false
	}
	for _, item := range m.navItems() {
		if v.X < item.x || v.X >= item.x+item.width {
			continue
		}
		switch item.id {
		case configButton:
			m.toggleConfig()
		case helpButton:
			m.showHelp()
		}
		break
	}

	return true
}

func (m *tuiModel) navigate(back bool) tea.Cmd {
	order := []int{0, 1}
	if m.configVisible || m.helpVisible {
		order = append([]int{2}, order...)
	}
	order = append(order, outputButton, configButton, helpButton)
	index := 0
	for i, f := range order {
		if f == m.focus {
			index = i
		}
	}
	delta := 1
	if back {
		delta = -1
	}
	return m.setFocus(order[(index+delta+len(order))%len(order)])
}
