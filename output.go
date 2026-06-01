package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/term/termios"
)

// ansiEscape matches ANSI escape sequences (colors, cursor moves, etc.) so
// they can be stripped from the plain-text log file copy. Compiled once.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripANSI(p []byte) []byte {
	return ansiEscape.ReplaceAll(p, nil)
}

type ptyPipe struct {
	pty, tty *os.File
}

type multiOutput struct {
	maxNameLength   int
	mutex           sync.Mutex
	pipes           map[*process]*ptyPipe
	printProcName   bool
	printTimestamp  bool
	logFile         *os.File
	interactiveProc *process
	eol             string
	program         *tea.Program
}

// logLineMsg carries one output line to the TUI. raw is the unprefixed line
// (shown in the profile's own tab); prefixed includes the colored
// timestamp/name prefix (shown in the aggregate tab), matching plain-mode
// formatting exactly.
type logLineMsg struct {
	proc     *process
	raw      string
	prefixed string
}

// OpenLogFile opens path for the aggregated log stream, creating parent
// directories as needed. By default the file is truncated; when appendMode is
// true it is opened with O_APPEND instead.
func (m *multiOutput) OpenLogFile(path string, appendMode bool) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return err
	}

	m.logFile = f
	return nil
}

// Close flushes and closes the log file, if one is open.
func (m *multiOutput) Close() {
	if m.logFile != nil {
		m.logFile.Sync()
		m.logFile.Close()
	}
}

func (m *multiOutput) openPipe(proc *process) (pipe *ptyPipe) {
	var err error

	pipe = m.pipes[proc]

	pipe.pty, pipe.tty, err = termios.Pty()
	fatalOnErr(err)

	proc.Stdout = pipe.tty
	proc.Stderr = pipe.tty
	proc.Stdin = pipe.tty
	proc.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}

	return
}

func (m *multiOutput) Connect(proc *process) {
	if len(proc.Name) > m.maxNameLength {
		m.maxNameLength = len(proc.Name)
	}

	if m.pipes == nil {
		m.pipes = make(map[*process]*ptyPipe)
	}

	m.pipes[proc] = &ptyPipe{}
}

func (m *multiOutput) PipeOutput(proc *process) {
	pipe := m.openPipe(proc)

	// Forward our terminal's stdin to the interactive process so its dev
	// server can read keypresses (e.g. Expo/Metro reload and platform keys).
	// In TUI mode the console owns the keyboard and routes keys per focused
	// tab via WriteStdin, so this direct copy must be disabled to avoid two
	// readers fighting over os.Stdin.
	if proc == m.interactiveProc && m.program == nil {
		go func(pipe *ptyPipe) {
			io.Copy(pipe.pty, os.Stdin)
		}(pipe)
	}

	go func(proc *process, pipe *ptyPipe) {
		scanLines(pipe.pty, func(b []byte) bool {
			m.WriteLine(proc, b)
			return true
		})
	}(proc, pipe)
}

func (m *multiOutput) ClosePipe(proc *process) {
	if pipe := m.pipes[proc]; pipe != nil {
		pipe.pty.Close()
		pipe.tty.Close()
	}
}

// linePrefix builds the colored timestamp/name prefix for a process line,
// honoring printTimestamp/printProcName. Returns "" when both are disabled.
func (m *multiOutput) linePrefix(proc *process, now time.Time) string {
	if !m.printProcName && !m.printTimestamp {
		return ""
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "\033[1;38;5;%vm", proc.Color)

	if m.printTimestamp {
		buf.WriteString(now.Format("15:04:05"))
		buf.WriteByte(' ')
	}

	if m.printProcName {
		buf.WriteString(proc.Name)

		for i := len(proc.Name); i <= m.maxNameLength; i++ {
			buf.WriteByte(' ')
		}
	}

	buf.WriteString("\033[0m| ")

	return buf.String()
}

func (m *multiOutput) WriteLine(proc *process, p []byte) {
	now := time.Now()

	prefix := m.linePrefix(proc, now)

	// TUI mode: hand the line to the program instead of writing to stdout. The
	// log file (below) is still written so --log-file works in both modes.
	if m.program != nil {
		m.program.Send(logLineMsg{proc: proc, raw: string(p), prefixed: prefix + string(p)})
	} else {
		var buf bytes.Buffer

		eol := m.eol
		if eol == "" {
			eol = "\n"
		}

		buf.WriteString(prefix)
		buf.Write(p)
		buf.WriteString(eol)

		m.mutex.Lock()
		buf.WriteTo(os.Stdout)
		m.mutex.Unlock()
	}

	if m.logFile != nil {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		var fbuf bytes.Buffer

		if m.printProcName || m.printTimestamp {
			if m.printTimestamp {
				fbuf.WriteString(now.Format("15:04:05"))
				fbuf.WriteByte(' ')
			}

			if m.printProcName {
				fbuf.WriteString(proc.Name)

				for i := len(proc.Name); i <= m.maxNameLength; i++ {
					fbuf.WriteByte(' ')
				}
			}

			fbuf.WriteString("| ")
		}

		fbuf.Write(stripANSI(p))
		fbuf.WriteByte('\n')

		fbuf.WriteTo(m.logFile)
	}
}

// WriteStdin forwards bytes to a process's PTY, used by the TUI to deliver
// keystrokes to the focused profile (per-tab passthrough).
func (m *multiOutput) WriteStdin(proc *process, b []byte) {
	if pipe := m.pipes[proc]; pipe != nil && pipe.pty != nil {
		pipe.pty.Write(b)
	}
}

func (m *multiOutput) WriteErr(proc *process, err error) {
	m.WriteLine(proc, []byte(
		fmt.Sprintf("\033[0;31m%v\033[0m", err),
	))
}
