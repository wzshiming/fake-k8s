#!/usr/bin/env bash

# Gets the release from the argument
releases=()
while [[ $# -gt 0 ]]; do
  releases+=("${1}")
  shift
done

function test_release() {
  local release="${1}"
  local name="cluster-${release//./-}"
  local targets
  local i

  KUBE_VERSION="${release}" ./fake-k8s create cluster --name "${name}" --quiet-pull --prometheus-port 9090

  for ((i = 0; i < 30; i++)); do
    ./fake-k8s --name "${name}" kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fake-pod
  namespace: default
spec:
  replicas: 1
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
    if ./fake-k8s --name="${name}" kubectl get pod 2>/dev/null | grep Running >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  echo kubectl --context="fake-k8s-${name}" config view --minify
  ./fake-k8s --name="${name}" kubectl config view --minify

  echo kubectl --context="fake-k8s-${name}" get pod
  ./fake-k8s --name="${name}" kubectl get pod

  if ! ./fake-k8s --name="${name}" kubectl get pod | grep Running >/dev/null 2>&1; then
    echo "=== release ${release} is not works ==="
    failed+=("${release}_not_works")

    ./fake-k8s --name="${name}" logs kube-apiserver
  fi

  for ((i = 0; i < 30; i++)); do
    targets="$(curl -s http://127.0.0.1:9090/api/v1/targets | jq -r '.data.activeTargets[] | "\(.health) \(.scrapePool)"')"
    if [[ "$(echo "${targets}" | grep "^up " | wc -l)" -eq "6" ]]; then
      break
    fi
    sleep 1
  done
  echo "${targets}"

  if ! [[ "$(echo "${targets}" | grep "^up " | wc -l)" -eq "6" ]]; then
    echo "=== prometheus of release ${release} is not works ==="
    failed+=("${release}_prometheus_not_works")
  fi

  ./fake-k8s delete cluster --name "${name}" >/dev/null 2>&1 &
}

failed=()
for release in "${releases[@]}"; do
  time test_release "${release}"
done

if [ "${#failed[*]}" -eq 0 ]; then
  echo "=== All releases are works ==="
else
  echo "=== Failed releases: ${failed[*]} ==="
  exit 1
fi
