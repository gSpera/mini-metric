package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// The output format is a series of lines
// [label1="value1",label2="value2"] floatValue\n
// where [...] is optional

type OutputParser struct{}

type Output []OutputLine

type OutputLine struct {
	Labels prometheus.Labels
	Value  float64
}

var NaNOutput Output = []OutputLine{{Labels: nil, Value: math.NaN()}}

var parsingRegex = regexp.MustCompile(`(?:(\w+?)="(.*?)"[,\s])+?`)

func ParseReader(r io.Reader) (Output, error) {
	// TODO: Implement
	rr := bufio.NewReader(r)
	lines := make([]OutputLine, 0)
	id := rand.Uint()
	log := slog.Default().With("id", id)

	log.Debug("Parsing output")
	for {
		// Read a single line
		line, err := rr.ReadString('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			return Output(lines), fmt.Errorf("cannot read output: %w", err)
		}

		// Read the last value after a space
		log.Debug("Reading metric value", "line", line)
		vs := strings.Split(line, " ")
		metricValue, err := strconv.ParseFloat(strings.TrimSpace(vs[len(vs)-1]), 64)
		if err != nil {
			return Output(lines), fmt.Errorf("invalid float value: (value=%q) %w", vs[len(vs)-1], err)
		}
		metricValueSize := len(vs[len(vs)-1])
		log.Debug("Done reading metric value", "value", metricValue, "size", metricValueSize)

		// Read the labels
		rawLabels := line[:len(line)-metricValueSize]
		labels := make(map[string]string)
		log.Debug("Reading labels", "rawlabels", rawLabels)

		matches := parsingRegex.FindAllStringSubmatch(rawLabels, -1)
		fmt.Println(matches)
		for _, match := range matches {
			// match[0] is the match
			// fmt.Println("Full match:", match[0])
			// fmt.Println("Label:", match[1])
			// fmt.Println("Value:", match[2])
			labels[match[1]] = match[2]
		}

		lines = append(lines, OutputLine{
			Labels: labels,
			Value:  metricValue,
		})
	}

	log.Debug("Done parsing")
	return Output(lines), nil
}
