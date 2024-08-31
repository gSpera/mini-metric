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
// ([label1="value1",label2="value2"]|value) floatValue\n
// where [...] is optional, (A|B) is one of A and B

type OutputParser struct{}

type Output []OutputLine

type OutputLine struct {
	Labels prometheus.Labels
	Value  float64
}

var NaNOutput Output = []OutputLine{{Labels: nil, Value: math.NaN()}}

var parsingRegex = regexp.MustCompile(`(?:(\w+?)="(.*?)"[,\s]?)+?`)

func ParseReader(r io.Reader) (Output, error) {
	rr := bufio.NewReader(r)
	lines := make([]OutputLine, 0)
	id := rand.Uint()
	log := slog.Default().With("id", id)

	log.Debug("Parsing output")
	for {
		// Read a single line
		line, err := rr.ReadString('\n')

		if line == "" && err == io.EOF {
			log.Debug("Found empty line with EOF")
			break
		}

		if err != nil && err != io.EOF {
			log.Debug("Error while reading not nil", "line", line, "err", err)
			return Output(lines), fmt.Errorf("cannot read output: %w", err)
		}

		log.Debug("Reading metric value", "line", line, "err", err)
		vs := strings.Fields(line)
		if len(vs) > 2 {
			log.Warn("Expected 1 or 2 fields in line", "fields-count", len(vs), "fields", vs)
		}

		// Read the last value after a space
		metricValue, err := strconv.ParseFloat(strings.TrimSpace(vs[len(vs)-1]), 64)
		if err != nil {
			log.Debug("Cannot convert metric value", "err", err, "value", vs[len(vs)-1])
			return Output(lines), fmt.Errorf("invalid float value: (value=%q) %w", vs[len(vs)-1], err)
		}
		metricValueSize := len(vs[len(vs)-1])
		log.Debug("Done reading metric value", "value", metricValue, "size", metricValueSize)

		if len(vs) == 1 {
			log.Debug("Found only metric value, skipping labels")
			lines = append(lines, OutputLine {
				Labels: nil,
				Value: metricValue,
			})
			continue
		}

		// Read the labels
		rawLabels := vs[0]
		labels := make(map[string]string)
		log.Debug("Reading labels", "rawlabels", rawLabels)

		matches := parsingRegex.FindAllStringSubmatch(rawLabels, -1)
		log.Debug("Matches", "matches", matches)
		for _, match := range matches {
			// match[0] is the match
			// fmt.Println("Full match:", match[0])
			// fmt.Println("Label:", match[1])
			// fmt.Println("Value:", match[2])
			labels[match[1]] = match[2]
		}

		if len(matches) == 0 && !strings.ContainsAny(rawLabels, "=,") {
			labels["type"] = rawLabels
		}

		lines = append(lines, OutputLine{
			Labels: labels,
			Value:  metricValue,
		})

		if err == io.EOF {
			log.Debug("Found EOF")
			break
		}

	}

	log.Debug("Done parsing")
	return Output(lines), nil
}
