package main

import (
	"context"
	"github.com/arl/statsviz"
	"gopkg.in/alecthomas/kingpin.v2"
	"kube-scheduler-extender/conf"
	"kube-scheduler-extender/controller"
	"kube-scheduler-extender/routers"
	"kube-scheduler-extender/util"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/common/log"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

var (
	prometheusUrl             = kingpin.Flag("prometheus_url", "Prometheus url. (env: PROMETHEUS_URL)").Default(util.GetEnv("PROMETHEUS_URL", "http://127.0.0.1:9090")).String()
	prometheusMemoryMetrics   = kingpin.Flag("prometheus_memory_metrics", "Prometheus memory metrics. (env: PROMETHEUS_MEMORY_METRICS)").Default(util.GetEnv("PROMETHEUS_MEMORY_METRICS", "HostMemoryUsagePercent")).String()
	prometheusMemoryThreshold = kingpin.Flag("prometheus_memory_threshold", "Prometheus memory threshold. (env: PROMETHEUS_MEMORY_THRESHOLD)").Default(util.GetEnv("PROMETHEUS_MEMORY_THRESHOLD", "80")).Int()
	listenAddress             = kingpin.Flag("listen_address", "Address to listen on for web interface and telemetry. (env: LISTEN_ADDRESS)").Default(util.GetEnv("LISTEN_ADDRESS", ":8888")).String()
	logRequestBody            = kingpin.Flag("log_request_body", "Log k8s request body. (env: LOG_REQUEST_BODY)").Default(util.GetEnv("LOG_REQUEST_BODY", "false")).Bool()
)

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	conf.NewConfig(*prometheusUrl, *prometheusMemoryMetrics, *prometheusMemoryThreshold, *logRequestBody)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controller.NewNodeInfo(ctx.Done())

	go func() {
		statsviz.RegisterDefault()
		http.ListenAndServe(":8889", nil)
	}()

	log.Infoln("start up debug!, API server listening at http://localhost:8889/debug/statsviz/")

	log.Infoln("start up kube-scheduler-extender!, API server listening at ", *listenAddress)
	http.ListenAndServe(*listenAddress, routers.Router)

}
