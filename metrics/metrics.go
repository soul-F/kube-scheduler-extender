package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
	"time"
)

var (
	scheduleAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schedule_attempts_total",
			Help: "Number of attempts to schedule pods, by the result. 'success' means a pod could be scheduled, while 'count - success' means an internal scheduler problem.",
		}, []string{"result", "algorithm"})

	//// PodScheduleSuccesses counts how many pods were scheduled.
	//PodScheduleSuccesses = scheduleAttempts.With(prometheus.Labels{"result": "scheduled"})
	//// PodScheduleErrors counts how many pods could not be scheduled due to a scheduler error.
	//PodScheduleErrors = scheduleAttempts.With(prometheus.Labels{"result": "error"})

	// PodScheduleSuccesses counts how many pods were scheduled.
	PodSchedulePredicateSuccesses = scheduleAttempts.With(prometheus.Labels{"result": "success", "algorithm": "predicate"})
	// PodScheduleErrors counts how many pods could not be scheduled due to a scheduler error.
	PodSchedulePredicate = scheduleAttempts.With(prometheus.Labels{"result": "count", "algorithm": "predicate"})
	// PodScheduleSuccesses counts how many pods were scheduled.
	PodSchedulePrioritySuccesses = scheduleAttempts.With(prometheus.Labels{"result": "success", "algorithm": "priority"})
	// PodScheduleErrors counts how many pods could not be scheduled due to a scheduler error.
	PodSchedulePriority = scheduleAttempts.With(prometheus.Labels{"result": "count", "algorithm": "priority"})

	SchedulingAlgorithmPredicateEvaluationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scheduling_algorithm_predicate_evaluation_seconds",
			Help:    "Scheduling algorithm predicate evaluation duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{})

	SchedulingAlgorithmPriorityEvaluationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{

			Name:    "scheduling_algorithm_priority_evaluation_seconds",
			Help:    "Scheduling algorithm priority evaluation duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{})

	FromPrometheusGetDataEvaluationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{

			Name:    "from_prometheus_get_data_evaluation_seconds",
			Help:    "From prometheus get data evaluation duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}, []string{})

	FromPrometheusGetDataError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "from_prometheus_get_data_error",
			Help: "Number of attempts to from prometheus get data error.",
		}, []string{})

	CacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_cache_size",
			Help: "Number of nodes from prometheus search, in the cache.",
		}, []string{})
)

var registerMetrics sync.Once

var PrometheusHandler http.Handler

// Register all metrics.
func Register() {
	registerMetrics.Do(func() {
		registry := prometheus.NewRegistry()
		registry.MustRegister(
			scheduleAttempts,
			SchedulingAlgorithmPredicateEvaluationDuration,
			SchedulingAlgorithmPriorityEvaluationDuration,
			FromPrometheusGetDataEvaluationDuration,
			FromPrometheusGetDataError,
			CacheSize)
		PrometheusHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	})

	// PodScheduleSuccesses counts how many pods were scheduled.
	PodSchedulePredicateSuccesses = scheduleAttempts.With(prometheus.Labels{"result": "success", "algorithm": "predicate"})
	// PodScheduleErrors counts how many pods could not be scheduled due to a scheduler error.
	PodSchedulePredicate = scheduleAttempts.With(prometheus.Labels{"result": "count", "algorithm": "predicate"})
	// PodScheduleSuccesses counts how many pods were scheduled.
	PodSchedulePrioritySuccesses = scheduleAttempts.With(prometheus.Labels{"result": "success", "algorithm": "priority"})
	// PodScheduleErrors counts how many pods could not be scheduled due to a scheduler error.
	PodSchedulePriority = scheduleAttempts.With(prometheus.Labels{"result": "count", "algorithm": "priority"})

}

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}
