package conf

var Conf *config

type config struct {
	PrometheusUrl             string
	PrometheusMemoryMetrics   string
	PrometheusMemoryThreshold int
	LogRequestBody            bool
}

func NewConfig(PrometheusUrl, PrometheusMemoryMetrics string, PrometheusMemoryThreshold int, LogRequestBody bool) {
	Conf = &config{
		PrometheusUrl:             PrometheusUrl,
		PrometheusMemoryMetrics:   PrometheusMemoryMetrics,
		PrometheusMemoryThreshold: PrometheusMemoryThreshold,
		LogRequestBody:            LogRequestBody,
	}

}
