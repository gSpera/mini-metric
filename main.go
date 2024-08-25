package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xellio/tools/acpi"
)

func retrieveBattery() (*acpi.BatteryInformation, error) {
	batt, err := acpi.Battery()
	if err != nil {
		return nil, fmt.Errorf("cannot get battery level: %w", err)
	}
	if len(batt) == 0 {
		return nil, fmt.Errorf("no battery detected")
	}
	if len(batt) != 1 {
		log.Println("Multiple battery presents, using first one")
	}

	return batt[0], nil
}

func batteryPercent() float64 {
	b, err := retrieveBattery()
	if err != nil {
		log.Println("Cannot retrieve battery:", err)
		return math.NaN()
	}

	return float64(b.Level) / 100
}

func batteryStatus() float64 {
	b, err := retrieveBattery()
	if err != nil {
		log.Println("Cannot retrieve battery:", err)
		return math.NaN()
	}

	label := strings.ToLower(b.Status)

	switch label {
	case "charging":
		return 1
	case "discharging":
		return 2
	case "not charging":
		return 3
	default:
		return 0
	}
}

func main() {
	listenAddr := flag.String("listen-addr", ":7002", "Listen address")
	configFile := flag.String("config-file", "mini-metric.toml", "Config file")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Read rules
	var cfg map[string]Rule
	_, err := toml.DecodeFile(*configFile, &cfg)
	if err != nil {
		log.Error("Cannot decode config file", "file", *configFile, "err", err)
		os.Exit(1)
		return
	}

	registry := prometheus.NewRegistry()

	for name, rule := range cfg {
		rule.Name = name
		log := log.With("rule-name", name)
		log.Info("Found rule", "shell", rule.Command)
		cmd := strings.TrimSpace(rule.Command)

		if cmd == "" {
			log.Warn("Empty shell command", "shell", cmd)
			continue
		}

		h := ShellHandler{
			Rule: rule,
			log:  log,
		}
		err := registry.Register(h.Collector())
		if err != nil {
			log.Error("Cannot register rule", "err", err)
			continue
		}
	}

	// Prepare metrics
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "battery_percent",
		Help: "Battery % value, range 0-100",
	}, batteryPercent))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "battery_status",
		Help: "Battery status, 0 -> Unkown, 1 -> Charging, 2 -> Discharging, 3 -> Not Charging (Connected and charged)",
	}, batteryStatus))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Info("Listening on", "listen-addr", *listenAddr)
	err = http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Error("Error while listening", "err", err)
		os.Exit(1)
	}
}
