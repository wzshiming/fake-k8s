#!/usr/bin/env bash

resource="ns,node,statefulset,daemonset,deployment,replicaset,pod"
kube_version="v1.23.4"

kind create cluster --wait 10s --image=docker.io/kindest/node:"${kube_version}"
sleep 30

kubectl --context=kind-kind get "${resource}" -A -o json | ./fake-k8s.sh create --kube-version "${kube_version}" --mock -
sleep 5

kind_content="$(kubectl --context=kind-kind get "${resource}" -A -o name)"
fake_k8s_content="$(kubectl --context=fake-k8s-default get "${resource}" -A -o name)"

kind delete cluster

./fake-k8s.sh delete

diff <(echo "${kind_content}") <(echo "${fake_k8s_content}")
