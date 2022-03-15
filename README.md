# fake-k8s

``` console
Usage ./fake-k8s.sh
Commands:
  create    Creates one fake cluster
  delete    Deletes one fake cluster
  list      List all fake cluster
Flags:
  -h, --help               show this help
  -n, --name string        cluster name (default: 'default')
  -r, --replicas uint32    number of replicas of the node (default: '5')
  -p, --port uint16        port of the apiserver of the cluster (default: '8080')
```

## Cteate cluster

``` console
./fake-k8s.sh create -n c1 -p 8081
./fake-k8s.sh create -n c2 -p 8082
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
./fake-k8s.sh delete -n c1
./fake-k8s.sh delete -n c2
```
