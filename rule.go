package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
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

// A Collector extends a prometheus.Collector enableing
// the user to change the metric values
type Collector interface {
	prometheus.Collector
	With(prometheus.Labels) prometheus.Gauge
}

type Rule struct {
	Name        string `toml:"-"`
	Description string
	MetricType  MetricType

	Labels      []string // Deprecated
	collector Collector
	handler   RuleHandler

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

func (r *Rule) Collector() prometheus.Collector {
	if r.collector != nil {
		return r.collector
	}

	switch r.MetricType {
	case Gauge:
		r.collector = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: r.Name,
			Help: r.Description,
		}, r.Labels)
	case Counter:
		// TODO
		//r.collector = prometheus.NewCounterVec(prometheus.CounterOpts{
		//	Name: r.Name,
		//	Help: r.Description,
		//}, r.Labels)
		fallthrough
	default:
		panic("Rule collector not implemented")
	}

	return r.collector
}
