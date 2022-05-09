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
	err = prometheus.Register(version.NewCollector("speedify_exporter"))
	if err != nil {
		log.Errorf("Failed to register collector: %s", err)
	}
	err = prometheus.Register(&SpeedifyCollector{
		stateMetric:                     prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "state"), "The state of Speedify: 0 (LOGGED_OUT), 1 (LOGGED_IN), 2 (CONNECTED), 3 (OTHERS)", nil, nil),
		adapterPriorityMetric:           prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "priority"), "The priority of the adapter: 0 (never), 1 (always), 2 (secondary), 3 (backup), 4 (unknown)", []string{"adapterId", "adapterType"}, nil),
		adapterStateMetric:              prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "state"), "The state of the adapter: 0 (disconnected), 1 (connected), 2 (unknown)", []string{"adapterId", "adapterType"}, nil),
		adpaterOverlimitRateLimitMetric: prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "overlimit_rate_limit"), "The overlimit rate limit of the adapter.", []string{"adapterId", "adapterType"}, nil),
		adpaterUsageDailyMetric:         prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "daily_usage"), "The daily data usage of the adapter.", []string{"adapterId", "adapterType"}, nil),
		adpaterUsageDailyBoostMetric:    prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "daily_usage_boost"), "The daily data usage boost of the adapter.", []string{"adapterId", "adapterType"}, nil),
		adapterUsageDailyLimitMetric:    prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "daily_usage_limit"), "The daily data usage limit of the adapter.", []string{"adapterId", "adapterType"}, nil),
		adpaterUsageMonthlyMetric:       prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "monthly_usage"), "The monthly data usage of the adapter.", []string{"adapterId", "adapterType"}, nil),
		adpaterUsageMonthlyLimitMetric:  prometheus.NewDesc(prometheus.BuildFQName(namespace, "adapter", "monthly_usage_limit"), "The monthly data usage limit of the adapter.", []string{"adapterId", "adapterType"}, nil)})
	if err != nil {
		log.Errorf("Failed to register collector: %s", err)
	}

	// Regist http handler
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h := promhttp.HandlerFor(prometheus.Gatherers{
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})

	// start server
	log.Infof("Starting http server - %s", bind)
	if err = http.ListenAndServe(bind, nil); err != nil {
		log.Errorf("Failed to start http server: %s", err)
	}
}

type SpeedifyCollector struct {
	stateMetric                     *prometheus.Desc
	adapterPriorityMetric           *prometheus.Desc
	adapterStateMetric              *prometheus.Desc
	adpaterOverlimitRateLimitMetric *prometheus.Desc
	adpaterUsageDailyMetric         *prometheus.Desc
	adpaterUsageDailyBoostMetric    *prometheus.Desc
	adapterUsageDailyLimitMetric    *prometheus.Desc
	adpaterUsageMonthlyMetric       *prometheus.Desc
	adpaterUsageMonthlyLimitMetric  *prometheus.Desc
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
	var state SpeedifyState
	err = json.Unmarshal([]byte(out), &state)
	if err != nil {
		log.Errorf("Errored while parsing speedify state response JSON: %s", err)
	}
	ch <- prometheus.MustNewConstMetric(collector.stateMetric, prometheus.GaugeValue, state.getStateMetric())

	out, err = exec.Command(speedify_cli, "show", "adapters").Output()
	if err != nil {
		log.Errorf("Errored while getting speedify adapters: %s", err)
	}
	var adapters []Adapter
	err = json.Unmarshal([]byte(out), &adapters)
	if err != nil {
		log.Errorf("Errored while parsing speedify show adapters response JSON: %s", err)
	}

	for _, a := range adapters {
		ch <- prometheus.MustNewConstMetric(collector.adapterPriorityMetric, prometheus.CounterValue, a.getPriorityMetric(), a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adapterStateMetric, prometheus.GaugeValue, a.getStateMetric(), a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adpaterOverlimitRateLimitMetric, prometheus.GaugeValue, a.DataUsage.OverlimitRateLimit, a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adpaterUsageDailyMetric, prometheus.CounterValue, a.DataUsage.UsageDaily, a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adpaterUsageDailyBoostMetric, prometheus.GaugeValue, a.DataUsage.UsageDailyBoost, a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adapterUsageDailyLimitMetric, prometheus.GaugeValue, a.DataUsage.UsageDailyLimit, a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adpaterUsageMonthlyMetric, prometheus.CounterValue, a.DataUsage.UsageMonthly, a.AdapterID, a.Type)
		ch <- prometheus.MustNewConstMetric(collector.adpaterUsageMonthlyLimitMetric, prometheus.GaugeValue, a.DataUsage.UsageMonthlyLimit, a.AdapterID, a.Type)
	}

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

type Adapter struct {
	AdapterID string
	DataUsage DataUsage
	Priority  string
	State     string
	Type      string
}

type DataUsage struct {
	OverlimitRateLimit float64
	UsageDaily         float64
	UsageDailyBoost    float64
	UsageDailyLimit    float64
	UsageMonthly       float64
	UsageMonthlyLimit  float64
}

func (adapter *Adapter) getPriorityMetric() float64 {
	switch adapter.Priority {
	case "never":
		return 0
	case "always":
		return 1
	case "secondary":
		return 2
	case "backup":
		return 3
	default:
		return 4
	}
}

func (adapter *Adapter) getStateMetric() float64 {
	switch adapter.State {
	case "disconnected":
		return 0
	case "connected":
		return 1
	default:
		return 2
	}
}
