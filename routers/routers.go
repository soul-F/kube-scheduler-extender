package routers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/log"
	"io"
	"kube-scheduler-extender/algorithm"
	"kube-scheduler-extender/conf"
	"kube-scheduler-extender/metrics"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	schedulerapi "k8s.io/kube-scheduler/extender/v1"
)

var Router *httprouter.Router

func init() {
	metrics.Register()
	Router = httprouter.New()
	Router.GET("/", Index)
	Router.GET("/healthcheck", HealthCheck)
	Router.POST("/filter", Filter)
	Router.POST("/prioritize", Prioritize)
	Router.Handler("GET", "/metrics", metrics.PrometheusHandler)

}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome to kube-scheduler-extender!\n")
}

func HealthCheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "OK\n")

}

func Filter(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	startPredicateEvalTime := time.Now()
	defer func() {
		metrics.PodSchedulePredicate.Inc()
		metrics.SchedulingAlgorithmPredicateEvaluationDuration.WithLabelValues().Observe(metrics.SinceInSeconds(startPredicateEvalTime))
	}()
	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)
	var extenderArgs schedulerapi.ExtenderArgs
	var extenderFilterResult *schedulerapi.ExtenderFilterResult
	if err := json.NewDecoder(body).Decode(&extenderArgs); err != nil {
		log.Errorln("解析参数错误:", err)
		extenderFilterResult = &schedulerapi.ExtenderFilterResult{
			Error: err.Error(),
		}
	} else {
		if conf.Conf.LogRequestBody {
			b, _ := json.Marshal(extenderArgs)
			log.Infoln(string(b))
		}

		extenderFilterResult = algorithm.Filter(extenderArgs)
	}

	if response, err := json.Marshal(extenderFilterResult); err != nil {
		log.Errorln("json 格式化 extenderFilterResult:", err)
		panic(err)
	} else {
		metrics.PodSchedulePredicateSuccesses.Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

func Prioritize(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	startPriorityEvalTime := time.Now()
	defer func() {
		metrics.PodSchedulePriority.Inc()
		metrics.SchedulingAlgorithmPriorityEvaluationDuration.WithLabelValues().Observe(metrics.SinceInSeconds(startPriorityEvalTime))
	}()

	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)
	var extenderArgs schedulerapi.ExtenderArgs
	var hostPriorityList *schedulerapi.HostPriorityList
	if err := json.NewDecoder(body).Decode(&extenderArgs); err != nil {
		log.Errorln("解析参数错误:", err)
		hostPriorityList = &schedulerapi.HostPriorityList{}
	} else {
		if conf.Conf.LogRequestBody {
			b, _ := json.Marshal(extenderArgs)
			log.Infoln(string(b))
		}

		hostPriorityList = algorithm.Prioritize(extenderArgs)

	}

	if response, err := json.Marshal(hostPriorityList); err != nil {
		log.Errorln("json 格式化 hostPriorityList:", err)
		panic(err)
	} else {
		metrics.PodSchedulePrioritySuccesses.Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

// func Bind(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {}
