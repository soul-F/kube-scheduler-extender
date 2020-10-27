# kube-scheduler-extender
`kube-scheduler` 组件的 HTTPExtender, 扩展原生组件不能根据主机实时负载(目前只实现了实时内存负载)调度的能力. 
 当 request resource 设置不合理时,非常容易引起集群node节点雪崩, 影响整个集群的稳定性.
 
# 实现步骤

- 在`prometheus`rules 中添加以下配置,计算内存使用百分比,使用 record,是为了提高`prometheus`性能.其中以HostMemoryUsagePercent metrics的label instance 取k8s node 节点主机名.

```
- record: HostMemoryUsagePercent
  expr:  (1 - (({__name__=~"node_memory_MemFree|node_memory_MemFree_bytes"} + {__name__=~"node_memory_Cached|node_memory_Cached_bytes"} + {__name__=~"node_memory_Buffers|node_memory_Buffers_bytes"} + {__name__=~"node_memory_Slab|node_memory_Slab_bytes"} ) / ({__name__=~"node_memory_MemTotal|node_memory_MemTotal_bytes"}))) * 100
```

- `kube-scheduler`启动文件添加配置.

```
--policy-configmap=kube-scheduler-extender
```

- `configmap` kube-scheduler-extender 内容. 
"nodeCacheCapable": true 是为了提供性能,不传递node详情(每个node详情大概15K左右),只传递 nodeName 列表.

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-scheduler-extender
  namespace: kube-system
data:
 policy.cfg : |
    {
      "kind" : "Policy",
      "apiVersion" : "v1",
      "extenders" : [{
          "urlPrefix": "http://127.0.0.1:8888/",
          "filterVerb": "filter",
          "prioritizeVerb": "prioritize",
          "weight": 1,
          "enableHttps": false,
          "nodeCacheCapable": true,
          "ignorable": true
      }]
    }

```

- 启动命令 `kube-scheduler-extender --prometheus_url="http://xx.xx.xx.xx:9090" --log.level="debug" --prometheus_memory_threshold=85` 节点内存大于85%节点将被过滤掉.

```
[root@fangyli-test kube-scheduler-extender]# ./kube-scheduler-extender  -h
usage: kube-scheduler-extender [<flags>]

Flags:
  -h, --help                    Show context-sensitive help (also try --help-long and --help-man).
      --prometheus_url="http://127.0.0.1:9090"
                                Prometheus url. (env: PROMETHEUS_URL)
      --prometheus_memory_metrics="HostMemoryUsagePercent"
                                Prometheus memory metrics. (env: PROMETHEUS_MEMORY_METRICS)
      --prometheus_memory_threshold=80
                                Prometheus memory threshold. (env: PROMETHEUS_MEMORY_THRESHOLD)
      --listen_address=":8888"  Address to listen on for web interface and telemetry. (env: LISTEN_ADDRESS)
      --log_request_body        Log k8s request body. (env: LOG_REQUEST_BODY)
      --log.level="info"        Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format="logger:stderr"
                                Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true"
```

- 效果

```
kube-scheduler-extender日志:
time="2020-10-16T16:20:27+08:00" level=debug msg="pod podName/namespaceName 调度算法前,node 数量: 36, node 详情: XX.XX.56.191,XX.XX.48.251,XX.XX.56.103,XX.XX.56.162,XX.XX.48.202..." source="predicates.go:56"
time="2020-10-16T16:20:27+08:00" level=info msg="pod podName/namespaceName 不能调度 node XX.XX.56.18,当前node 内存使用率 88%" source="predicates.go:138"
time="2020-10-16T16:20:27+08:00" level=debug msg="pod podName/namespaceName 调度算法后,node 数量: 35, node 详情: XX.XX.56.191,XX.XX.48.251,XX.XX.56.103,XX.XX.56.162,XX.XX.48.202..." source="predicates.go:101"
time="2020-10-16T16:20:27+08:00" level=debug msg="pod podName/namespaceName 优选算法, node 节点: XX.XX.56.191,XX.XX.48.251,XX.XX.56.103,XX.XX.56.162,XX.XX.48.202,XX.XX.56.223..." source="priorities.go:46"
time="2020-10-16T16:20:27+08:00" level=debug msg="执行优选算法 CheckMemoryLoad,node XX.XX.56.191,设置 Score 为 5" source="priorities.go:151"
time="2020-10-16T16:20:27+08:00" level=debug msg="执行优选算法 CheckMemoryLoad,node XX.XX.48.251,设置 Score 为 4" source="priorities.go:151"
...
time="2020-10-16T16:20:27+08:00" level=debug msg="执行优选算法 CheckMemoryLoad,node XX.XX.56.160,设置 Score 为 5" source="priorities.go:151"
time="2020-10-16T16:20:27+08:00" level=debug msg="执行优选算法 CheckMemoryLoad,node XX.XX.48.184,设置 Score 为 5" source="priorities.go:151"
time="2020-10-16T16:20:27+08:00" level=debug msg="最终得分: podName/namespaceName -> XX.XX.56.191, Score: (5)" source="priorities.go:130"
time="2020-10-16T16:20:27+08:00" level=debug msg="最终得分: podName/namespaceName -> XX.XX.48.251, Score: (4)" source="priorities.go:130"
...
time="2020-10-16T16:20:27+08:00" level=debug msg="最终得分: podName/namespaceName -> XX.XX.48.150, Score: (4)" source="priorities.go:130"
time="2020-10-16T16:20:27+08:00" level=debug msg="最终得分: podName/namespaceName -> XX.XX.56.160, Score: (5)" source="priorities.go:130"

查询pod:
kubectl get po --all-namespaces| grep pjfi66me
35568e76-1ef1-4d77-b5cf-8fb66d2c8002   pjfi66me-6777b7bc8-6hksx                   1/1     Running             0          15h
35568e76-1ef1-4d77-b5cf-8fb66d2c8002   pjfi66me-774fb75fc9-vlbdr                  0/1     Pending             0          5s

查询pod Pending原因:
Events:
  Type     Reason            Age                From               Message
  ----     ------            ----               ----               -------
  Warning  FailedScheduling  5s (x12 over 21s)  default-scheduler  0/28 nodes are available: 1 node memory load high, 27 node(s) didn't match node selector.
(1 node memory load high)就是kube-scheduler-extender返回的节点失败调度信息.
```