package controller

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"kube-scheduler-extender/conf"
	"kube-scheduler-extender/metrics"
	"kube-scheduler-extender/util"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// 节点数据失效时间
	NodeOverdueTime = 180 * time.Second
)

var NodeInfo *Nodes

func NewNodeInfo(stopCh <-chan struct{}) {
	NodeInfo = &Nodes{
		stop:    stopCh,
		NodeMem: make(map[string]*NodeMemory),
	}

	NodeInfo.run()
}

type Nodes struct {
	stop <-chan struct{}
	Lock sync.RWMutex

	NodeMem map[string]*NodeMemory
}

type NodeMemory struct {
	NodeName string
	Value    int
	// 节点过期时间, 如果 currentTime - CheckTime > nodeOverdueTime,说明节点内存恢复正常,从NodeMems.Nodes 删除
	CheckTime time.Time
}

func (n *Nodes) run() {
	go wait.Until(n.fromPrometheusGetMemData, 60*time.Second, n.stop)
	go wait.Until(n.flushOverdueNode, 30*time.Second, n.stop)
	ListenForSignal(n.stop)
}

func (n *Nodes) flushOverdueNode() {
	currentTime := time.Now()
	n.Lock.Lock()
	for k, v := range n.NodeMem {
		if currentTime.Sub(v.CheckTime) >= NodeOverdueTime {
			log.Infoln("节点 ", k, " 数据过期,从cache中删除,", " memoryValue:"+strconv.Itoa(v.Value)+"; checkTime:"+v.CheckTime.Format("2006-01-02 15:04:05")+";")
			delete(n.NodeMem, k)
		}
	}
	n.Lock.Unlock()

	// updateMetrics
	metrics.CacheSize.WithLabelValues().Set(float64(len(n.NodeMem)))
}

type PrometheusResult struct {
	Data struct {
		Result []struct {
			Metric struct {
				Instance string `json:"instance"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
		ResultType string `json:"resultType"`
	} `json:"data"`
	Status string `json:"status"`
}

func (n *Nodes) fromPrometheusGetMemData() {
	startGetDataEvalTime := time.Now()
	defer func() {
		metrics.FromPrometheusGetDataEvaluationDuration.WithLabelValues().Observe(metrics.SinceInSeconds(startGetDataEvalTime))
	}()
	urlStr := conf.Conf.PrometheusUrl + "/api/v1/query?query=" + conf.Conf.PrometheusMemoryMetrics
	urlParse, _ := url.Parse(urlStr)
	q := urlParse.Query()
	urlParse.RawQuery = q.Encode()
	urlStr = urlParse.String()

	log.Debugln("从 prometheus 查询 node 内存信息,url: ", urlStr)

	resp, err := util.GetResponse("GET", urlStr, "", "Content-Type=application/json", "", 30*time.Second, nil)
	if err != nil {
		metrics.FromPrometheusGetDataError.WithLabelValues().Inc()
		log.Errorln("http 请求 prometheus 出错: ", err.Error())
		return
	}

	defer resp.Body.Close()

	var presult PrometheusResult
	result, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(result, &presult)
	if err != nil {
		metrics.FromPrometheusGetDataError.WithLabelValues().Inc()
		log.Errorln("json 格式化 resp.Body 出错: ", err.Error())
		return
	}

	if presult.Status == "success" {
		currentTime := time.Now()
		for _, v := range presult.Data.Result {
			int, err := strconv.Atoi(strings.Split(v.Value[1].(string), ".")[0])
			if err != nil {
				log.Errorln("prometheus 结果转换错误: ", err.Error())
			}
			// 定时任务加锁更改
			n.Lock.Lock()
			n.NodeMem[v.Metric.Instance] = &NodeMemory{
				NodeName:  v.Metric.Instance,
				Value:     int,
				CheckTime: currentTime,
			}
			n.Lock.Unlock()
		}
	} else {
		metrics.FromPrometheusGetDataError.WithLabelValues().Inc()
		log.Errorln("prometheus 查询出错")
	}

}
