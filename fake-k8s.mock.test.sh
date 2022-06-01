#!/usr/bin/env bash

resource="ns,node,statefulset,daemonset,deployment,replicaset,pod"
kube_version="v1.23.5"

kind create cluster --wait 10s --image=docker.io/kindest/node:"${kube_version}"
sleep 30

KUBE_VERSION="${kube_version}" ./fake-k8s create cluster --quiet-pull --generate-replicas 0
kubectl --context=kind-kind get "${resource}" -A -o yaml | ./fake-k8s load resource -f -
sleep 30

kind_content="$(kubectl --context=kind-kind get "${resource}" -A -o name)"
fake_k8s_content="$(kubectl --context=fake-k8s-default get "${resource}" -A -o name)"

kind delete cluster

./fake-k8s delete cluster

echo "=== kind content ==="
echo "${kind_content}"

echo "=== fake-k8s content ==="
echo "${fake_k8s_content}"

echo "=== diff ==="
diff <(echo "${kind_content}") <(echo "${fake_k8s_content}")
