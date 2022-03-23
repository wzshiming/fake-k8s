#!/usr/bin/env bash

declare -A etcd_versions=(
  ["8"]="3.0.17"
  ["9"]="3.1.12"
  ["10"]="3.1.12"
  ["11"]="3.2.18"
  ["12"]="3.2.24"
  ["13"]="3.2.24"
  ["14"]="3.3.10"
  ["15"]="3.3.10"
  ["16"]="3.3.17-0"
  ["17"]="3.4.3-0"
  ["18"]="3.4.3-0"
  ["19"]="3.4.13-0"
  ["20"]="3.4.13-0"
  ["21"]="3.4.13-0"
  ["22"]="3.5.1-0"
  ["23"]="3.5.1-0"
  ["24"]="3.5.1-0"
)

function get_release_version() {
  local version="${1}"
  local release_version
  release_version="${version#*.}"
  release_version="${release_version%%.*}"
  echo "${release_version}"
}

function get_etcd_version() {
  local kube_version="${1}"
  local release_version
  local etcd_version
  release_version="$(get_release_version "${kube_version}")"
  if [[ "${release_version}" -gt "24" ]]; then
    etcd_version="3.5.1-0"
  elif [[ "${release_version}" -lt "8" ]]; then
    etcd_version="3.0.17"
  else
    etcd_version="${etcd_versions[${release_version}]}"
  fi
  echo "${etcd_version}"
}

function init_global_flags() {
  FAKE_VERSION="${FAKE_VERSION:-v0.3.3}"
  KUBE_VERSION="${KUBE_VERSION:-v1.19.16}"
  ETCD_VERSION="${ETCD_VERSION:-$(get_etcd_version "${KUBE_VERSION}")}"

  KUBE_IMAGE_PREFIX="${KUBE_IMAGE_PREFIX:-k8s.gcr.io}"
  FAKE_IMAGE_PREFIX="${FAKE_IMAGE_PREFIX:-ghcr.io/wzshiming/fake-kubelet}"
  IMAGE_ETCD="${IMAGE_ETCD:-${KUBE_IMAGE_PREFIX}/etcd:${ETCD_VERSION}}"
  IMAGE_KUBE_APISERVER="${IMAGE_KUBE_APISERVER:-${KUBE_IMAGE_PREFIX}/kube-apiserver:${KUBE_VERSION}}"
  IMAGE_KUBE_CONTROLLER_MANAGER="${IMAGE_KUBE_CONTROLLER_MANAGER:-${KUBE_IMAGE_PREFIX}/kube-controller-manager:${KUBE_VERSION}}"
  IMAGE_KUBE_SCHEDULER="${IMAGE_KUBE_SCHEDULER:-${KUBE_IMAGE_PREFIX}/kube-scheduler:${KUBE_VERSION}}"
  IMAGE_FAKE_KUBELET="${IMAGE_FAKE_KUBELET:-${FAKE_IMAGE_PREFIX}/fake-kubelet:${FAKE_VERSION}}"
}

function incluster_kubeconfig_tls() {
  local name="${1}"
  local admin_crt_path="${2}"
  local admin_key_path="${3}"
  local ca_crt_path="${4}"
  cat <<EOF
apiVersion: v1
kind: Config
preferences: {}
current-context: ${name}
clusters:
  - cluster:
      server: https://${name}-kube-apiserver:6443
      certificate-authority-data: $(cat ${ca_crt_path} | base64 | tr -d '\n')
    name: ${name}
contexts:
  - context:
      cluster: ${name}
      user: ${name}
    name: ${name}
users:
  - name: ${name}
    user:
      client-certificate-data: $(cat ${admin_crt_path} | base64 | tr -d '\n')
      client-key-data: $(cat ${admin_key_path} | base64 | tr -d '\n')
EOF
}

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

function docker_compose_file_tls() {
  local name="${1}"
  local port="${2}"
  local replicas="${3}"
  local kubeconfig_path="${4}"
  local admin_crt_path="${5}"
  local admin_key_path="${6}"
  local ca_crt_path="${7}"

  cat <<EOF
version: "3.1"
services:
  etcd:
    container_name: "${name}-etcd"
    image: ${IMAGE_ETCD}
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
    image: ${IMAGE_KUBE_APISERVER}
    restart: always
    ports:
      - ${port}:6443
    command:
      - kube-apiserver
      - --admission-control
      - ""
      - --etcd-servers
      - http://${name}-etcd:2379
      - --etcd-prefix
      - /prefix/registry
      - --default-watch-cache-size
      - "10000"
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "6443"
      - --tls-cert-file
      - /etc/kubernetes/pki/apiserver.crt
      - --tls-private-key-file
      - /etc/kubernetes/pki/apiserver.key
      - --client-ca-file
      - /etc/kubernetes/pki/ca.crt
      - --service-account-key-file
      - /etc/kubernetes/pki/apiserver.key
      - --service-account-signing-key-file
      - /etc/kubernetes/pki/apiserver.key
      - --service-account-issuer
      - https://kubernetes.default.svc.cluster.local
    links:
      - etcd
    configs:
      - source: admin-crt
        target: /etc/kubernetes/pki/apiserver.crt
      - source: admin-key
        target: /etc/kubernetes/pki/apiserver.key
      - source: ca-crt
        target: /etc/kubernetes/pki/ca.crt

  kube_controller:
    container_name: "${name}-kube-controller"
    image: ${IMAGE_KUBE_CONTROLLER_MANAGER}
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
    image: ${IMAGE_KUBE_SCHEDULER}
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
    image: ${IMAGE_FAKE_KUBELET}
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
  admin-crt:
    file: ${admin_crt_path}
  admin-key:
    file: ${admin_key_path}
  ca-crt:
    file: ${ca_crt_path}

networks:
  default:
    name: ${name}

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
    image: ${IMAGE_ETCD}
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
    image: ${IMAGE_KUBE_APISERVER}
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
    image: ${IMAGE_KUBE_CONTROLLER_MANAGER}
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
    image: ${IMAGE_KUBE_SCHEDULER}
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
    image: ${IMAGE_FAKE_KUBELET}
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
    name: ${name}
EOF
}

function gen_cert() {
  local name="${1}"
  local dir="${2}"
  if [[ ! -f "${dir}/openssl.cnf" ]]; then
    cat <<EOF >"${dir}/openssl.cnf"
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = kubernetes
DNS.2 = kubernetes.default
DNS.3 = kubernetes.default.svc
DNS.4 = kubernetes.default.svc.cluster.local
DNS.5 = ${name}-kube-apiserver
IP.1 = 127.0.0.1
EOF
  fi

  if [[ ! -f "${dir}/ca.key" ]]; then
    openssl genrsa -out "${dir}/ca.key" 2048
  fi

  if [[ ! -f "${dir}/ca.crt" ]]; then
    openssl req -x509 -new -nodes -key "${dir}/ca.key" -subj "/CN=fake-ca" -out "${dir}/ca.crt" -days 36500
  fi

  if [[ ! -f "${dir}/admin.key" ]]; then
    openssl genrsa -out "${dir}/admin.key" 2048
  fi

  if [[ ! -f "${dir}/admin.crt" ]]; then
    if [[ ! -f "${dir}/admin.csr" ]]; then
      openssl req -new -key "${dir}/admin.key" -subj "/CN=fake-admin" -out "${dir}/admin.csr" -config "${dir}/openssl.cnf"
    fi
    openssl x509 -req -in "${dir}/admin.csr" -CA "${dir}/ca.crt" -CAkey "${dir}/ca.key" -CAcreateserial -out "${dir}/admin.crt" -days 36500 -extensions v3_req -extfile "${dir}/openssl.cnf"
  fi
}

function set_default_kubeconfig_tls() {
  local name="${1}"
  local port="${2}"
  local admin_crt_path="${3}"
  local admin_key_path="${4}"
  local ca_crt_path="${5}"
  kubectl config set "clusters.${name}.server" "https://127.0.0.1:${port}"
  kubectl config set "clusters.${name}.certificate-authority-data" "$(cat "${ca_crt_path}" | base64 | tr -d '\n')"
  kubectl config set "contexts.${name}.cluster" "${name}"
  kubectl config set "contexts.${name}.user" "${name}"
  kubectl config set "users.${name}.client-certificate-data" "$(cat "${admin_crt_path}" | base64 | tr -d '\n')"
  kubectl config set "users.${name}.client-key-data" "$(cat "${admin_key_path}" | base64 | tr -d '\n')"
}

function set_default_kubeconfig() {
  local name="${1}"
  local port="${2}"
  kubectl config set "clusters.${name}.server" "http://127.0.0.1:${port}"
  kubectl config set "contexts.${name}.cluster" "${name}"
}

function create_cluster() {
  local name="${1}"
  local port="${2}"
  local replicas="${3}"
  local full_name="fake-k8s-${name}"
  local tmpdir="${TMPDIR}/fake-k8s/${name}"
  local kube_version

  kube_version="$(get_release_version "${KUBE_VERSION}")"
  mkdir -p "${tmpdir}"

  if [[ "${kube_version}" -ge "20" ]]; then
    gen_cert "${full_name}" "${tmpdir}"

    incluster_kubeconfig_tls "${full_name}" "${tmpdir}/admin.crt" "${tmpdir}/admin.key" "${tmpdir}/ca.crt" >"${tmpdir}/kubeconfig"
    docker_compose_file_tls "${full_name}" "${port}" "${replicas}" "${tmpdir}/kubeconfig" "${tmpdir}/admin.crt" "${tmpdir}/admin.key" "${tmpdir}/ca.crt" >"${tmpdir}/docker-compose.yaml"
  else
    incluster_kubeconfig "${full_name}" >"${tmpdir}/kubeconfig"
    docker_compose_file "${full_name}" "${port}" "${replicas}" "${tmpdir}/kubeconfig" >"${tmpdir}/docker-compose.yaml"
  fi

  docker compose -p "${full_name}" -f "${tmpdir}/docker-compose.yaml" up -d

  if [[ "${kube_version}" -ge "20" ]]; then
    set_default_kubeconfig_tls "${full_name}" "${port}" "${tmpdir}/admin.crt" "${tmpdir}/admin.key" "${tmpdir}/ca.crt"
  else
    set_default_kubeconfig "${full_name}" "${port}"
  fi

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

  kubectl config delete-context "${full_name}"
  kubectl config delete-cluster "${full_name}"
  kubectl config delete-user "${full_name}"

  docker compose -p "${full_name}" down

  echo "Deleted cluster ${full_name}."
}

function list_cluster() {
  docker compose ls --all --filter=name=fake-k8s-
}

TMPDIR="${TMPDIR:-/tmp/}"

function usage() {
  init_global_flags
  echo "Usage $0"
  echo "Commands:"
  echo "  create    Creates one fake cluster"
  echo "  delete    Deletes one fake cluster"
  echo "  list      List all fake cluster"
  echo "Flags:"
  echo "  -h, --help                             show this help"
  echo "  -n, --name string                      cluster name (default: 'default')"
  echo "  -r, --replicas uint32                  number of replicas of the node (default: '5')"
  echo "  -p, --port uint16                      port of the apiserver of the cluster (default: '8080')"
  echo "  --fake-version string                  version of the fake image (default: '${FAKE_VERSION}')"
  echo "  --kube-version string                  version of the kubernetes image (default: '${KUBE_VERSION}')"
  echo "  --etcd-version string                  version of the etcd image (default: '${ETCD_VERSION}')"
  echo "  --kube-image-prefix string             prefix of the kubernetes image (default: '${KUBE_IMAGE_PREFIX}')"
  echo "  --fake-image-prefix string             prefix of the fake image (default: '${FAKE_IMAGE_PREFIX}')"
  echo "  --image-etcd string                    etcd image (default: '${IMAGE_ETCD}')"
  echo "  --image-kube-apiserver string          kube-apiserver image (default: '${IMAGE_KUBE_APISERVER}')"
  echo "  --image-kube-controller-manager string kube-controller-manager image (default: '${IMAGE_KUBE_CONTROLLER_MANAGER}')"
  echo "  --image-kube-scheduler string          kube-scheduler image (default: '${IMAGE_KUBE_SCHEDULER}')"
  echo "  --image-fake-kubelet string            fake-kubelet image (default: '${IMAGE_FAKE_KUBELET}')"
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
    --fake-version | --fake-version=*)
      [[ "${key#*=}" != "$key" ]] && FAKE_VERSION="${key#*=}" || { FAKE_VERSION="$2" && shift; }
      ;;
    --kube-version | --kube-version=*)
      [[ "${key#*=}" != "$key" ]] && KUBE_VERSION="${key#*=}" || { KUBE_VERSION="$2" && shift; }
      ;;
    --etcd-version | --etcd-version=*)
      [[ "${key#*=}" != "$key" ]] && ETCD_VERSION="${key#*=}" || { ETCD_VERSION="$2" && shift; }
      ;;
    --kube-image-prefix | --kube-image-prefix=*)
      [[ "${key#*=}" != "$key" ]] && KUBE_IMAGE_PREFIX="${key#*=}" || { KUBE_IMAGE_PREFIX="$2" && shift; }
      ;;
    --fake-image-prefix | --fake-image-prefix=*)
      [[ "${key#*=}" != "$key" ]] && FAKE_IMAGE_PREFIX="${key#*=}" || { FAKE_IMAGE_PREFIX="$2" && shift; }
      ;;
    --image-etcd | --image-etcd=*)
      [[ "${key#*=}" != "$key" ]] && IMAGE_ETCD="${key#*=}" || { IMAGE_ETCD="$2" && shift; }
      ;;
    --image-kube-apiserver | --image-kube-apiserver=*)
      [[ "${key#*=}" != "$key" ]] && IMAGE_KUBE_APISERVER="${key#*=}" || { IMAGE_KUBE_APISERVER="$2" && shift; }
      ;;
    --image-kube-controller-manager | --image-kube-controller-manager=*)
      [[ "${key#*=}" != "$key" ]] && IMAGE_KUBE_CONTROLLER_MANAGER="${key#*=}" || { IMAGE_KUBE_CONTROLLER_MANAGER="$2" && shift; }
      ;;
    --image-kube-scheduler | --image-kube-scheduler=*)
      [[ "${key#*=}" != "$key" ]] && IMAGE_KUBE_SCHEDULER="${key#*=}" || { IMAGE_KUBE_SCHEDULER="$2" && shift; }
      ;;
    --image-fake-kubelet | --image-fake-kubelet=*)
      [[ "${key#*=}" != "$key" ]] && IMAGE_FAKE_KUBELET="${key#*=}" || { IMAGE_FAKE_KUBELET="$2" && shift; }
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

  init_global_flags

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
