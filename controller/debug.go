package controller

import (
	"github.com/prometheus/common/log"
	"kube-scheduler-extender/conf"
	"kube-scheduler-extender/util/debugger"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

// ListenForSignal starts a goroutine that will trigger the node info
// behavior when the process receives SIGINT (Windows) or SIGUSER2 (non-Windows).
func ListenForSignal(stopCh <-chan struct{}) {
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, debugger.CompareSignal)

	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-ch:
				log.Infof("当前prometheus_url: %v, prometheus_memory_metrics: %v, prometheus_memory_threshold: %v",
					conf.Conf.PrometheusUrl, conf.Conf.PrometheusMemoryMetrics, conf.Conf.PrometheusMemoryThreshold)
				builder := strings.Builder{}

				NodeInfo.Lock.RLock()
				for nodeName, node := range NodeInfo.NodeMem {
					builder.WriteString("\nnodeName:" + nodeName + "; memoryValue:" + strconv.Itoa(node.Value) + "; checkTime:" + node.CheckTime.Format("2006-01-02 15:04:05") + ";")
				}

				info := builder.String()
				log.Infoln("cache node number: ", strconv.Itoa(len(NodeInfo.NodeMem)))
				log.Infoln("node info: ", info)

				NodeInfo.Lock.RUnlock()
			}
		}
	}()
}
