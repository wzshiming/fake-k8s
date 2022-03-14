# fake-k8s

``` console
Usage ./fake-k8s.sh
COMMAND:
  create [NAME]
  delete [NAME]
ARGUMENTS:
  NAME: name of the cluster
        default: 'default'
FLAGS:
  -h, --help: show this help
  -r, --replicas: number of replicas of the node
        default: '5'
  -p, --port: port of the apiserver of the cluster
        default: '8080'
```

## Cteate cluster
``` console
./fake-k8s.sh create c1 -p 8081
./fake-k8s.sh create c2 -p 8082
```

## Delete cluster

``` console
./fake-k8s.sh delete c1
./fake-k8s.sh delete c2
```
