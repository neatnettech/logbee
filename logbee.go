package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
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
}

type logbee struct {
	title       string
	output      *multiOutput
	procs       []*process
	procWg      sync.WaitGroup
	done        chan bool
	interrupted chan os.Signal
	timeout     time.Duration
}

func newLogbee(conf logbeeConfig) (h *logbee) {
	h = &logbee{timeout: time.Duration(conf.Timeout) * time.Second}

	if len(conf.Title) > 0 {
		h.title = conf.Title
	} else {
		h.title = filepath.Base(conf.Root)
	}

	h.output = &multiOutput{printProcName: !conf.NoPrefix, printTimestamp: conf.PrintTimestamps}

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

	return
}

func (h *logbee) runProcess(proc *process) {
	h.procWg.Add(1)

	go func() {
		defer h.procWg.Done()
		defer func() { h.done <- true }()

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

	for _, proc := range h.procs {
		h.runProcess(proc)
	}

	go h.waitForExit()

	h.procWg.Wait()

	h.output.Close()
}
