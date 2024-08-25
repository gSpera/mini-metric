package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

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

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "battery_percent",
		Help: "Battery % value, range 0-100",
	}, batteryPercent))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "battery_status",
		Help: "Battery status, 0 -> Unkown, 1 -> Charging, 2 -> Discharging, 3 -> Not Charging (Connected and charged)",
	}, batteryStatus))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Println("Listening on", *listenAddr)
	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Fatalln("While listening:", err)
	}
}
