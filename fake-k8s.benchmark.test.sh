#!/usr/bin/env bash

kube_version="v1.23.5"

function child_timeout() {
  local to="${1}"
  shift
  "${@}" &
  local wp=$!
  local start=0
  while kill -0 "${wp}" 2>/dev/null; do
    if [[ "${start}" -ge "${to}" ]]; then
      kill "${wp}"
      echo "Timeout ${to}s" >&2
      return 1
    fi
    ((start++))
    sleep 1
  done
  echo "Took ${start}s" >&2
}

function wait_resource() {
  local name="${1}"
  local reason="${2}"
  local want="${3}"
  local raw
  local got
  local all
  while true; do
    raw="$(kubectl --context=fake-k8s-default get --no-headers "${name}" 2>/dev/null)"
    got=$(echo "${raw}" | grep -c "${reason}")
    if [ "${got}" -eq "${want}" ]; then
      echo "${name} ${got} done"
      break
    else
      all=$(echo "${raw}" | wc -l)
      echo "${name} ${got}/${all} => ${want}"
    fi
    sleep 1
  done
}

function gen_pods() {
  local size="${1}"
  for ((i = 0; i < "${size}"; i++)); do
    cat <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: fake-pod-${i}
  namespace: default
  labels:
    app: fake-pod
spec:
  containers:
  - name: fake-pod
    image: fake
  nodeName: fake-0
EOF
  done
}

function test_create_pod() {
  local size="${1}"
  gen_pods "${size}" | kubectl --context=fake-k8s-default create -f - >/dev/null 2>&1 &
  wait_resource Pod Running "${size}"
}

function test_delete_pod() {
  local size="${1}"
  kubectl --context=fake-k8s-default delete pod -l app=fake-pod --grace-period 1 >/dev/null 2>&1 &
  wait_resource Pod fake-pod- "${size}"
}

function test_create_node() {
  local size="${1}"
  wait_resource Node Ready "${size}"
}

failed=()

KUBE_VERSION="${kube_version}" ./fake-k8s create cluster --quiet-pull --generate-replicas 1

echo "=== Test create pod ==="
child_timeout 120 test_create_pod 10000 || failed+=("test_create_pod_timeout")

echo "=== Test delete pod ==="
child_timeout 120 test_delete_pod 0 || failed+=("test_delete_pod_timeout")

./fake-k8s delete cluster

KUBE_VERSION="${kube_version}" ./fake-k8s create cluster --quiet-pull --generate-replicas 10000

echo "=== Test create node ==="
child_timeout 120 test_create_node 10000 || failed+=("test_create_node_timeout")

./fake-k8s delete cluster

if [[ "${#failed[*]}" -eq 0 ]]; then
  echo "=== All tests are passed ==="
else
  echo "=== Failed tests: ${failed[*]} ==="
  exit 1
fi
