#!/usr/bin/env bash

function incluster_kubeconfig() {
  local name="${1}"
  cat <<EOF
apiVersion: v1
kind: Config
clusters:
  - cluster:
      server: http://${name}-kube-apiserver:8080
    name: test
contexts:
  - context:
      cluster: test
    name: test
current-context: test
preferences: {}
EOF
}

function docker_compose_file() {
  local name="${1}"
  local port="${2}"
  local replicas="${3}"
  local kubeconfig_path="${4}"
  cat <<EOF
version: "3.1"
services:
  etcd:
    container_name: "${name}-etcd"
    image: docker.io/wzshiming/etcd:v3.4.3
    restart: always
    command:
      - etcd
      - --data-dir
      - /etcd-data
      - --name
      - node0
      - --initial-advertise-peer-urls
      - http://0.0.0.0:2380
      - --listen-peer-urls
      - http://0.0.0.0:2380
      - --advertise-client-urls
      - http://0.0.0.0:2379
      - --listen-client-urls
      - http://0.0.0.0:2379
      - --initial-cluster
      - node0=http://0.0.0.0:2380

  kube_apiserver:
    container_name: "${name}-kube-apiserver"
    image: docker.io/wzshiming/kube-apiserver:v1.18.8
    restart: always
    ports:
      - ${port}:8080
    command:
      - kube-apiserver
      - --admission-control
      - ""
      - --etcd-servers
      - http://${name}-etcd:2379
      - --etcd-prefix
      - /prefix/registry
      - --insecure-bind-address
      - 0.0.0.0
      - --insecure-port
      - "8080"
      - --default-watch-cache-size
      - "10000"
    links:
      - etcd

  kube_controller:
    container_name: "${name}-kube-controller"
    image: docker.io/wzshiming/kube-controller-manager:v1.18.8
    restart: always
    command:
      - kube-controller-manager
      - --kubeconfig
      - /root/.kube/config
    links:
      - kube_apiserver
    configs:
      - source: kubeconfig
        target: /root/.kube/config

  kube_scheduler:
    container_name: "${name}-kube-scheduler"
    image: docker.io/wzshiming/kube-scheduler:v1.18.8
    restart: always
    command:
      - kube-scheduler
      - --kubeconfig
      - /root/.kube/config
    links:
      - kube_apiserver
    configs:
      - source: kubeconfig
        target: /root/.kube/config

  fake_kubelet:
    container_name: "${name}-fake-kubelet"
    image: ghcr.io/wzshiming/fake-kubelet/fake-kubelet:v0.3.3
    restart: always
    command:
      - --kubeconfig
      - /root/.kube/config
    links:
      - kube_apiserver
    configs:
      - source: kubeconfig
        target: /root/.kube/config
    environment:
      NODE_NAME: ""
      GENERATE_NODE_NAME: fake-
      GENERATE_REPLICAS: "${replicas}"
      CIDR: 10.0.0.1/24
      NODE_TEMPLATE: |-
        apiVersion: v1
        kind: Node
        metadata:
          annotations:
            node.alpha.kubernetes.io/ttl: "0"
          labels:
            app: fake-kubelet
            beta.kubernetes.io/arch: amd64
            beta.kubernetes.io/os: linux
            kubernetes.io/arch: amd64
            kubernetes.io/hostname: {{ .metadata.name }}
            kubernetes.io/os: linux
            kubernetes.io/role: agent
            node-role.kubernetes.io/agent: ""
            type: fake-kubelet
          name: {{ .metadata.name }}
      NODE_INITIALIZATION_TEMPLATE: |-
        addresses:
        - address: {{ NodeIP }}
          type: InternalIP
        allocatable:
          cpu: 1k
          memory: 1Ti
          pods: 1M
        capacity:
          cpu: 1k
          memory: 1Ti
          pods: 1M
        daemonEndpoints:
          kubeletEndpoint:
            Port: 0
        nodeInfo:
          architecture: amd64
          bootID: ""
          containerRuntimeVersion: ""
          kernelVersion: ""
          kubeProxyVersion: ""
          kubeletVersion: fake
          machineID: ""
          operatingSystem: Linux
          osImage: ""
          systemUUID: ""
        phase: Running

configs:
  kubeconfig:
    file: ${kubeconfig_path}

networks:
  default:
    external:
      name: ${name}

EOF
}

function create_cluster() {
  local name="${1}"
  local port="${2}"
  local replicas="${3}"
  local full_name="fake-k8s-${name}"
  local tmpdir="${TMPDIR}/fake-k8s/${name}"
  mkdir -p "${tmpdir}"
  incluster_kubeconfig "${full_name}" >"${tmpdir}/kubeconfig"
  docker_compose_file "${full_name}" "${port}" "${replicas}" "${tmpdir}/kubeconfig" >"${tmpdir}/docker-compose.yaml"

  docker network create "${full_name}"

  docker-compose -p "${full_name}" -f "${tmpdir}/docker-compose.yaml" up -d

  kubectl config set "clusters.${full_name}.server" "http://127.0.0.1:${port}"
  kubectl config set "contexts.${full_name}.cluster" "${full_name}"

  echo "kubectl --context=${full_name} get node"
  for i in $(seq 1 10); do
    kubectl --context="${full_name}" get node >/dev/null 2>&1 && break
    sleep 1
  done
  kubectl --context="${full_name}" get node
  echo "Created cluster ${full_name}."
}

function delete_cluster() {
  local name="${1}"
  local full_name="fake-k8s-${name}"

  docker-compose -p "${full_name}" down

  docker network rm "${full_name}"

  kubectl config delete-context "${full_name}"
  kubectl config delete-cluster "${full_name}"

  echo "Deleted cluster ${full_name}."
}

function list_cluster() {
  docker-compose ls --all --filter=name=fake-k8s-
}

TMPDIR="${TMPDIR:-/tmp/}"

function usage() {
  echo "Usage $0"
  echo "Commands:"
  echo "  create    Creates one fake cluster"
  echo "  delete    Deletes one fake cluster"
  echo "  list      List all fake cluster"
  echo "Flags:"
  echo "  -h, --help               show this help"
  echo "  -n, --name string        cluster name (default: 'default')"
  echo "  -r, --replicas uint32    number of replicas of the node (default: '5')"
  echo "  -p, --port uint16        port of the apiserver of the cluster (default: '8080')"
}

function main() {
  if [[ "$#" -eq 0 ]]; then
    usage
    return 1
  fi

  local replicas="5"
  local port="8080"
  local name="default"
  local args=()

  while [[ $# -gt 0 ]]; do
    key="$1"
    case ${key} in
    -r | -r=* | --replicas | --replicas=*)
      [[ "${key#*=}" != "$key" ]] && replicas="${key#*=}" || { replicas="$2" && shift; }
      ;;
    -p | -p=* | --port | --port=*)
      [[ "${key#*=}" != "$key" ]] && port="${key#*=}" || { port="$2" && shift; }
      ;;
    -n | -n=* | --name | --name=*)
      [[ "${key#*=}" != "$key" ]] && name="${key#*=}" || { name="$2" && shift; }
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      args+=("${key}")
      ;;
    esac
    shift
  done

  if [[ "${#args[*]}" -eq 0 ]]; then
    usage
    return 1
  fi

  local command="${args[0]}"

  case "${command}" in
  "create")
    create_cluster "${name}" "${port}" "${replicas}"
    ;;
  "delete")
    delete_cluster "${name}"
    ;;
  "list")
    list_cluster
    ;;
  *)
    usage
    return 1
    ;;
  esac
}

main "${@}"
