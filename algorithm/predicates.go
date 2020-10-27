package algorithm

import (
	"context"
	"github.com/prometheus/common/log"
	"kube-scheduler-extender/conf"
	"kube-scheduler-extender/controller"
	"kube-scheduler-extender/util"
	"strings"
	"sync"

	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	extender "k8s.io/kube-scheduler/extender/v1"
)

const (
	// CheckMemoryLoadPred rejects a node if node memory load high
	CheckMemoryLoadPred        = "CheckMemoryLoad"
	CheckMemoryLoadPredFailMsg = "node memory load high"
)

var predicatesFuncs = map[string]FitPredicate{
	CheckMemoryLoadPred: CheckMemoryLoadPredicate,
}

type FitPredicate func(pod *v1.Pod, node v1.Node, nodeName string) (bool, []string, error)

// 预选算法 list
var predicatesSorted = []string{CheckMemoryLoadPred}

// filter filters nodes according to predicates defined in this extender
// it's webhooked to pkg/scheduler/core/generic_scheduler.go#findNodesThatFitPod()
func Filter(args extender.ExtenderArgs) *extender.ExtenderFilterResult {
	var node v1.Node

	var filteredNodeNames []string
	failedNodes := make(extender.FailedNodesMap)
	pod := args.Pod

	result := extender.ExtenderFilterResult{
		NodeNames:   &filteredNodeNames,
		FailedNodes: failedNodes,
		Error:       "",
	}

	if args.NodeNames == nil {
		log.Errorln("请查看policy配置,目前只支持 nodeCacheCapable: true")
		result.Error = "请查看policy配置,目前只支持 nodeCacheCapable: true"
		return &result
	}

	numNodesToFind := len(*args.NodeNames)

	log.Debugf("pod %v/%v 调度算法前,node 数量: %v, node 详情: %v", pod.Name, pod.Namespace, len(*args.NodeNames), strings.Join(*args.NodeNames, ","))

	// 如果预选函数==0,直接返回所有节点
	if len(predicatesSorted) == 0 {
		log.Debugln("预选函数为空,跳过Filter,直接返回")
		result.NodeNames = args.NodeNames
		return &result
	} else {
		// 调度器 args 可以传递有 node详情 或者 nodeName列表 ,podFitsOnNode 为了兼容参数都提供，但是只生效一个，为了效率，这里只传递nodeName列表
		errCh := util.NewErrorChannel()
		ctx, cancel := context.WithCancel(context.Background())

		var (
			predicateResultLock sync.Mutex
		)

		checkNode := func(i int) {
			nodeName := (*args.NodeNames)[i]
			fits, failReasons, err := podFitsOnNode(pod, node, nodeName)

			if err != nil {
				errCh.SendErrorWithCancel(err, cancel)
				return
			}
			if fits {
				// 并发安全
				predicateResultLock.Lock()
				filteredNodeNames = append(filteredNodeNames, nodeName)
				predicateResultLock.Unlock()
			} else {
				predicateResultLock.Lock()
				if len(failReasons) != 0 {
					failedNodes[nodeName] = strings.Join(failReasons, ",")
				}
				predicateResultLock.Unlock()
			}
		}

		workqueue.ParallelizeUntil(ctx, 16, numNodesToFind, checkNode)

		if err := errCh.ReceiveError(); err != nil {
			result.Error = err.Error()
			return &result
		}
	}

	log.Debugf("pod %v/%v 调度算法后,node 数量: %v, node 详情: %v", pod.Name, pod.Namespace, len(*result.NodeNames), strings.Join(*result.NodeNames, ","))

	return &result
}

// 对一个 node 进行预选算法 Filter
func podFitsOnNode(pod *v1.Pod, node v1.Node, nodeName string) (bool, []string, error) {
	var failReasons []string
	// 遍历预选算法,有一个失败则直接返回,不继续执行后续预选算法
	for _, predicateKey := range predicatesSorted {
		if predicate, exist := predicatesFuncs[predicateKey]; exist {
			fit, failures, err := predicate(pod, node, nodeName)

			// 出错直接返回
			if err != nil {
				log.Errorf("预选算法 %v 检查失败,错误:%v", predicateKey, err.Error())
				return false, nil, err
			}
			// 预选失败，跳出循环
			if !fit {
				failReasons = append(failReasons, failures...)
				break
			}
		}

	}
	return len(failReasons) == 0, failReasons, nil
}

func CheckMemoryLoadPredicate(pod *v1.Pod, node v1.Node, nodeName string) (bool, []string, error) {
	//log.Debugf("开始并发执行预选 %v 算法,在 node %v 调度 pod %v/%v", CheckMemoryLoadPred, nodeName, pod.Name, pod.Namespace)
	var failReasons []string

	currentTime := time.Now()
	controller.NodeInfo.Lock.RLock()
	defer controller.NodeInfo.Lock.RUnlock()
	if n, exist := controller.NodeInfo.NodeMem[nodeName]; exist {
		// 节点内存大于调度阀值，并且检查时间小于节点数据失效时间,检查失败
		if n.Value >= conf.Conf.PrometheusMemoryThreshold && currentTime.Sub(n.CheckTime) <= controller.NodeOverdueTime {
			log.Infof("pod %v/%v 不能调度 node %v,当前node 内存使用率 %v%%", pod.Name, pod.Namespace, nodeName, n.Value)
			failReasons = append(failReasons, CheckMemoryLoadPredFailMsg)
			return false, failReasons, nil
		}
	}

	return true, nil, nil

}
