package main

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIRunArgs(t *testing.T) {
	for _, mode := range []string{"-Y", "-j", "-T", "--NODE"} {
		args := []string{mode, "-o", "v3"}
		if !reflect.DeepEqual(tuiRunArgs(args), args) {
			t.Fatalf("changed explicit mode %s", mode)
		}
	}
	if got := tuiRunArgs(nil); !reflect.DeepEqual(got, []string{"-y"}) {
		t.Fatal(got)
	}
}

func TestTUIShellQuote(t *testing.T) {
	if got := shellQuote("a'b"); got != "'a'\"'\"'b'" {
		t.Fatal(got)
	}
}

func TestTUILayoutGeometry(t *testing.T) {
	for layout := 0; layout < 4; layout++ {
		m := tuiModel{width: 120, height: 40, configVisible: true, controls: tuiControls{layout: layout}}
		m.geometry()
		for _, r := range m.controls.rects {
			if r.w <= 0 || r.h <= 0 || r.x+r.w > 120 || r.y+r.h > 40 {
				t.Fatalf("layout %d: %+v", layout, r)
			}
		}
	}
}

func TestTUIShortcuts(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	}()
	newModel := func(focus int) *tuiModel {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, focus: focus}
		m.input.SetValue("hello: world")
		m.config.CharLimit = 0
		m.config.SetValue("version: 1\nargs: [-y]\nlive: false\n")
		m.resize()
		return m
	}
	keys := []tea.KeyMsg{
		{Type: tea.KeyCtrlE},
		{Type: tea.KeyCtrlR},
		{Type: tea.KeyCtrlS},
		{Type: tea.KeyCtrlL},
		{Type: tea.KeyCtrlQ},
	}
	for focus := 0; focus < 3; focus++ {
		for i, key := range keys {
			m := newModel(focus)
			_, cmd := m.Update(key)
			switch i {
			case 0:
				if focus != 1 && cmd == nil {
					t.Fatal("editor not launched")
				}
			case 1:
				if cmd == nil || m.status != "Running" {
					t.Fatal("run not scheduled")
				}
				m.cancel()
			case 2:
				if !strings.Contains(m.modal, ".go-yaml/input.yaml") {
					t.Fatal("command not shown")
				}
			case 3:
				if m.controls.layout != 1 {
					t.Fatal("layout not changed")
				}
			case 4:
				if cmd == nil {
					t.Fatal("quit not scheduled")
				}
			}
		}
	}
	for _, key := range keys[:4] {
		m := newModel(0)
		m.modal = "dialog"
		m.Update(key)
		if m.modal != "dialog" || m.controls.layout != 0 {
			t.Fatal("shortcut escaped modal")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.modal != "" {
			t.Fatal("Escape did not close modal")
		}
	}
	m := newModel(0)
	m.input.Focus()
	m.input.CursorStart()
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.input.LineInfo().ColumnOffset == 0 {
		t.Fatal("End did not move cursor")
	}
	if !m.input.KeyMap.Paste.Enabled() {
		t.Fatal("paste disabled")
	}
	for _, key := range []tea.KeyType{tea.KeyF1, tea.KeyF2, tea.KeyF3, tea.KeyF5, tea.KeyF6, tea.KeyF7} {
		m := newModel(1)
		m.Update(tea.KeyMsg{Type: key})
		if m.modal != "" || m.focus != 1 || m.controls.layout != 0 {
			t.Fatal("function key still bound")
		}
	}
}

func TestTUIToolbar(t *testing.T) {
	m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40}
	m.config.CharLimit = 0
	m.config.SetValue("version: 1\nargs: [-y]\nlive: false\n")
	m.resize()
	if m.controls.rects[2].w != 0 {
		t.Fatal("config starts visible")
	}
	m.setFocus(1)
	m.Update(tea.MouseMsg{X: testNavX(m, configButton), Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !m.configVisible || m.focus != 2 {
		t.Fatal("config did not open")
	}
	m.toggleConfig()
	if m.focus != 1 {
		t.Fatal("workspace focus lost")
	}
	m.navigate(false)
	if m.focus != outputButton {
		t.Fatal("output not first in nav order")
	}
	m.navigate(false)
	if m.focus != configButton {
		t.Fatal("hidden config in tab order")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.configVisible {
		t.Fatal("button inactive")
	}
	m.setFocus(helpButton)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.helpVisible || m.configVisible || m.modal != "" {
		t.Fatal("help is not a third pane")
	}
	if !strings.Contains(m.panelsView(), "HELP") {
		t.Fatal("help pane missing")
	}
	m.Update(tea.MouseMsg{X: testNavX(m, helpButton), Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if m.helpVisible {
		t.Fatal("Help button did not close pane")
	}
	for _, r := range []rune{'h', '1'} {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true})
		if m.modal != "" {
			t.Fatal("removed Alt shortcut activated")
		}
	}
	if !strings.Contains(m.controlView(), "[ ] compact-seq-indent") || !strings.Contains(m.controlView(), "[-] Live run") {
		t.Fatal(m.controlView())
	}
	m.config.CharLimit = 0
	m.config.SetValue("version: 1\nargs: [-y, -o, compact-seq-indent, -o, no-unicode]\nlive: true\n")
	if !strings.Contains(m.controlView(), "[+] compact-seq-indent") || !strings.Contains(m.controlView(), "[-] unicode") {
		t.Fatal(m.controlView())
	}
	for layout := 0; layout < 4; layout++ {
		m.controls.layout = layout
		m.configVisible = false
		m.resize()
		if m.controls.rects[2].w != 0 {
			t.Fatal("hidden geometry")
		}
		m.panelsView()
	}
}

func TestTUIConfigShortcut(t *testing.T) {
	for _, focus := range []int{0, 1, configButton, helpButton} {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40}
		m.setFocus(1)
		m.setFocus(focus)
		previous := m.lastWorkspace
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
		if !m.configVisible || m.focus != 2 {
			t.Fatal("Ctrl-G did not open config")
		}
		m.controls.dropdown = true
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.controls.dropdown || !m.configVisible {
			t.Fatal("Escape did not close dropdown first")
		}
		m.showHelp()
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.helpVisible || m.configVisible {
			t.Fatal("Escape did not close help pane")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.configVisible || m.focus != previous {
			t.Fatal("Escape did not restore workspace")
		}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd != nil {
			t.Fatal("Escape in workspace should do nothing")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
		m.controls.dropdown = true
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
		if m.configVisible || m.controls.dropdown {
			t.Fatal("Ctrl-G did not close config")
		}
	}
}

func TestTUICommandClipboard(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var copied string
		m := &tuiModel{width: 100, height: 30, copyClipboard: func(s string) error {
			copied = s
			if fail {
				return errors.New("clipboard unavailable")
			}
			return nil
		}}
		command := "go-yaml -y < .go-yaml/input.yaml"
		result := m.showCommand(command)()
		if copied != command {
			t.Fatalf("copied %q", copied)
		}
		m.Update(result)
		title := "Run as CLI. Cmd copied to clipboard."
		if fail {
			title = "Run as CLI. Clipboard copy failed."
		}
		if m.modalTitle != title {
			t.Fatal(m.modalTitle)
		}
		if !strings.Contains(m.View(), title) || !strings.Contains(m.View(), "╭") {
			t.Fatal("missing titled box")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m.Update(result)
		if m.modal != "" {
			t.Fatal("late clipboard result reopened modal")
		}
	}
}

func TestTUIReadableCommand(t *testing.T) {
	for _, tc := range []struct {
		binary string
		args   []string
		want   string
	}{
		{"/work/go-yaml", []string{"-E"}, "./go-yaml -E .go-yaml/input.yaml"},
		{"/bin/go-yaml", []string{"-y", "-o", "indent=4"}, "../bin/go-yaml -y -o indent=4 .go-yaml/input.yaml"},
		{"/work/my tools/go-yaml", []string{"-y", "--type", "map[string]any"}, "'./my tools/go-yaml' -y --type 'map[string]any' .go-yaml/input.yaml"},
	} {
		got, err := tuiCLICommand(tc.binary, "/work", tc.args)
		if err != nil || got != tc.want {
			t.Fatalf("got %q, %v; want %q", got, err, tc.want)
		}
	}
	for _, tc := range []struct{ value, want string }{
		{"", "''"},
		{"a b", "'a b'"},
		{"$(id)", "'$(id)'"},
		{"a;b", "'a;b'"},
		{"a\nb", "'a\nb'"},
		{"-E", "-E"},
	} {
		if got := shellQuote(tc.value); got != tc.want {
			t.Fatalf("quote %q: %q", tc.value, got)
		}
	}
}

func TestTUISelfDocumentingSession(t *testing.T) {
	s := tuiSession{Version: 1, Layout: 1, Live: true, Args: []string{"-E", "-o", "unicode,no-unique-keys", "--type", "any"}}
	data, err := encodeTUISession(s)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "### go-yaml session configuration\n#\n# Boolean values:\n") {
		t.Fatal("unexpected header", text)
	}
	if !strings.Contains(text, "#   true  → enabled   `+`\n#   false → disabled  `-`\n#   blank → default\n###\n\nlive: true\n\nlayout: stacked\n# Layout choices:\n") {
		t.Fatal("unexpected legend or field order", text)
	}
	if strings.Contains(text, "version:") {
		t.Fatal("version should not be written")
	}
	if !strings.Contains(text, "#   focus\n\noutput: events-detailed\n# Output choices:") {
		t.Fatal("unexpected layout spacing", text)
	}
	for _, name := range tuiOutputNames {
		if !strings.Contains(text, "#   "+name) {
			t.Fatal("missing output", name)
		}
	}
	for _, name := range tuiLayoutNames() {
		if !strings.Contains(text, "#   "+name) {
			t.Fatal("missing layout", name)
		}
	}
	for _, name := range tuiBooleans {
		if !strings.Contains(text, "  "+name+":") {
			t.Fatal("missing option", name)
		}
	}
	for _, want := range []string{"layout: stacked", "output: events-detailed", "  unicode: true\n", "  unique-keys: false\n", "  canonical:\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	loaded, err := decodeTUISession(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := encodeTUISession(loaded)
	if err != nil || string(again) != text {
		t.Fatalf("round trip changed config: %v\n%s", err, again)
	}
	for _, replacement := range []string{"layout: bad", "layout: [columns]"} {
		_, err := decodeTUISession([]byte(strings.Replace(text, "layout: stacked", replacement, 1)))
		if err == nil {
			t.Fatal("accepted invalid layout")
		}
	}
	_, err = decodeTUISession([]byte(strings.Replace(text, "unicode: true", "unicode: nonsense", 1)))
	if err == nil {
		t.Fatal("accepted invalid boolean")
	}
}

func TestTUIOptionChangesRunWithoutLive(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	}()
	newModel := func() *tuiModel {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, configVisible: true, focus: 2}
		m.config.CharLimit = 0
		m.config.SetValue("version: 1\nargs: [-y]\nlive: false\n")
		m.resize()
		return m
	}
	for _, kind := range []string{"keyboard", "mouse", "dropdown", "pane keyboard", "pane mouse", "vim"} {
		m := newModel()
		m.controls.selected = 3
		var cmd tea.Cmd
		switch kind {
		case "keyboard":
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		case "mouse":
			r := m.controls.rects[2]
			_, cmd = m.Update(tea.MouseMsg{X: r.x + 4, Y: r.y + 7, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		case "dropdown":
			m.controls.selected = 1
			m.controls.dropdown = true
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		case "pane keyboard":
			m.configVisible = false
			m.setFocus(outputButton)
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.Update(tea.KeyMsg{Type: tea.KeyDown})
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if m.configVisible || m.controls.mode != 1 || m.outputMenu {
				t.Fatal("pane selector failed")
			}
		case "pane mouse":
			m.configVisible = false
			m.resize()
			selector := m.outputSelectorRect()
			m.Update(tea.MouseMsg{X: selector.x + 1, Y: selector.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			menu := m.outputMenuRect()
			_, cmd = m.Update(tea.MouseMsg{X: menu.x + 1, Y: menu.y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if m.controls.mode != 1 {
				t.Fatal("mouse output selection failed")
			}
		case "vim":
			if err := m.save(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(".go-yaml/session.yaml", []byte("version: 1\nargs: [-E]\nlive: false\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, cmd = m.Update(tuiEditorDone{panel: 2})
		}
		if cmd == nil || m.status != "Running" {
			t.Fatalf("%s did not run: %s", kind, m.status)
		}
		m.cancel()
		old := m.revision
		m.configVisible = true
		m.setFocus(2)
		m.controls.dropdown = false
		m.controls.selected = 4
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil || m.revision <= old {
			t.Fatal("next option did not invalidate old run")
		}
		m.Update(tuiResult{revision: old, output: "stale"})
		if m.status != "Running" {
			t.Fatal("stale result accepted")
		}
		m.cancel()
	}
	m := newModel()
	m.controls.selected = 0
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("layout triggered processing")
	}
}

func TestTUIConfigPanelLayout(t *testing.T) {
	m := &tuiModel{config: textarea.New()}
	m.config.SetValue("version: 1\nargs: [-y]\nlive: true\n")
	view := m.controlView()
	if !strings.HasPrefix(view, "› Layout: Columns ▾\n  Output: YAML ▾\n\n  Options:\n  [+] Live run\n") {
		t.Fatal(view)
	}
	if !strings.HasSuffix(view, "[+] true  [-] false [ ] default\nEnter/Space: change\nCtrl-E: Edit config in Vim\nEsc: Close config panel") {
		t.Fatal(view)
	}
	for row, want := range []int{0, 1, -1, -1, 2, 3} {
		if got := tuiControlAtRow(row); got != want {
			t.Fatalf("row %d: got %d, want %d", row, got, want)
		}
	}
	for _, row := range []int{-1, 4 + 1 + len(tuiBooleans), 30} {
		if tuiControlAtRow(row) != -1 {
			t.Fatalf("non-control row %d selectable", row)
		}
	}
	m.controls.selected = 3
	if !strings.Contains(m.controlView(), "› [ ] compact-seq-indent") {
		t.Fatal("selection moved to wrong row")
	}
}

func TestTUIInterruptConfirmation(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	}()
	for _, state := range []string{"workspace", "modal", "output menu", "config"} {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40}
		m.config.CharLimit = 0
		m.config.SetValue("version: 1\nargs: [-y]\nlive: false\n")
		m.resize()
		switch state {
		case "modal":
			m.modal = "dialog"
		case "output menu":
			m.openOutputMenu()
		case "config":
			m.toggleConfig()
		}
		canceled := false
		m.cancel = func() { canceled = true }
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd != nil || !m.quitPending || canceled {
			t.Fatalf("%s: first Ctrl-C quit", state)
		}
		if !strings.Contains(m.View(), "Press Ctrl-C again to quit") {
			t.Fatal("prompt not visible")
		}
		m.Update(tuiResult{revision: m.revision, output: "done"})
		if !m.quitPending {
			t.Fatal("background result canceled prompt")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if m.quitPending {
			t.Fatal("Escape did not cancel")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m.Update(tea.MouseMsg{Action: tea.MouseActionRelease})
		if m.quitPending {
			t.Fatal("mouse did not cancel")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil || !canceled {
			t.Fatal("second Ctrl-C did not quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("missing quit message")
		}
		if _, err := os.Stat(".go-yaml/session.yaml"); err != nil {
			t.Fatal("session not saved", err)
		}
	}
}

func testNavX(m *tuiModel, id int) int {
	for _, item := range m.navItems() {
		if item.id == id {
			return item.x + 2
		}
	}
	return -1
}

func TestTUINavigationOrderAndTargets(t *testing.T) {
	for mode := range tuiModes {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40}
		m.controls.mode = mode
		for _, active := range []int{0, configButton, helpButton} {
			m.configVisible = active == configButton
			m.helpVisible = active == helpButton
			items := m.navItems()
			if len(items) != 2 || items[0].id != configButton || items[1].id != helpButton {
				t.Fatal("wrong order")
			}
			for i := 1; i < len(items); i++ {
				if items[i].x != items[i-1].x+items[i-1].width+2 {
					t.Fatal("overlapping nav targets")
				}
			}
			m.outputMenu = false
			m.resize()
			selector := m.outputSelectorRect()
			m.Update(tea.MouseMsg{X: selector.x + 1, Y: selector.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if !m.outputMenu {
				t.Fatalf("output target failed for mode %d", mode)
			}
		}
	}
}

func TestTUIAuxiliaryPanePlacement(t *testing.T) {
	for _, help := range []bool{false, true} {
		for layout := 0; layout < 3; layout++ {
			m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, configVisible: !help, helpVisible: help}
			m.controls.layout = layout
			m.resize()
			input, output, side := m.controls.rects[0], m.controls.rects[1], m.controls.rects[2]
			if layout < 2 {
				if side.x != 0 || side.x+side.w != input.x || output.x < side.w {
					t.Fatal("side pane not left of workspace")
				}
			} else if side.y != 1 || side.y+side.h != input.y || input.y+input.h != output.y {
				t.Fatal("side pane not above workspace")
			}
			view := m.panelsView()
			title := "CONFIG"
			if help {
				title = "HELP"
			}
			if strings.Index(view, title) > strings.Index(view, "INPUT") {
				t.Fatal("render order differs from geometry")
			}
		}
	}
}

func TestTUIResetConfig(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	}()
	for _, mouse := range []bool{false, true} {
		m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, configVisible: true, focus: 2}
		m.config.CharLimit = 0
		m.input.SetValue("keep: this\n")
		m.config.SetValue("version: 1\nlayout: 1\nsplit: 0.5\nlive: false\nargs: [-E, -o, unicode, -o, no-unique-keys]\n")
		m.syncControls()
		m.resize()
		m.controls.selected = tuiResetIndex()
		var cmd tea.Cmd
		if mouse {
			r := m.controls.rects[2]
			_, cmd = m.Update(tea.MouseMsg{X: r.x + 4, Y: r.y + 2 + tuiResetIndex() + 3, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		} else {
			_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		}
		if cmd == nil {
			t.Fatal("reset did not rerun")
		}
		m.cancel()
		s, err := m.session()
		if err != nil {
			t.Fatal(err)
		}
		if s.Layout != 0 || s.Split != 0 || !s.Live || !reflect.DeepEqual(s.Args, []string{"-y"}) {
			t.Fatalf("reset failed: %+v", s)
		}
		if m.input.Value() != "keep: this\n" {
			t.Fatal("reset changed input")
		}
		saved, err := os.ReadFile(".go-yaml/session.yaml")
		if err != nil {
			t.Fatal(err)
		}
		restored, err := decodeTUISession(saved)
		if err != nil || !reflect.DeepEqual(s, restored) {
			t.Fatal("reset not persisted")
		}
		if !strings.Contains(m.controlView(), "› [Reset]") {
			t.Fatal("reset focus missing")
		}
	}
}

func TestTUIClearInput(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	}()
	for layout := 0; layout < 4; layout++ {
		for _, live := range []bool{false, true} {
			m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, configVisible: true}
			m.config.CharLimit = 0
			data, err := encodeTUISession(tuiSession{Version: 1, Layout: layout, Live: live, Args: []string{"-E"}})
			if err != nil {
				t.Fatal(err)
			}
			m.config.SetValue(string(data))
			m.input.SetValue("foo: 1\n")
			m.output.SetContent("old result")
			m.syncControls()
			m.resize()
			m.setFocus(0)
			savedConfig := m.config.Value()
			rect := m.clearButtonRect()
			if !strings.Contains(m.panelsView(), "[Clear]") {
				t.Fatal("clear button not rendered")
			}
			canceled := false
			m.cancel = func() { canceled = true }
			oldRevision := m.revision
			_, cmd := m.Update(tea.MouseMsg{X: rect.x + 2, Y: rect.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if m.input.Value() != "" || m.config.Value() != savedConfig || m.focus != 0 {
				t.Fatal("clear changed wrong state")
			}
			if !canceled || m.revision <= oldRevision {
				t.Fatal("old output was not invalidated")
			}
			if (cmd != nil) != live {
				t.Fatal("live-run setting ignored")
			}
			if m.cancel != nil {
				m.cancel()
			}
			data, err = os.ReadFile(".go-yaml/input.yaml")
			if err != nil || len(data) != 0 {
				t.Fatal("cleared input not saved")
			}
			if layout == 3 {
				m.setFocus(1)
				if m.clearButtonClicked(tea.MouseMsg{X: rect.x, Y: rect.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}) {
					t.Fatal("hidden input is clickable")
				}
			}
		}
	}
}

func TestTUIOutputSelectorPlacement(t *testing.T) {
	for layout := 0; layout < 4; layout++ {
		for _, sidebar := range []bool{false, true} {
			m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), width: 120, height: 40, configVisible: sidebar}
			m.controls.layout = layout
			m.resize()
			m.setFocus(1)
			if strings.Contains(m.toolbar(), "Output:") {
				t.Fatal("selector remains in navbar")
			}
			if !strings.Contains(m.panelsView(), m.outputSelectorLabel()) {
				t.Fatal("selector missing from output pane")
			}
			selector := m.outputSelectorRect()
			m.Update(tea.MouseMsg{X: selector.x + 1, Y: selector.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if !m.outputMenu || m.focus != outputButton {
				t.Fatal("pane selector did not open")
			}
			menu := m.outputMenuRect()
			if menu.x < 0 || menu.x+menu.w > m.width || menu.y < 1 || menu.y+menu.h > m.height-4 {
				t.Fatal("dropdown outside workspace")
			}
			m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if m.outputMenu || m.lastWorkspace != 1 {
				t.Fatal("dropdown did not restore output")
			}
			if layout == 3 {
				m.setFocus(0)
				if m.outputSelectorClicked(tea.MouseMsg{X: selector.x + 1, Y: selector.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}) {
					t.Fatal("hidden selector clickable")
				}
			}
		}
	}
}
