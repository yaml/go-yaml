package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tuiSession struct {
	Split   float64  `yaml:"split,omitempty"`
	Layout  int      `yaml:"layout,omitempty"`
	Version int      `yaml:"version"`
	Args    []string `yaml:"args"`
	Live    bool     `yaml:"live"`
}
type (
	tuiTick   int
	tuiResult struct {
		revision int
		output   string
	}
)

type tuiEditorDone struct {
	err   error
	panel int
}
type tuiModel struct {
	quitPending                    bool
	helpVisible                    bool
	outputMenu                     bool
	outputChoice                   int
	modalTitle                     string
	commandGeneration              int
	copyClipboard                  func(string) error
	configVisible                  bool
	lastWorkspace                  int
	controls                       tuiControls
	input, config                  textarea.Model
	output                         viewport.Model
	focus, width, height, revision int
	status, modal                  string
	live                           bool
	cancel                         context.CancelFunc
}

func atomicTUIFile(path string, data []byte) error {
	f, err := os.CreateTemp(".go-yaml", ".save-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (m *tuiModel) session() (tuiSession, error) {
	s, err := decodeTUISession([]byte(m.config.Value()))
	if err != nil {
		return s, err
	}
	if s.Version != 1 {
		return s, errors.New("session version must be 1")
	}
	for _, a := range s.Args {
		if strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-") == "tui" {
			return s, errors.New("session args cannot include --tui")
		}
	}
	if len(s.Args) == 0 {
		s.Args = []string{"-y"}
	}
	return s, nil
}

func (m *tuiModel) save() error {
	if err := os.MkdirAll(".go-yaml", 0o700); err != nil {
		return err
	}
	if err := atomicTUIFile(".go-yaml/.gitignore", []byte("*\n")); err != nil {
		return err
	}
	if err := atomicTUIFile(".go-yaml/input.yaml", []byte(m.input.Value())); err != nil {
		return err
	}
	s, err := m.session()
	if err != nil {
		return err
	}
	data, err := encodeTUISession(s)
	if err != nil {
		return err
	}
	return atomicTUIFile(".go-yaml/session.yaml", data)
}

func (m *tuiModel) run() tea.Cmd {
	s, err := m.session()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if err = m.save(); err != nil {
		m.status = err.Error()
		return nil
	}
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.revision++
	revision, input := m.revision, m.input.Value()
	m.status = "Running"
	return func() tea.Msg {
		binary, err := os.Executable()
		if err != nil {
			return tuiResult{revision, err.Error()}
		}
		args := tuiRunArgs(s.Args)
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Stdin = strings.NewReader(input)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		result := out.String()
		if err != nil || stderr.Len() > 0 {
			result += "\nErrors:\n" + stderr.String()
			if stderr.Len() == 0 {
				result += fmt.Sprint(err)
			}
		}
		return tuiResult{revision, result}
	}
}
func (m *tuiModel) Init() tea.Cmd { return tea.Batch(textarea.Blink, m.run()) }
func (m *tuiModel) resize() {
	m.geometry()
	r := m.controls.rects[0]
	m.input.SetWidth(maxTUI(10, r.w-2))
	m.input.SetHeight(maxTUI(3, r.h-3))
	r = m.controls.rects[1]
	m.output.Width = maxTUI(10, r.w-2)
	m.output.Height = maxTUI(3, r.h-3)
}

func maxTUI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *tuiModel) Update(msg tea.Msg) (model tea.Model, command tea.Cmd) {
	beforeSession, beforeErr := m.session()
	defer func() {
		after, err := m.session()
		if err == nil && (beforeErr != nil || !reflect.DeepEqual(beforeSession.Args, after.Args) || !beforeSession.Live && after.Live) {
			command = m.run()
		}
	}()
	switch v := msg.(type) {
	case tuiCopyResult:
		if v.generation == m.commandGeneration && m.modal != "" && m.modalTitle != "" {
			if v.err == nil {
				m.modalTitle = "Run as CLI. Cmd copied to clipboard."
			} else {
				m.modalTitle = "Run as CLI. Clipboard copy failed."
				m.status = v.err.Error()
			}
		}
		return m, nil
	case tea.MouseMsg:
		m.quitPending = false
		if m.outputMenu {
			return m, m.outputMenuMouse(v)
		}
		if m.toolbarMouse(v) {
			return m, nil
		}
		if m.modal == "" {
			if m.outputSelectorClicked(v) {
				m.openOutputMenu()
				return m, nil
			}
			if m.clearButtonClicked(v) {
				return m, m.clearInput()
			}
			m.mouse(v)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.resize()
	case tuiResult:
		if v.revision == m.revision {
			m.output.SetContent(v.output)
			m.status = "Ready"
		}
		return m, nil
	case tuiTick:
		if int(v) == m.revision {
			if err := m.save(); err != nil {
				m.status = err.Error()
				return m, nil
			}
			s, err := m.session()
			if err == nil && s.Live {
				return m, m.run()
			}
		}
		return m, nil
	case tuiEditorDone:
		if v.err != nil {
			m.status = v.err.Error()
			return m, nil
		}
		path := ".go-yaml/input.yaml"
		if v.panel == 2 {
			path = ".go-yaml/session.yaml"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		if v.panel == 2 {
			m.config.SetValue(string(data))
		} else {
			m.input.SetValue(string(data))
		}
		m.revision++
		s, err := m.session()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.syncControls()
		m.resize()
		if s.Live && v.panel != 2 {
			return m, m.run()
		}
		return m, nil
	case tea.KeyMsg:
		key := v.String()
		if key == "ctrl+c" && !m.quitPending {
			m.quitPending = true
			return m, nil
		}
		m.quitPending = false
		if key == tuiQuitKey || key == "ctrl+c" {
			if err := m.save(); err != nil {
				m.status = err.Error()
				return m, nil
			}
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if m.modal != "" {
			if key == "esc" {
				m.modal = ""
			}
			return m, nil
		}
		if m.outputMenu {
			return m, m.outputMenuKey(key)
		}
		if key == "esc" && m.helpVisible {
			m.showHelp()
			return m, nil
		}
		if key == tuiConfigKey {
			return m, m.toggleConfig()
		}
		if key == "esc" && m.configVisible && !m.controls.dropdown {
			return m, m.toggleConfig()
		}
		if m.focus >= configButton {
			if key == "enter" || key == " " {
				if m.focus == configButton {
					return m, m.toggleConfig()
				}
				if m.focus == outputButton {
					m.openOutputMenu()
				} else {
					m.showHelp()
				}
				return m, nil
			}
		}
		if handled, cmd := m.controlsKey(key); handled {
			return m, cmd
		}
		switch key {
		case tuiCommandKey:
			s, err := m.session()
			if err != nil {
				m.status = err.Error()
				return m, nil
			}
			if err = m.save(); err != nil {
				m.status = err.Error()
				return m, nil
			}
			binary, err := os.Executable()
			if err != nil {
				m.status = err.Error()
				return m, nil
			}
			cwd, err := os.Getwd()
			if err != nil {
				m.status = err.Error()
				return m, nil
			}
			command, err := tuiCLICommand(binary, cwd, tuiRunArgs(s.Args))
			if err != nil {
				m.status = err.Error()
				return m, nil
			}
			return m, m.showCommand(command)
		case tuiRunKey:
			return m, m.run()
		case "tab", "shift+tab":
			return m, m.navigate(key == "shift+tab")
		case tuiEditKey:
			if m.focus == 1 || m.focus >= configButton || m.helpVisible && m.focus == 2 {
				return m, nil
			}
			_, configErr := m.session()
			if configErr == nil || m.focus != 2 {
				if err := m.save(); err != nil {
					m.status = err.Error()
					return m, nil
				}
			}
			if m.focus == 2 {
				if _, err := exec.LookPath("vim"); err != nil {
					m.status = err.Error()
					return m, nil
				}
				return m, tea.ExecProcess(exec.Command("vim", ".go-yaml/session.yaml"), func(err error) tea.Msg { return tuiEditorDone{err, 2} })
			}
			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				editor = "vi"
			}
			path := ".go-yaml/input.yaml"
			if m.focus == 2 {
				path = ".go-yaml/session.yaml"
			}
			panel := m.focus
			return m, tea.ExecProcess(exec.Command("sh", "-c", editor+" \"$1\"", "editor", path), func(err error) tea.Msg { return tuiEditorDone{err, panel} })
		}
	}
	before := m.input.Value() + m.config.Value()
	var cmd tea.Cmd
	switch m.focus {
	case 0:
		m.input, cmd = m.input.Update(msg)
	case 1:
		m.output, cmd = m.output.Update(msg)
	case 2:
		if !m.helpVisible {
			m.config, cmd = m.config.Update(msg)
		}
	}
	if before != m.input.Value()+m.config.Value() {
		m.revision++
		if m.cancel != nil {
			m.cancel()
		}
		m.status = "Edited; output stale"
		revision := m.revision
		return m, tea.Batch(cmd, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return tuiTick(revision) }))
	}
	return m, cmd
}

func (m *tuiModel) View() (view string) {
	defer func() {
		if m.quitPending {
			prompt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render("Press Ctrl-C again to quit; any other key cancels.")
			_, rest, found := strings.Cut(view, "\n")
			view = prompt
			if found {
				view += "\n" + rest
			}
		}
	}()
	if m.modal != "" {
		if m.modalTitle != "" {
			return m.commandView()
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(m.modal)
	}
	if m.outputMenu {
		return m.toolbar() + "\n" + m.outputMenuView()
	}
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(m.status)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Use Mouse or Tab key to navigate.")
	return m.toolbar() + "\n" + m.panelsView() + "\n" + status + "\n" + hint
}

func startTUI(args []string) error {
	m := &tuiModel{input: textarea.New(), config: textarea.New(), output: viewport.New(30, 20), status: "Ready"}
	m.input.CharLimit = 0
	m.config.CharLimit = 0
	m.input.SetValue("hello: world\nitems:\n- YAML\n- Go\n")
	s := tuiSession{Version: 1, Args: []string{"-y"}, Live: true}
	if data, err := os.ReadFile(".go-yaml/input.yaml"); err == nil {
		m.input.SetValue(string(data))
	}
	data, _ := encodeTUISession(s)
	m.config.SetValue(string(data))
	if data, err := os.ReadFile(".go-yaml/session.yaml"); err == nil {
		m.config.SetValue(string(data))
	}
	if len(args) > 0 {
		s.Args = args
		data, _ = encodeTUISession(s)
		m.config.SetValue(string(data))
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		m.input.SetValue(string(data))
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("TUI requires a controlling terminal: %w", err)
	}
	defer tty.Close()
	m.syncControls()
	m.input.Focus()
	_, err = tea.NewProgram(m, tea.WithInput(tty), tea.WithOutput(tty), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func tuiRunArgs(args []string) []string {
	for _, arg := range args {
		switch arg {
		case "-y", "-Y", "-j", "-J", "-t", "-T", "-e", "-E", "-n", "-N", "--yaml", "--YAML", "--json", "--JSON", "--token", "--TOKEN", "--event", "--EVENT", "--node", "--NODE":
			return args
		}
	}
	return append([]string{"-y"}, args...)
}
