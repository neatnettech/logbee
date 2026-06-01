package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ringBuffer holds the last max lines of output for a tab, dropping the oldest
// once full so memory stays bounded regardless of how chatty a process is.
type ringBuffer struct {
	lines []string
	max   int
}

func newRingBuffer(max int) *ringBuffer {
	if max < 1 {
		max = 1
	}
	return &ringBuffer{lines: make([]string, 0, max), max: max}
}

func (r *ringBuffer) push(s string) {
	if len(r.lines) >= r.max {
		// drop oldest; shift in place
		copy(r.lines, r.lines[1:])
		r.lines[len(r.lines)-1] = s
		return
	}
	r.lines = append(r.lines, s)
}

func (r *ringBuffer) joined() string {
	return strings.Join(r.lines, "\n")
}

// logTab is one tab: either a single profile or the aggregate ("all") view.
type logTab struct {
	name   string
	color  int
	proc   *process // nil for the aggregate tab
	ring   *ringBuffer
	vp     viewport.Model
	follow bool
	exited bool
	// passthrough: keystrokes on this tab go straight to the process (live
	// mode), so you can drive an interactive dev server (e.g. Expo: r/i/a/j/m).
	passthrough bool
}

type tuiModel struct {
	tabs   []*logTab
	active int
	width  int
	height int
	// leader is true after the leader key (ctrl+b) is pressed on a live
	// (passthrough) tab; the next key is interpreted as a logbee command
	// instead of being forwarded to the process.
	leader bool
	// help is true while the full key-map overlay is shown.
	help  bool
	ready bool
	out   *multiOutput
}

// procExitMsg notifies the TUI that a process has exited so its tab can be
// marked.
type procExitMsg struct{ proc *process }

func newTUIModel(out *multiOutput, procs []*process, scrollback int) *tuiModel {
	tabs := make([]*logTab, 0, len(procs)+1)

	// Tab 0: aggregate of every process, with colored prefixes.
	tabs = append(tabs, &logTab{
		name:   "all",
		color:  7,
		proc:   nil,
		ring:   newRingBuffer(scrollback),
		follow: true,
	})

	for _, p := range procs {
		tabs = append(tabs, &logTab{
			name:   p.Name,
			color:  p.Color,
			proc:   p,
			ring:   newRingBuffer(scrollback),
			follow: true,
		})
	}

	active := 0
	// If an interactive process was named, focus its tab by default.
	if out.interactiveProc != nil {
		for i, t := range tabs {
			if t.proc == out.interactiveProc {
				active = i
				t.passthrough = true
				break
			}
		}
	}

	return &tuiModel{tabs: tabs, active: active, out: out}
}

func (m *tuiModel) Init() tea.Cmd { return nil }

func (m *tuiModel) tabForProc(p *process) *logTab {
	for _, t := range m.tabs {
		if t.proc == p {
			return t
		}
	}
	return nil
}

func (m *tuiModel) refresh(t *logTab) {
	t.vp.SetContent(t.ring.joined())
	if t.follow {
		t.vp.GotoBottom()
	}
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - 2 // tab bar + status line
		if vpHeight < 1 {
			vpHeight = 1
		}
		for _, t := range m.tabs {
			t.vp.Width = msg.Width
			t.vp.Height = vpHeight
			m.refresh(t)
		}
		m.ready = true
		return m, nil

	case logLineMsg:
		// Per-profile tab gets the raw line.
		if t := m.tabForProc(msg.proc); t != nil {
			t.ring.push(msg.raw)
			m.refresh(t)
		}
		// Aggregate tab gets the prefixed line.
		agg := m.tabs[0]
		agg.ring.push(msg.prefixed)
		m.refresh(agg)
		return m, nil

	case procExitMsg:
		if t := m.tabForProc(msg.proc); t != nil {
			t.exited = true
		}
		return m, nil

	case tea.KeyMsg:
		// Help overlay swallows all keys; any key dismisses it.
		if m.help {
			m.help = false
			return m, nil
		}
		// Leader pressed last: this key is a logbee command.
		if m.leader {
			m.leader = false
			return m.handleLeaderKey(msg)
		}
		// On a live tab, the leader key arms command mode; everything else is
		// forwarded to the process.
		if m.tabs[m.active].passthrough {
			if msg.String() == leaderKey {
				m.leader = true
				return m, nil
			}
			return m.forwardKey(msg)
		}
		return m.handleNavKey(msg)
	}

	return m, nil
}

// leaderKey arms command mode on a live (passthrough) tab. ctrl+b is tmux's
// prefix and is not used by Expo/Metro, so it won't collide with the dev server.
const leaderKey = "ctrl+b"

// forwardKey sends a keystroke straight to the active tab's process.
func (m *tuiModel) forwardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if t := m.tabs[m.active]; t.proc != nil {
		if b := keyToBytes(msg); len(b) > 0 {
			m.out.WriteStdin(t.proc, b)
		}
	}
	return m, nil
}

// handleLeaderKey runs a logbee command after the leader was pressed on a live
// tab (e.g. ctrl+b then 0 jumps to the aggregate tab, ctrl+b q quits).
func (m *tuiModel) handleLeaderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case leaderKey:
		// Leader twice: send a literal leader byte (0x02) to the process.
		return m.forwardKey(msg)
	case "i":
		m.tabs[m.active].passthrough = false
		return m, nil
	case "?":
		m.help = true
		return m, nil
	default:
		return m.handleNavKey(msg)
	}
}

func (m *tuiModel) handleNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.help = true
		return m, nil
	case "tab", "right", "l":
		m.active = (m.active + 1) % len(m.tabs)
		return m, nil
	case "shift+tab", "left", "h":
		m.active = (m.active - 1 + len(m.tabs)) % len(m.tabs)
		return m, nil
	case "f":
		t := m.tabs[m.active]
		t.follow = !t.follow
		if t.follow {
			t.vp.GotoBottom()
		}
		return m, nil
	case "i", "enter":
		// Go live: keystrokes now forward to this process (Expo etc.).
		if m.tabs[m.active].proc != nil {
			m.tabs[m.active].passthrough = true
		}
		return m, nil
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		n, _ := strconv.Atoi(msg.String())
		if n < len(m.tabs) {
			m.active = n
		}
		return m, nil
	case "g", "home":
		t := m.tabs[m.active]
		t.vp.GotoTop()
		t.follow = false
		return m, nil
	case "G", "end":
		t := m.tabs[m.active]
		t.vp.GotoBottom()
		t.follow = true
		return m, nil
	}

	// Delegate remaining scrolling keys (up/down/pgup/pgdn) to the viewport.
	var cmd tea.Cmd
	t := m.tabs[m.active]
	t.vp, cmd = t.vp.Update(msg)
	// Manual scroll disables follow; re-enable when scrolled back to bottom.
	t.follow = t.vp.AtBottom()
	return m, cmd
}

func (m *tuiModel) View() string {
	if !m.ready {
		return "loading..."
	}
	if m.help {
		return m.renderHelp()
	}
	// Layout: log viewport on top, then the tab bar and status line pinned to
	// the bottom of the console.
	return strings.Join([]string{
		m.tabs[m.active].vp.View(),
		m.renderTabBar(),
		m.renderStatus(),
	}, "\n")
}

// renderHelp draws the full key-map overlay, sized to the console.
func (m *tuiModel) renderHelp() string {
	rows := [][2]string{
		{"Tabs", ""},
		{"  ←/→  ·  Tab/Shift+Tab", "previous / next tab"},
		{"  0-9", "jump to tab by index (0 = aggregate \"all\")"},
		{"", ""},
		{"Scroll", ""},
		{"  ↑/↓  ·  PgUp/PgDn", "scroll the active tab"},
		{"  Home/End  ·  g/G", "jump to top / bottom"},
		{"  f", "toggle follow (auto-scroll to newest)"},
		{"", ""},
		{"Live mode (drive a process: Expo r/i/a/j/m …)", ""},
		{"  i  ·  Enter", "go live on the focused process tab"},
		{"  <keys>", "forwarded straight to the process while live"},
		{"  Ctrl+C", "forwarded to the process while live"},
		{"", ""},
		{"Leader (Ctrl+B, while live — then:)", ""},
		{"  Ctrl+B 0-9", "jump to a tab"},
		{"  Ctrl+B i", "stop live (back to scroll/nav)"},
		{"  Ctrl+B q", "quit logbee"},
		{"  Ctrl+B ?", "this help"},
		{"  Ctrl+B Ctrl+B", "send a literal Ctrl+B to the process"},
		{"", ""},
		{"General", ""},
		{"  ?", "toggle this help"},
		{"  q  ·  Ctrl+C", "quit (when not live)"},
	}

	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render("logbee --tui · key map")
	b.WriteString(title + "\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	hdrStyle := lipgloss.NewStyle().Bold(true)
	for _, r := range rows {
		if r[1] == "" {
			if r[0] == "" {
				b.WriteString("\n")
			} else {
				b.WriteString(hdrStyle.Render(r[0]) + "\n")
			}
			continue
		}
		b.WriteString(keyStyle.Render(fmt.Sprintf("%-26s", r[0])) + r[1] + "\n")
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("press any key to close"))

	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(b.String())
}

func (m *tuiModel) renderTabBar() string {
	var parts []string
	for i, t := range m.tabs {
		mark := ""
		if t.exited {
			mark = "*"
		} else if t.passthrough {
			mark = "●"
		}
		label := fmt.Sprintf(" %d:%s%s ", i, t.name, mark)

		style := lipgloss.NewStyle().Foreground(lipgloss.Color(strconv.Itoa(t.color)))
		if i == m.active {
			style = style.Bold(true).Reverse(true)
		}
		parts = append(parts, style.Render(label))
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.NewStyle().Width(m.width).Render(bar)
}

func (m *tuiModel) renderStatus() string {
	t := m.tabs[m.active]

	if m.leader {
		s := fmt.Sprintf(" ctrl+b … %s · 0-9 jump · q quit · i stop-live ", t.name)
		return lipgloss.NewStyle().Width(m.width).Reverse(true).Render(s)
	}

	if t.passthrough {
		s := fmt.Sprintf(" ● LIVE → %s  keys go to process · ctrl+b: menu ", t.name)
		return lipgloss.NewStyle().Width(m.width).Reverse(true).Render(s)
	}

	follow := "follow:off"
	if t.follow {
		follow = "follow:on"
	}
	live := ""
	if t.proc != nil {
		live = " · i:live"
	}
	s := fmt.Sprintf(" ←/→ tabs · 0-9 jump · ↑/↓ scroll · f:%s%s · q:quit ", follow, live)
	return lipgloss.NewStyle().Width(m.width).Faint(true).Render(s)
}

// keyToBytes converts a key press into the bytes a process expects on its
// stdin, so the TUI can drive interactive dev servers.
func keyToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyRunes:
		return []byte(string(msg.Runes))
	case tea.KeySpace:
		return []byte(" ")
	case tea.KeyEnter:
		return []byte("\r")
	case tea.KeyTab:
		return []byte("\t")
	case tea.KeyBackspace:
		return []byte("\x7f")
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	default:
		// Control chars: ctrl+<letter> → byte 1..26 (e.g. ctrl+c → 0x03), so a
		// dev server in a focused tab can be interrupted/EOF'd.
		s := msg.String()
		if strings.HasPrefix(s, "ctrl+") && len(s) == 6 {
			c := s[5]
			if c >= 'a' && c <= 'z' {
				return []byte{c - 'a' + 1}
			}
		}
		if len(msg.Runes) > 0 {
			return []byte(string(msg.Runes))
		}
		return nil
	}
}
