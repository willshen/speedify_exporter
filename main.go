package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

const (
	namespace    = "speedify"
	speedify_cli = "/usr/share/speedify/speedify_cli"
)

func main() {
	_, err := exec.LookPath(speedify_cli)

	if err != nil {
		log.Fatalf("Failed to find speedify_cli: %s", err)
	}

	// =====================
	// Get OS parameter
	// =====================
	var bind string
	flag.StringVar(&bind, "bind", "0.0.0.0:9961", "bind")
	flag.Parse()

	// ========================
	// Regist handler
	// ========================
	prometheus.Register(version.NewCollector("speedify_exporter"))
	prometheus.Register(&SpeedifyCollector{stateMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "state"), "The state of Speedify: 0 (LOGGED_OUT), 1 (LOGGED_IN), 2 (CONNECTED)", nil, nil)})

	// Regist http handler
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h := promhttp.HandlerFor(prometheus.Gatherers{
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})

	// start server
	log.Infof("Starting http server - %s", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		log.Errorf("Failed to start http server: %s", err)
	}
}

type SpeedifyCollector struct {
	stateMetric *prometheus.Desc
}

// Describe prometheus describe
func (collector *SpeedifyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.stateMetric
}

// Collect prometheus collect
func (collector *SpeedifyCollector) Collect(ch chan<- prometheus.Metric) {
	out, err := exec.Command(speedify_cli, "state").Output()
	if err != nil {
		log.Errorf("Errored while getting speedify state: %s", err)
	}
	state := SpeedifyState{}
	json.Unmarshal([]byte(out), &state)
	ch <- prometheus.MustNewConstMetric(collector.stateMetric, prometheus.GaugeValue, state.getStateMetric())
}

type SpeedifyState struct {
	State string
}

func (state *SpeedifyState) getStateMetric() float64 {
	switch state.State {
	case "LOGGED_OUT":
		return 0
	case "LOGGED_IN":
		return 1
	case "CONNECTED":
		return 2
	default:
		return 3
	}
}
