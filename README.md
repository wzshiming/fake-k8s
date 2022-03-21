# fake-k8s

fake-k8s is a tool for running Fake Kubernetes clusters using Docker.

``` console
$ time fake-k8s.sh create
[+] Running 5/5
 ⠿ Container fake-k8s-default-etcd             Started                                                         1.7s
 ⠿ Container fake-k8s-default-kube-apiserver   Started                                                         1.7s
 ⠿ Container fake-k8s-default-kube-scheduler   Started                                                         1.6s
 ⠿ Container fake-k8s-default-kube-controller  Started                                                         1.5s
 ⠿ Container fake-k8s-default-fake-kubelet     Started                                                         1.7s
Property "clusters.fake-k8s-default.server" set.
Property "contexts.fake-k8s-default.cluster" set.
Created cluster fake-k8s-default.

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

## Usage

``` console
Usage ./fake-k8s.sh
Commands:
  create    Creates one fake cluster
  delete    Deletes one fake cluster
  list      List all fake cluster
Flags:
  -h, --help                             show this help
  -n, --name string                      cluster name (default: 'default')
  -r, --replicas uint32                  number of replicas of the node (default: '5')
  -p, --port uint16                      port of the apiserver of the cluster (default: '8080')
  --fake-version string                  version of the fake image (default: 'v0.3.3')
  --kube-version string                  version of the kubernetes image (default: 'v1.19.16')
  --etcd-version string                  version of the etcd image (default: '3.4.13-0')
  --kube-image-prefix string             prefix of the kubernetes image (default: 'k8s.gcr.io')
  --fake-image-prefix string             prefix of the fake image (default: 'ghcr.io/wzshiming/fake-kubelet')
  --image-etcd string                    etcd image (default: 'k8s.gcr.io/etcd:3.4.13-0')
  --image-kube-apiserver string          kube-apiserver image (default: 'k8s.gcr.io/kube-apiserver:v1.19.16')
  --image-kube-controller-manager string kube-controller-manager image (default: 'k8s.gcr.io/kube-controller-manager:v1.19.16')
  --image-kube-scheduler string          kube-scheduler image (default: 'k8s.gcr.io/kube-scheduler:v1.19.16')
  --image-fake-kubelet string            fake-kubelet image (default: 'ghcr.io/wzshiming/fake-kubelet/fake-kubelet:v0.3.3')
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
