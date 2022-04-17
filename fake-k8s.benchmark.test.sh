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
  local got
  while true; do
    got=$(kubectl --context=fake-k8s-default get "${name}" | grep -c "${reason}")
    if [ "${got}" -eq "${want}" ]; then
      echo "${name} ${got} done"
      break
    else
      echo "${name} ${got}/${want}"
    fi
    sleep 1
  done
}

function gen_pods() {
  local size="${1}"
  for ((i = 0; i < ${size}; i++)); do
    cat <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  generateName: fake-pod-
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

./fake-k8s.sh create --kube-version "${kube_version}" --quiet-pull -r 1

echo "=== Test create pod ==="
child_timeout 30 test_create_pod 1000 || failed+=("test_create_pod_timeout")

echo "=== Test delete pod ==="
child_timeout 30 test_delete_pod 0 || failed+=("test_delete_pod_timeout")

./fake-k8s.sh create --kube-version "${kube_version}" --quiet-pull -r 1000

echo "=== Test create node ==="
child_timeout 30 test_create_node 1000 || failed+=("test_create_node_timeout")

./fake-k8s.sh delete

if [[ "${#failed[*]}" -eq 0 ]]; then
  echo "=== All tests are passed ==="
else
  echo "=== Failed tests: ${failed[*]} ==="
  exit 1
fi
