#!/usr/bin/env bash

resource="deployment,replicaset,pod"
kube_version="$1"

KUBE_VERSION="${kube_version}" ./fake-k8s create cluster --quiet-pull --generate-replicas 1
sleep 10
./fake-k8s kubectl create -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-pod
  template:
    metadata:
      labels:
        app: test-pod
    spec:
      containers:
        - name: test-container
          image: ghcr.io/wzshiming/echoserver/echoserver:v0.0.1
EOF
sleep 10

./fake-k8s kubectl get "${resource}" -n default -o yaml > tmp.yaml
old_fake_k8s_content="$(./fake-k8s kubectl get "${resource}" -n default -o name)"
./fake-k8s delete cluster


KUBE_VERSION="${kube_version}" ./fake-k8s create cluster --quiet-pull --generate-replicas 1
sleep 10
cat tmp.yaml | ./fake-k8s load resource -f -
sleep 10

fake_k8s_content="$(./fake-k8s kubectl get "${resource}" -n default -o name)"

./fake-k8s delete cluster

echo "=== old fale-k8s content ==="
echo "${old_fake_k8s_content}"

echo "=== fake-k8s content ==="
echo "${fake_k8s_content}"

echo "=== diff ==="
diff <(echo "${old_fake_k8s_content}") <(echo "${fake_k8s_content}")
