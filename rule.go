package main

import (
	"fmt"
	"strings"
)

type MetricType int

const (
	Gauge MetricType = iota // Gauge is the default
	Counter
)

func (m MetricType) String() string {
	switch m {
	case Gauge:
		return "Gauge"
	case Counter:
		return "Counter"
	default:
		panic("Invalid MetricType")
	}
}

type RuleType int

const (
	Shell RuleType = iota
	File
)

func (r RuleType) String() string {
	switch r {
	case Shell:
		return "Shell"
	case File:
		return "File"
	default:
		panic("Invalid RuleType")
	}
}

type Rule struct {
	Name        string
	Description string
	MetricType  MetricType

	// Shell
	ShellCommand string `toml:"Command"`
	// File
	FilePath string `toml:"File"`
}

func (r *MetricType) TextUnmarshaler(data []byte) error {
	str := string(data)

	switch str {
	case "counter":
		*r = Counter
	case "gauge":
		*r = Gauge
	default:
		return fmt.Errorf("invalid RuleType %q", str)
	}

	return nil
}

func (r *Rule) detectType() (RuleType, bool) {
	if strings.TrimSpace(r.ShellCommand) != "" {
		return Shell, true
	}
	if strings.TrimSpace(r.FilePath) != "" {
		return File, true
	}

	return Shell, false
}
