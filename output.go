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
	if proc == m.interactiveProc {
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

func (m *multiOutput) WriteLine(proc *process, p []byte) {
	now := time.Now()

	var buf bytes.Buffer

	if m.printProcName || m.printTimestamp {
		color := fmt.Sprintf("\033[1;38;5;%vm", proc.Color)

		buf.WriteString(color)

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
	}

	eol := m.eol
	if eol == "" {
		eol = "\n"
	}

	buf.Write(p)
	buf.WriteString(eol)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	buf.WriteTo(os.Stdout)

	if m.logFile != nil {
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

func (m *multiOutput) WriteErr(proc *process, err error) {
	m.WriteLine(proc, []byte(
		fmt.Sprintf("\033[0;31m%v\033[0m", err),
	))
}
