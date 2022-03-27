#!/usr/bin/env bash

releases=(
  v1.23.5
  v1.22.8
  v1.21.11
  v1.20.15
  v1.19.16
  v1.18.20
  v1.17.17
  v1.16.15
  v1.15.12
  v1.14.10
  v1.13.12
  v1.12.10
  v1.11.10
  v1.10.13
)

function test_release() {
  local release="${1}"
  local name="cluster-${release//./-}"
  ./fake-k8s.sh create --name "${name}" --kube-version "${release}"

  for _ in $(seq 1 30); do
    kubectl --context="fake-k8s-${name}" apply -f - <<EOF
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
    if kubectl --context="fake-k8s-${name}" get pod 2>/dev/null | grep Running >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  echo kubectl --context="fake-k8s-${name}" get pod
  kubectl --context="fake-k8s-${name}" get pod

  if kubectl --context="fake-k8s-${name}" get pod | grep Running >/dev/null 2>&1; then
    echo "=== release ${release} is works ==="
  else
    echo "=== release ${release} is not works ==="
    failed+=("${release}")
  fi

  ./fake-k8s.sh delete -n "${name}" >/dev/null 2>&1 &
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
