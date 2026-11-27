// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tuiCopyResult struct {
	generation int
	err        error
}

func (m *tuiModel) showCommand(command string) tea.Cmd {
	m.commandGeneration++
	generation := m.commandGeneration
	m.modalTitle = "Run as CLI. Copying command..."
	m.modal = command + "\n\nEsc: close"
	write := m.copyClipboard
	if write == nil {
		write = clipboard.WriteAll
	}
	return func() tea.Msg { return tuiCopyResult{generation, write(command)} }
}

func (m *tuiModel) commandView() string {
	width := m.width - 8
	if width < 10 {
		width = 10
	}
	if width > 90 {
		width = 90
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")).Render(m.modalTitle)
	box := lipgloss.NewStyle().Width(width).Border(lipgloss.RoundedBorder()).Padding(1, 2).Render(title + "\n\n" + m.modal)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
