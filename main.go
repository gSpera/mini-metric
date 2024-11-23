package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	listenAddr := flag.String("listen-addr", ":7002", "Listen address")
	configFile := flag.String("config-file", "mini-metric.toml", "Config file")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(log)

	// Read rules
	var cfg map[string]Rule
	_, err := toml.DecodeFile(*configFile, &cfg)
	if err != nil {
		log.Error("Cannot decode config file", "file", *configFile, "err", err)
		os.Exit(1)
		return
	}

	registry := prometheus.NewRegistry()

	var initRule Rule
	handlers := make([]Rule, 0, len(cfg))
	for name, rule := range cfg {
		rule.Name = name
		log := log.With("rule-name", name)
		log.Info("Found rule", "shell", rule.ShellCommand)
		typ, ok := rule.detectType()
		if !ok {
			log.Error("Cannot deduce rule type")
			continue
		}
		log.Info("Detected type", "type", typ.String())

		switch typ {
		case Shell:
			rule.handler = ShellHandler{
				Rule: rule,
				log:  log,
			}
		case File:
			rule.handler = FileHandler{
				Rule: rule,
				log:  log,
			}
		case Init:
			initRule = rule
			initRule.handler = ShellHandler{
				Rule: rule,
				log:  log,
			}
			log.Info("Found init rule", "name", initRule.Name, "description", initRule.Description)
			continue // Do not register
		default:
			panic("Rule handler not implemented")
		}

		log.Info("Metric type", "metric-type", rule.MetricType)

		err := registry.Register(rule.Collector())
		if err != nil {
			log.Error("Cannot register rule", "err", err)
			continue
		}
		handlers = append(handlers, rule)
	}

	// Prepare metrics
	http.HandleFunc("/metrics", httpMetricHandler(log.With("what", "metric-handler"), initRule, handlers, promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))

	log.Info("Listening on", "listen-addr", *listenAddr)
	err = http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Error("Error while listening", "err", err)
		os.Exit(1)
	}
}

func httpMetricHandler(log *slog.Logger, initRule Rule, metrics []Rule, httpHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Running init rule")
		initRule.handler.Exec()

		log.Info("Updating metrics")
		for _, rule := range metrics {
			out := rule.handler.Exec()
			for _, line := range out {
				rule.collector.With(line.Labels).Set(line.Value)
			}
		}

		httpHandler.ServeHTTP(w, r)
	}
}
