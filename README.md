# fake-k8s

``` console
Usage ./fake-k8s.sh
COMMAND:
  create [NAME]
  delete [NAME]
  list
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

## Get node of cluster

``` console
kubectl --context=fake-k8s-c1 get node
NAME     STATUS   ROLES   AGE  VERSION
fake-0   Ready    agent   1s   fake
fake-1   Ready    agent   1s   fake
fake-2   Ready    agent   1s   fake
fake-3   Ready    agent   1s   fake
fake-4   Ready    agent   1s   fake
```

## List cluster

``` console
./fake-k8s.sh list             
NAME                STATUS
fake-k8s-c1         running(5)
fake-k8s-c2         running(5)
```

## Delete cluster

``` console
./fake-k8s.sh delete c1
./fake-k8s.sh delete c2
```
