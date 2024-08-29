package main

import (
	"bytes"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type RuleHandler interface {
	Collector() prometheus.Collector
}

type ShellHandler struct {
	Rule

	log *slog.Logger
}

func (s ShellHandler) Exec() float64 {
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
		if err != nil {
			s.log.Error("Cannot copy stdout to buffer", "err", err)
			prStdout.CloseWithError(err)
		}
	}()
	go func() {
		_, err := io.Copy(&stderr, prStderr)
		if err != nil {
			s.log.Error("Cannot copy stderr to buffer", "err", err)
			prStderr.CloseWithError(err)
		}
	}()

	cmd.WaitDelay = 10 * time.Second
	err := cmd.Start()
	if err != nil {
		s.log.Error("Cannot start command", "err", err)
		return math.NaN()
	}

	err = cmd.Wait()
	if err != nil {
		s.log.Error("Cannot execute command", "err", err, "rawStdout", stdout.String(), "rawStderr", stderr.String())
		return math.NaN()
	}

	// Parse output
	output := strings.TrimSpace(stdout.String())
	v, err := strconv.ParseFloat(output, 64)
	if err != nil {
		s.log.Error("Cannot parse command output as float64", "err", err, "rawStdout", stdout.String(), "rawStderr", stderr.String())
		return math.NaN()
	}

	return v
}

func (s ShellHandler) Collector() prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: s.Name,
		Help: s.Description,
	}, s.Exec)
}

type FileHandler struct {
	Rule

	log *slog.Logger
}

func (f FileHandler) Exec() float64 {
	content, err := os.ReadFile(f.FilePath)
	if err != nil {
		f.log.Error("Cannot read file", "file", f.FilePath)
		return math.NaN()
	}

	str := strings.TrimSpace(string(content))
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		f.log.Error("Cannot parse file content", "rawContent", string(content))
		return math.NaN()
	}

	return v
}

func (f FileHandler) Collector() prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: f.Name,
		Help: f.Description,
	}, f.Exec)
}
