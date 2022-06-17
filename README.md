# fake-k8s

[![Build](https://github.com/wzshiming/fake-k8s/actions/workflows/go-cross-build.yml/badge.svg)](https://github.com/wzshiming/fake-k8s/actions/workflows/go-cross-build.yml)
[![GitHub license](https://img.shields.io/github/license/wzshiming/fake-k8s.svg)](https://github.com/wzshiming/fake-k8s/blob/master/LICENSE)

fake-k8s is a tool for running Fake Kubernetes clusters

[The old version](https://github.com/wzshiming/fake-k8s/blob/v0.1.1/fake-k8s.sh) uses shell implementation and is no longer maintained

## Usage

### Support runtime

- Binary (the default of Linux)  
  Download and start the fake cluster using the binaries,  
  Since Kubernetes officially only provides binaries for Linux, it is only the default for Linux.  
  For other than Linux you will need to compile Kubernetes binaries yourself.  
- Docker (compose v2) (the default of other than Linux)  
  Start the fake cluster using the container.  
- Nerdctl  
  ditto.  
- Kind  
  Start the fake cluster using the kind container.  

### Cteate cluster

``` console
fake-k8s create cluster --name c1
fake-k8s create cluster --name c2
```

### Simulates the specified cluster

``` console
kubectl get ns,node,statefulset,daemonset,deployment,replicaset,pod -A -o yaml > mock.yaml
fake-k8s create cluster --name m1 --generate-replicas 0
fake-k8s load resource --name m1 -f mock.yaml
```

### Get node of cluster

``` console
kubectl --context=fake-k8s-c1 get node
NAME     STATUS   ROLES   AGE  VERSION
fake-0   Ready    agent   1s   fake
fake-1   Ready    agent   1s   fake
fake-2   Ready    agent   1s   fake
fake-3   Ready    agent   1s   fake
fake-4   Ready    agent   1s   fake
```

### List cluster

``` console
fake-k8s get clusters             
c1
c2
```

### Delete cluster

``` console
fake-k8s delete cluster --name c1
fake-k8s delete cluster --name c2
```

## Examples

``` console
$ time fake-k8s create cluster
[+] Running 5/5
 ⠿ Container fake-k8s-default-etcd             Started                                                         1.7s
 ⠿ Container fake-k8s-default-kube-apiserver   Started                                                         1.7s
 ⠿ Container fake-k8s-default-kube-scheduler   Started                                                         1.6s
 ⠿ Container fake-k8s-default-kube-controller  Started                                                         1.5s
 ⠿ Container fake-k8s-default-fake-kubelet     Started                                                         1.7s

real    0m2.587s
user    0m0.418s
sys     0m0.283s

$ kubectl --context=fake-k8s-default get node -o wide
NAME     STATUS   ROLES   AGE  VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE    KERNEL-VERSION   CONTAINER-RUNTIME
fake-0   Ready    agent   2s   fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>
fake-1   Ready    agent   2s   fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>
fake-2   Ready    agent   2s   fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>
fake-3   Ready    agent   2s   fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>
fake-4   Ready    agent   2s   fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>

$ kubectl --context=fake-k8s-default apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fake-pod
  namespace: default
spec:
  replicas: 10
  selector:
    matchLabels:
      app: fake-pod
  template:
    metadata:
      labels:
        app: fake-pod
    spec:
      containers:
        - name: fake-pod
          image: fake
EOF
deployment.apps/fake-pod created

$ kubectl --context=fake-k8s-default get pod -o wide
NAME                        READY   STATUS    RESTARTS   AGE   IP          NODE     NOMINATED NODE   READINESS GATES
fake-pod-794f9d7464-246z6   1/1     Running   0          1s    10.0.0.15   fake-3   <none>           <none>
fake-pod-794f9d7464-2cvdk   1/1     Running   0          1s    10.0.0.19   fake-4   <none>           <none>
fake-pod-794f9d7464-447d5   1/1     Running   0          1s    10.0.0.5    fake-3   <none>           <none>
fake-pod-794f9d7464-5n5hv   1/1     Running   0          1s    10.0.0.13   fake-2   <none>           <none>
fake-pod-794f9d7464-8zdqp   1/1     Running   0          1s    10.0.0.17   fake-0   <none>           <none>
fake-pod-794f9d7464-bmt9q   1/1     Running   0          1s    10.0.0.7    fake-0   <none>           <none>
fake-pod-794f9d7464-lg4x8   1/1     Running   0          1s    10.0.0.9    fake-4   <none>           <none>
fake-pod-794f9d7464-ln2mg   1/1     Running   0          1s    10.0.0.3    fake-2   <none>           <none>
fake-pod-794f9d7464-mz92q   1/1     Running   0          1s    10.0.0.11   fake-1   <none>           <none>
fake-pod-794f9d7464-sgtdv   1/1     Running   0          1s    10.0.0.21   fake-1   <none>           <none>

```

## License

Licensed under the MIT License. See [LICENSE](https://github.com/wzshiming/fake-k8s/blob/master/LICENSE) for the full license text.
