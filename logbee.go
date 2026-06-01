package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

var colors = []int{2, 3, 4, 5, 6, 42, 130, 103, 129, 108}

type logbeeConfig struct {
	Title              string
	Procfile           string
	ProcNames          string
	Root               string
	PortBase, PortStep int
	Timeout            int
	NoPrefix           bool
	PrintTimestamps    bool
	LogFile            string
	LogAppend          bool
	Interactive        string
	TUI                bool
	Scrollback         int
}

type logbee struct {
	title       string
	output      *multiOutput
	procs       []*process
	procWg      sync.WaitGroup
	done        chan bool
	interrupted chan os.Signal
	timeout     time.Duration
	tui         bool
	scrollback  int
}

func newLogbee(conf logbeeConfig) (h *logbee) {
	h = &logbee{timeout: time.Duration(conf.Timeout) * time.Second, tui: conf.TUI, scrollback: conf.Scrollback}

	if len(conf.Title) > 0 {
		h.title = conf.Title
	} else {
		h.title = filepath.Base(conf.Root)
	}

	h.output = &multiOutput{printProcName: !conf.NoPrefix, printTimestamp: conf.PrintTimestamps, eol: "\n"}

	if len(conf.LogFile) > 0 {
		fatalOnErr(h.output.OpenLogFile(conf.LogFile, conf.LogAppend))
	}

	entries := parseProcfile(conf.Procfile, conf.PortBase, conf.PortStep)
	h.procs = make([]*process, 0)

	procNames := splitAndTrim(conf.ProcNames)

	for i, entry := range entries {
		if len(procNames) == 0 || stringsContain(procNames, entry.Name) {
			h.procs = append(h.procs, newProcess(entry.Name, entry.Command, colors[i%len(colors)], conf.Root, entry.Port, h.output))
		}
	}

	if len(conf.Interactive) > 0 {
		for _, proc := range h.procs {
			if proc.Name == conf.Interactive {
				h.output.interactiveProc = proc
				break
			}
		}
		if h.output.interactiveProc == nil {
			fatal(fmt.Sprintf("interactive process %q not found among launched processes", conf.Interactive))
		}
	}

	return
}

func (h *logbee) runProcess(proc *process) {
	h.procWg.Add(1)

	go func() {
		defer h.procWg.Done()
		defer func() {
			if h.output.program != nil {
				h.output.program.Send(procExitMsg{proc})
			}
			h.done <- true
		}()

		proc.Run()
	}()
}

func (h *logbee) waitForDoneOrInterrupt() {
	select {
	case <-h.done:
	case <-h.interrupted:
	}
}

func (h *logbee) waitForTimeoutOrInterrupt() {
	select {
	case <-time.After(h.timeout):
	case <-h.interrupted:
	}
}

func (h *logbee) waitForExit() {
	h.waitForDoneOrInterrupt()

	for _, proc := range h.procs {
		go proc.Interrupt()
	}

	h.waitForTimeoutOrInterrupt()

	for _, proc := range h.procs {
		go proc.Kill()
	}
}

func (h *logbee) Run() {
	fmt.Printf("\033]0;%s | logbee\007", h.title)

	h.done = make(chan bool, len(h.procs))

	h.interrupted = make(chan os.Signal)
	signal.Notify(h.interrupted, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if h.tui {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			h.runTUI()
			h.output.Close()
			return
		}
		fmt.Fprintln(os.Stderr, "logbee: --tui requires a terminal; falling back to plain output")
	}

	h.runPlain()
	h.output.Close()
}

// runTUI launches the Bubble Tea console (tab per process + aggregate) and
// drives process lifecycle around it.
func (h *logbee) runTUI() {
	model := newTUIModel(h.output, h.procs, h.scrollback)
	program := tea.NewProgram(model, tea.WithAltScreen())
	h.output.program = program

	for _, proc := range h.procs {
		h.runProcess(proc)
	}

	// Quit the TUI once every process has exited on its own, or on an OS signal.
	allDone := make(chan struct{})
	go func() {
		h.procWg.Wait()
		close(allDone)
	}()
	go func() {
		select {
		case <-allDone:
		case <-h.interrupted:
		}
		program.Quit()
	}()

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "logbee: tui error: %v\n", err)
	}

	// Tear down any processes still running (e.g. the user pressed q).
	for _, proc := range h.procs {
		go proc.Interrupt()
	}
	h.waitForTimeoutOrInterrupt()
	for _, proc := range h.procs {
		go proc.Kill()
	}
	h.procWg.Wait()
}

func (h *logbee) runPlain() {
	// When a process is interactive, put our own terminal into raw mode so
	// single keypresses are forwarded to it immediately (no line buffering or
	// local echo). Raw mode disables output post-processing, so aggregated
	// output switches to CRLF line endings to avoid staircasing.
	if h.output.interactiveProc != nil {
		if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
			if state, err := term.MakeRaw(fd); err == nil {
				h.output.eol = "\r\n"
				defer term.Restore(fd, state)
			}
		}
	}

	for _, proc := range h.procs {
		h.runProcess(proc)
	}

	go h.waitForExit()

	h.procWg.Wait()
}
