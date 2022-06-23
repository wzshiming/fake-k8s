#!/bin/sh

APISERVER_PORT=0 fake-k8s create cluster || exit 1

echo "==============================" >&2
cat <<EOF
apiVersion: v1
clusters:
- cluster:
    server: http://127.0.0.1:${APISERVER_PORT}
  name: fake-k8s
contexts:
- context:
    cluster: fake-k8s
  name: fake-k8s
current-context: fake-k8s
kind: Config
preferences: {}
users: null
EOF
echo "==============================" >&2
echo "kubectl -s :${APISERVER_PORT} get node"
echo "==============================" >&2
fake-k8s kubectl version
echo "==============================" >&2
fake-k8s kubectl proxy --port="${APISERVER_PORT}" --accept-hosts='^*$' --address="0.0.0.0" 
