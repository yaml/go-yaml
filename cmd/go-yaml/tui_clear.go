// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import tea "github.com/charmbracelet/bubbletea"

const tuiClearLabel = "[Clear]"

func (m *tuiModel) clearButtonRect() tuiRect {
	r := m.controls.rects[0]
	// Account for the border, INPUT title, and two separating spaces.
	return tuiRect{x: r.x + 1 + len("INPUT  "), y: r.y + 1, w: len(tuiClearLabel), h: 1}
}

func (m *tuiModel) clearButtonClicked(v tea.MouseMsg) bool {
	if v.Action != tea.MouseActionPress || v.Button != tea.MouseButtonLeft {
		return false
	}
	focus := m.focus
	if focus >= configButton {
		focus = m.lastWorkspace
	}
	if (m.controls.layout == 3 || m.width < 70 || m.height < 17) && focus != 0 {
		return false
	}
	button := m.clearButtonRect()
	panel := m.controls.rects[0]
	return button.x+button.w <= panel.x+panel.w-1 && button.contains(v.X, v.Y)
}

func (m *tuiModel) clearInput() tea.Cmd {
	m.input.SetValue("")
	m.setFocus(0)
	m.revision++
	if m.cancel != nil {
		m.cancel()
	}
	m.status = "Input cleared; output stale"
	if err := m.save(); err != nil {
		m.status = err.Error()
		return nil
	}
	session, err := m.session()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if session.Live {
		return m.run()
	}
	return nil
}
