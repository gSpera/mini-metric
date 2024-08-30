package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

type RuleHandler interface {
	Exec() Output
}

type ShellHandler struct {
	Rule

	log *slog.Logger
}

func (s ShellHandler) Exec() Output {
	s.log.Debug("Executing command", "cmd", s.ShellCommand)
	cmd := exec.Command("sh", "-c", s.ShellCommand)
	prStdout, pwStdout := io.Pipe()
	prStderr, pwStderr := io.Pipe()
	defer pwStdout.Close()
	defer pwStderr.Close()
	cmd.Stdout = pwStdout
	cmd.Stderr = pwStderr
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	go func() {
		_, err := io.Copy(&stdout, prStdout)
		s.log.Debug("Done copying stdout", "err", err)
		if err != nil {
			s.log.Error("Cannot copy stdout to buffer", "err", err)
			prStdout.CloseWithError(err)
		}
	}()
	go func() {
		_, err := io.Copy(&stderr, prStderr)
		s.log.Debug("Done copying stderr", "err", err)
		if err != nil {
			s.log.Error("Cannot copy stderr to buffer", "err", err)
			prStderr.CloseWithError(err)
		}
	}()

	cmd.WaitDelay = 10 * time.Second
	cmd.WaitDelay = 1 * time.Second
	s.log.Debug("Starting command")
	err := cmd.Start()
	s.log.Debug("Done cmd.Start")
	if err != nil {
		s.log.Error("Cannot start command", "err", err)
		return NaNOutput
	}

	s.log.Debug("Waiting command")
	err = cmd.Wait()
	s.log.Debug("Done cmd.Wait")
	if err != nil {
		s.log.Error("Cannot execute command", "err", err, "rawStdout", stdout.String(), "rawStderr", stderr.String())
		return NaNOutput
	}

	s.log.Debug("Closing stdout and stderr")
	pwStdout.Close()
	pwStderr.Close()

	s.log.Debug("Parsing output")
	out, err := ParseReader(prStdout)
	s.log.Debug("Done parsing output")
	if err != nil {
		s.log.Error("Cannot parse command output", "err", err, "rawStdout", stdout.String(), "rawStderr", stderr.String())
	}

	return out
}

type FileHandler struct {
	Rule

	log *slog.Logger
}

func (f FileHandler) Exec() Output {
	fl, err := os.Open(f.FilePath)
	if err != nil {
		f.log.Error("Cannot read file", "file", f.FilePath)
		return NaNOutput
	}
	defer fl.Close()

	out, err := ParseReader(fl)
	if err != nil {
		f.log.Error("Cannot parse file content", "file", f.FilePath, "err", err)
	}

	return out
}
