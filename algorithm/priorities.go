package algorithm

import (
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	extender "k8s.io/kube-scheduler/extender/v1"
	"kube-scheduler-extender/controller"
	"strings"
	"sync"
)

const (
	// CheckMemoryLoadPriority 优选算法名字
	CheckMemoryLoadPriority = "CheckMemoryLoad"
)

var priorityFuncs = map[string]FitPriority{
	CheckMemoryLoadPriority: CheckMemoryLoadPriorityMap,
}

type FitPriority func(pod *v1.Pod, node v1.Node, nodeName string) (extender.HostPriority, error)

// 优选算法 list，list中的优选函数 一定 保存在 priorityFuncs 中
var prioritySorted = []string{CheckMemoryLoadPriority}

// it's webhooked to pkg/scheduler/core/generic_scheduler.go#prioritizeNodes()
// you can't see existing scores calculated so far by default scheduler
// instead, scores output by this function will be added back to default scheduler
func Prioritize(args extender.ExtenderArgs) *extender.HostPriorityList {

	if args.NodeNames == nil {
		log.Errorln("请查看policy配置,目前只支持 nodeCacheCapable: true,返回所有节点 Score: 1")
		result := make(extender.HostPriorityList, 0, len(args.Nodes.Items))
		for _, v := range args.Nodes.Items {
			result = append(result, extender.HostPriority{
				Host:  v.Name,
				Score: 1,
			})
		}

		return &result
	}

	numNode := len(*args.NodeNames)
	log.Debugf("pod %v/%v 优选算法, node 节点: %v", args.Pod.Name, args.Pod.Namespace, strings.Join(*args.NodeNames, ","))

	// 优选算法为0,则直接返回所有节点，Score = 1
	if len(prioritySorted) == 0 {
		log.Debugln("优选函数为空,跳过Prioritize,所有节点Score为1")
		result := make(extender.HostPriorityList, 0, numNode)
		for _, v := range *args.NodeNames {
			result = append(result, extender.HostPriority{
				Host:  v,
				Score: 1,
			})
		}

		return &result
	}

	var (
		mu   = sync.Mutex{}
		errs []string
	)

	appendError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, err.Error())
	}

	// 二位数组，index 是算法索引，value 是 extender.HostPriorityList，extender.HostPriorityList 中 index 是 *args.NodeNames中的 index，value 是 extender.HostPriority
	results := make([]extender.HostPriorityList, len(predicatesSorted))
	for i := range prioritySorted {
		results[i] = make(extender.HostPriorityList, numNode)
	}

	workqueue.ParallelizeUntil(nil, 16, numNode, func(index int) {
		var node v1.Node
		var pod *v1.Pod
		nodeName := (*args.NodeNames)[index]
		for i, priorityKey := range prioritySorted {
			var err error

			if priority, exist := priorityFuncs[priorityKey]; exist {
				// 优选 Map 过程
				results[i][index], err = priority(pod, node, nodeName)

				if err != nil {
					appendError(err)
					results[i][index].Host = nodeName
				}

			} else {
				// priorityFuncs没有查到对应算法，正常来说不存在这种情况
				results[i][index] = extender.HostPriority{
					Host:  nodeName,
					Score: 0,
				}
			}

		}
	})

	if len(errs) != 0 {
		log.Error("优选过程出错:", errs)
		result := make(extender.HostPriorityList, 0, numNode)
		for _, name := range *args.NodeNames {
			result = append(result, extender.HostPriority{Host: name, Score: 0})
		}
		return &result
	}

	// Summarize all scores.
	result := make(extender.HostPriorityList, 0, numNode)
	for i, name := range *args.NodeNames {
		result = append(result, extender.HostPriority{Host: name, Score: 0})

		for j := range prioritySorted {
			result[i].Score += results[j][i].Score
		}
	}

	numPriority := len(prioritySorted)

	// Reduce 过程
	for _, node := range result {
		node.Score = node.Score / int64(numPriority)
		log.Debugf("最终得分: %v/%v -> %v, Score: (%d)", args.Pod.Name, args.Pod.Namespace, node.Host, node.Score)
	}

	return &result

}

func CheckMemoryLoadPriorityMap(pod *v1.Pod, node v1.Node, nodeName string) (extender.HostPriority, error) {
	//log.Debugf("开始执行优选 %v 算法,计算 node %v 得分", CheckMemoryLoadPriority, nodeName)
	var score int64

	controller.NodeInfo.Lock.RLock()
	defer controller.NodeInfo.Lock.RUnlock()
	if n, exist := controller.NodeInfo.NodeMem[nodeName]; exist {
		score = int64((100 - n.Value) / 10)

		switch {
		case score >= extender.MaxExtenderPriority:
			score = extender.MaxExtenderPriority
		case score <= extender.MinExtenderPriority:
			score = extender.MinExtenderPriority
		}

		log.Debugf("执行优选算法 %v,node %v,设置 Score 为 %v", CheckMemoryLoadPriority, nodeName, score)

	} else {
		log.Debugf("执行优选算法 %v,node %v 缓存未命中,设置 Score 为 1", CheckMemoryLoadPriority, nodeName)
		score = 1
	}

	return extender.HostPriority{
		Host:  nodeName,
		Score: score,
	}, nil

}
