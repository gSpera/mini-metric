package main

import "fmt"

type Rule struct {
	Name string
	Description string
	Type        RuleType
	Command       string
}

type RuleType int

const (
	Counter RuleType = iota
	Gauge
)

func (r *RuleType) TextUnmarshaler(data []byte) error {
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
