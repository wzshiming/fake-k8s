#!/usr/bin/env bash

# initialize the global variables
function init_global_flags() {
  PROJECT_NAME="fake-k8s"
  TMPDIR="${TMPDIR:-/tmp}/${PROJECT_NAME}"
  TMPDIR="${TMPDIR//\/\//\/}"

  RUNTIME="${RUNTIME:-$(detection_runtime)}"

  # take content from the file or stdin
  MOCK_FILENAME="${MOCK_FILENAME:-}"
  if [[ "${MOCK_FILENAME}" != "" ]]; then
    if [[ "${MOCK_FILENAME}" == "-" ]]; then
      MOCK_CONTENT="$(cat)"
    else
      MOCK_CONTENT="$(cat "${MOCK_FILENAME}")"
    fi
    if [[ "$(echo "${MOCK_CONTENT}" | jq -r '.kind')" == "List" ]]; then
      MOCK_CONTENT="$(echo "${MOCK_CONTENT}" | jq '.items | .[]')"
    fi
    NODE_NAME="${NODE_NAME:-$(echo "${MOCK_CONTENT}" | jq -r 'select( .kind == "Node" ) | .metadata.name' | tr '\n' ',' | sed 's/,$//')}"
    GENERATE_NODE_NAME="${GENERATE_NODE_NAME:-}"
    GENERATE_REPLICAS="${GENERATE_REPLICAS:-0}"
  else
    NODE_NAME="${NODE_NAME:-}"
    GENERATE_NODE_NAME="${GENERATE_NODE_NAME:-fake-}"
    GENERATE_REPLICAS="${GENERATE_REPLICAS:-5}"
  fi

  FAKE_VERSION="${FAKE_VERSION:-v0.3.4}"
  KUBE_VERSION="${KUBE_VERSION:-v1.19.16}"
  ETCD_VERSION="${ETCD_VERSION:-$(get_etcd_version "${KUBE_VERSION}")}"

  # kubernetes v1.20 secure port must be enabled
  if [[ "$(get_release_version "${KUBE_VERSION}")" -ge "20" ]]; then
    SECURE_PORT="${SECURE_PORT:-true}"
  else
    SECURE_PORT="${SECURE_PORT:-false}"
  fi

  KUBE_IMAGE_PREFIX="${KUBE_IMAGE_PREFIX:-k8s.gcr.io}"
  FAKE_IMAGE_PREFIX="${FAKE_IMAGE_PREFIX:-ghcr.io/wzshiming/fake-kubelet}"
  IMAGE_ETCD="${IMAGE_ETCD:-${KUBE_IMAGE_PREFIX}/etcd:${ETCD_VERSION}}"
  IMAGE_KUBE_APISERVER="${IMAGE_KUBE_APISERVER:-${KUBE_IMAGE_PREFIX}/kube-apiserver:${KUBE_VERSION}}"
  IMAGE_KUBE_CONTROLLER_MANAGER="${IMAGE_KUBE_CONTROLLER_MANAGER:-${KUBE_IMAGE_PREFIX}/kube-controller-manager:${KUBE_VERSION}}"
  IMAGE_KUBE_SCHEDULER="${IMAGE_KUBE_SCHEDULER:-${KUBE_IMAGE_PREFIX}/kube-scheduler:${KUBE_VERSION}}"
  IMAGE_FAKE_KUBELET="${IMAGE_FAKE_KUBELET:-${FAKE_IMAGE_PREFIX}/fake-kubelet:${FAKE_VERSION}}"
}

# Etcd version of each kubernetes version
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

# get the etcd version of kubernetes version
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

# check command exists
function command_exist() {
  local command="${1}"
  type "${command}" >/dev/null 2>&1
}

# check the string is true
function is_true() {
  local bool="${1}"
  case "${bool}" in
  T* | t* | Y* | y* | 1 | on)
    return 0
    ;;
  F* | f* | N* | n* | 0 | off)
    return 1
    ;;
  *)
    return 2
    ;;
  esac
}

# get unused local port
function unusad_port() {
  local low_bound=45536
  local range=20000
  local candidate
  for _ in $(seq 1 1000); do
    candidate="$((low_bound + (RANDOM % range)))"
    # just works on bash
    if ! (echo "" >/dev/tcp/127.0.0.1/${candidate}) >/dev/null 2>&1; then
      echo ${candidate}
      return 0
    fi
  done
  # fallback to low_bound port if no unusad port is available
  echo "${low_bound}"
}

# get the minor of version
function get_release_version() {
  local version="${1}"
  local release_version
  release_version="${version#*.}"
  release_version="${release_version%%.*}"
  echo "${release_version}"
}

# build the kubeconfig file with the given context
function build_kubeconfig() {
  local address="${1}"
  local admin_crt_path="${2}"
  local admin_key_path="${3}"
  local ca_crt_path="${4}"
  cat <<EOF
apiVersion: v1
kind: Config
preferences: {}
current-context: ${PROJECT_NAME}
clusters:
  - name: ${PROJECT_NAME}
    cluster:
      server: ${address}
EOF
  if [[ "${admin_key_path}" != "" ]]; then
    cat <<EOF
      certificate-authority-data: $(cat ${ca_crt_path} | base64 | tr -d '\n')
EOF
  fi
  cat <<EOF
contexts:
  - name: ${PROJECT_NAME}
    context:
      cluster: ${PROJECT_NAME}
EOF
  if [[ "${admin_key_path}" != "" ]]; then
    cat <<EOF
      user: ${PROJECT_NAME}
users:
  - name: ${PROJECT_NAME}
    user:
      client-certificate-data: $(cat ${admin_crt_path} | base64 | tr -d '\n')
      client-key-data: $(cat ${admin_key_path} | base64 | tr -d '\n')
EOF
  fi
}

# build the docker-compose file with the given context
function build_compose() {
  local name="${1}"
  local port="${2}"
  local kubeconfig_path="${3}"
  local admin_crt_path="${4}"
  local admin_key_path="${5}"
  local ca_crt_path="${6}"

  cat <<EOF
version: "3.1"
services:
  etcd:
    container_name: "${name}-etcd"
    image: ${IMAGE_ETCD}
    restart: unless-stopped
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
    restart: unless-stopped
    links:
      - etcd
    ports:
EOF

  if [[ "${admin_key_path}" != "" ]]; then
    cat <<EOF
      - ${port}:6443
EOF
  else
    cat <<EOF
      - ${port}:8080
EOF
  fi

  cat <<EOF
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
      - --allow-privileged
EOF

  if [[ "${admin_key_path}" != "" ]]; then
    cat <<EOF
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
    configs:
      - source: admin-crt
        target: /etc/kubernetes/pki/apiserver.crt
      - source: admin-key
        target: /etc/kubernetes/pki/apiserver.key
      - source: ca-crt
        target: /etc/kubernetes/pki/ca.crt
EOF
  else
    cat <<EOF
      - --insecure-bind-address
      - 0.0.0.0
      - --insecure-port
      - "8080"
EOF
  fi

  cat <<EOF
  kube_controller:
    container_name: "${name}-kube-controller"
    image: ${IMAGE_KUBE_CONTROLLER_MANAGER}
    restart: unless-stopped
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
    restart: unless-stopped
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
    restart: unless-stopped
    command:
      - --kubeconfig
      - /root/.kube/config
    links:
      - kube_apiserver
    configs:
      - source: kubeconfig
        target: /root/.kube/config
    environment:
      NODE_NAME: "${NODE_NAME}"
      GENERATE_NODE_NAME: "${GENERATE_NODE_NAME}"
      GENERATE_REPLICAS: "${GENERATE_REPLICAS}"
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
EOF

  if [[ "${admin_key_path}" != "" ]]; then
    cat <<EOF
  admin-crt:
    file: ${admin_crt_path}
  admin-key:
    file: ${admin_key_path}
  ca-crt:
    file: ${ca_crt_path}
EOF
  fi
  cat <<EOF
networks:
  default:
    name: ${name}
EOF
}

# Generate the certificates for the kube-apiserver.
function gen_cert() {
  local name="${1}"
  local dir="${2}"

  # Generate the ca private key and certificate.
  if [[ ! -f "${dir}/ca.key" ]]; then
    openssl genrsa -out "${dir}/ca.key" 2048
  fi
  if [[ ! -f "${dir}/ca.crt" ]]; then
    openssl req -x509 -new -nodes -key "${dir}/ca.key" -subj "/CN=fake-ca" -out "${dir}/ca.crt" -days 36500
  fi

  # Generate the admin private key and certificate signing request and sign it with the ca.
  if [[ ! -f "${dir}/admin.key" ]]; then
    openssl genrsa -out "${dir}/admin.key" 2048
  fi
  if [[ ! -f "${dir}/admin.crt" ]]; then
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
    if [[ ! -f "${dir}/admin.csr" ]]; then
      openssl req -new -key "${dir}/admin.key" -subj "/CN=fake-admin" -out "${dir}/admin.csr" -config "${dir}/openssl.cnf"
    fi
    openssl x509 -req -in "${dir}/admin.csr" -CA "${dir}/ca.crt" -CAkey "${dir}/ca.key" -CAcreateserial -out "${dir}/admin.crt" -days 36500 -extensions v3_req -extfile "${dir}/openssl.cnf"
  fi
}

# set the context in default kubeconfig
function set_default_kubeconfig() {
  local name="${1}"
  local port="${2}"
  local admin_crt_path="${3}"
  local admin_key_path="${4}"
  local ca_crt_path="${5}"

  if [[ "${admin_key_path}" != "" ]]; then
    kubectl config set "clusters.${name}.server" "https://127.0.0.1:${port}"
    kubectl config set "clusters.${name}.certificate-authority-data" "$(cat "${ca_crt_path}" | base64 | tr -d '\n')"
    kubectl config set "users.${name}.client-certificate-data" "$(cat "${admin_crt_path}" | base64 | tr -d '\n')"
    kubectl config set "users.${name}.client-key-data" "$(cat "${admin_key_path}" | base64 | tr -d '\n')"
    kubectl config set "contexts.${name}.user" "${name}"
    kubectl config set "contexts.${name}.cluster" "${name}"
  else
    kubectl config set "clusters.${name}.server" "http://127.0.0.1:${port}"
    kubectl config set "contexts.${name}.cluster" "${name}"
  fi
}

# unset the context in default kubeconfig
function unset_default_kubeconfig() {
  local name="${1}"
  kubectl config delete-context "${name}"
  kubectl config delete-cluster "${name}"
  kubectl config delete-user "${name}"
}

# detect the runtime for the current system
function detection_runtime() {
  if command_exist docker; then
    echo docker
  elif command_exist nerdctl; then
    echo nerdctl
  else
    echo docker
  fi
}

# get resource kind full name for kubernetes resource
function get_resource_kind() {
  local api_version="${1}"
  local kind="${2}"
  if [[ "${api_version}" =~ / ]]; then
    echo "${kind}.${api_version%%/*}"
  else
    echo "${kind}"
  fi
}

# import to cluster from given resource and modify the resource uid
function mock_resource() {
  local kubeconfig="${1}"
  local resource="${2}"
  local other_resource="${3}"
  local apply_resource
  local new_resource
  local resource_old_uid
  local resource_uid
  local resource_version
  local resource_kind
  local resource_name
  local resource_namespace
  local item
  if [[ "${resource}" == "" ]]; then
    return
  fi
  echo "${resource}" | jq -r '"Imported \(.kind) \(.metadata.name)"'

  if [[ "${other_resource}" == "" ]]; then
    return
  fi

  for resource_uid in $(echo "${resource}" | jq '.metadata.uid'); do
    item="$(echo "${resource}" | jq "select( .metadata.uid == ${resource_uid} )")"
    resource_version="$(echo "${item}" | jq '.apiVersion')"
    resource_kind="$(echo "${item}" | jq '.kind')"
    resource_name="$(echo "${item}" | jq '.metadata.name')"
    resource_namespace="$(echo "${item}" | jq '.metadata.namespace')"

    if [[ "${resource_namespace}" == "null" ]]; then
      apply_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion == ${resource_version} and .metadata.ownerReferences[0].kind == ${resource_kind} and .metadata.ownerReferences[0].name == ${resource_name} )")"
    else
      apply_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion == ${resource_version} and .metadata.ownerReferences[0].kind == ${resource_kind} and .metadata.ownerReferences[0].name == ${resource_name} and .metadata.namespace == ${resource_namespace} )")"
    fi

    if [[ "${apply_resource}" == "" ]]; then
      continue
    fi

    resource_old_uid="$(echo "${apply_resource}" | jq '.metadata.ownerReferences[0].uid' | head -n 1)"
    new_resource="$(echo "${apply_resource//${resource_old_uid}/${resource_uid}}" | kubectl --kubeconfig="${kubeconfig}" apply --force -o json -f -)"
    if [[ "$(echo "${new_resource}" | jq -r '.kind')" == "List" ]]; then
      new_resource="$(echo "${new_resource}" | jq '.items | .[]')"
    fi
    if [[ "${new_resource}" == "" ]]; then
      continue
    fi

    other_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion != ${resource_version} or .metadata.ownerReferences[0].kind != ${resource_kind} or .metadata.ownerReferences[0].name != ${resource_name} or .metadata.namespace != ${resource_namespace} )")"

    mock_resource "${kubeconfig}" "${new_resource}" "${other_resource}"
  done
}

# import to cluster from given resource
function mock_cluster() {
  local kubeconfig="${1}"
  local resources="${2}"
  local new_resource
  local other_resource
  local apply_resource

  resources="$(echo "${resources}" | jq 'select( .kind != "Namespace" or ( .metadata.name != "kube-public" and .metadata.name != "kube-node-lease" and .metadata.name != "kube-system" and .metadata.name != "default" ) )')"
  apply_resource="$(echo "${resources}" | jq 'select( .metadata.ownerReferences == null )')"
  new_resource="$(echo "${apply_resource}" | kubectl --kubeconfig="${kubeconfig}" apply --force -o json -f -)"
  if [[ "$(echo "${new_resource}" | jq -r '.kind')" == "List" ]]; then
    new_resource="$(echo "${new_resource}" | jq '.items | .[]')"
  fi
  other_resource="$(echo "${resources}" | jq 'select( .metadata.ownerReferences != null )')"
  mock_resource "${kubeconfig}" "${new_resource}" "${other_resource}"
}

# create a cluster
function create_cluster() {
  local name="${1}"
  local port="${2}"
  if [[ "${port}" == "random" || "${port}" == "" ]]; then
    port="$(unusad_port)"
  fi
  local full_name="${PROJECT_NAME}-${name}"
  local tmpdir="${TMPDIR}/clusters/${name}"
  local pkidir="${tmpdir}/pki"
  local scheme="http"
  local incluster_port="8080"
  local admin_crt=""
  local admin_key=""
  local ca_crt=""

  mkdir -p "${tmpdir}"

  if is_true "${SECURE_PORT}"; then
    # generate pki
    mkdir -p "${pkidir}"
    gen_cert "${full_name}" "${pkidir}"
    admin_crt="${pkidir}/admin.crt"
    admin_key="${pkidir}/admin.key"
    ca_crt="${pkidir}/ca.crt"
    scheme="https"
    incluster_port="6443"
  fi

  # Create in-cluster kubeconfig
  build_kubeconfig "${scheme}://${full_name}-kube-apiserver:${incluster_port}" "${admin_crt}" "${admin_key}" "${ca_crt}" >"${tmpdir}/kubeconfig"

  # Create local kubeconfig
  build_kubeconfig "${scheme}://127.0.0.1:${port}" "${admin_crt}" "${admin_key}" "${ca_crt}" >"${tmpdir}/kubeconfig.yaml"

  # Create cluster compose
  build_compose "${full_name}" "${port}" "${tmpdir}/kubeconfig" "${admin_crt}" "${admin_key}" "${ca_crt}" >"${tmpdir}/docker-compose.yaml"

  # Start cluster with compose
  "${RUNTIME}" compose -p "${full_name}" -f "${tmpdir}/docker-compose.yaml" up -d

  if command_exist kubectl; then
    # Set up default kubeconfig
    set_default_kubeconfig "${full_name}" "${port}" "${admin_crt}" "${admin_key}" "${ca_crt}" >/dev/null 2>&1

    # Wait for apiserver to be ready
    echo "kubectl --context=${full_name} get node"
    for _ in $(seq 1 10); do
      kubectl --context="${full_name}" get node >/dev/null 2>&1 && break
      sleep 1
    done
    kubectl --context="${full_name}" get node
  fi

  if [[ "${MOCK_CONTENT}" != "" ]]; then
    # Stop kube-controller-manager and import mock data
    echo "Importing mock data"
    "${RUNTIME}" stop "${full_name}-kube-controller" >/dev/null 2>&1
    mock_cluster "${tmpdir}/kubeconfig.yaml" "${MOCK_CONTENT}"
    "${RUNTIME}" start "${full_name}-kube-controller" >/dev/null 2>&1
  fi

  echo "Created cluster ${full_name}."
}

# delete a cluster
function delete_cluster() {
  local name="${1}"
  local tmpdir="${TMPDIR}/clusters/${name}"
  local full_name="${PROJECT_NAME}-${name}"

  if command_exist kubectl; then
    unset_default_kubeconfig "${full_name}" >/dev/null 2>&1
  fi

  if [[ -f "${tmpdir}/docker-compose.yaml" ]]; then
    "${RUNTIME}" compose -p "${full_name}" -f "${tmpdir}/docker-compose.yaml" kill
    "${RUNTIME}" compose -p "${full_name}" -f "${tmpdir}/docker-compose.yaml" down
  else
    "${RUNTIME}" compose -p "${full_name}" down
  fi

  rm -rf "${tmpdir}"
  echo "Deleted cluster ${full_name}."
}

# list all clusters
function list_cluster() {
  for file in "${TMPDIR}"/clusters/*/kubeconfig.yaml; do
    echo "${file}" | grep -o -e "/\([^/]\+\)/kubeconfig\.yaml$" | sed "s|/kubeconfig.yaml$||" | sed "s|^/||"
  done
}

# list all images used by fake cluster
function images() {
  echo "${IMAGE_ETCD}"
  echo "${IMAGE_KUBE_APISERVER}"
  echo "${IMAGE_KUBE_CONTROLLER_MANAGER}"
  echo "${IMAGE_KUBE_SCHEDULER}"
  echo "${IMAGE_FAKE_KUBELET}"
}

# usage info
function usage() {
  init_global_flags
  echo "Usage $0"
  echo "Commands:"
  echo "  create    Creates one fake cluster"
  echo "  delete    Deletes one fake cluster"
  echo "  list      List all fake cluster"
  echo "  images    List all images used by fake cluster"
  echo "Flags:"
  echo "  -h, --help                                 show this help"
  echo "  -n, --name string                          cluster name (default: 'default')"
  echo "  -p, --port uint16|random                   port of the apiserver of the cluster (default: 'random')"
  echo "  -r, --generate-replicas uint32             number of replicas of the generate node (default: '${GENERATE_REPLICAS}')"
  echo "  --generate-node-name string                generate node name prefix (default: '${GENERATE_NODE_NAME}')"
  echo "  --mock string                              mock specifies the cluster from file (default: '${MOCK_FILENAME}')"
  echo "  --node-name strings                        extra node name (default: '${NODE_NAME}')"
  echo "  --runtime string                           runtime to use (default: '${RUNTIME}')"
  echo "  --secure-port boolean                      use secure port (default: '${SECURE_PORT}')"
  echo "  --fake-version string                      version of the fake image (default: '${FAKE_VERSION}')"
  echo "  --kube-version string                      version of the kubernetes image (default: '${KUBE_VERSION}')"
  echo "  --etcd-version string                      version of the etcd image (default: '${ETCD_VERSION}')"
  echo "  --kube-image-prefix string                 prefix of the kubernetes image (default: '${KUBE_IMAGE_PREFIX}')"
  echo "  --fake-image-prefix string                 prefix of the fake image (default: '${FAKE_IMAGE_PREFIX}')"
  echo "  --image-etcd string                        etcd image (default: '${IMAGE_ETCD}')"
  echo "  --image-kube-apiserver string              kube-apiserver image (default: '${IMAGE_KUBE_APISERVER}')"
  echo "  --image-kube-controller-manager string     kube-controller-manager image (default: '${IMAGE_KUBE_CONTROLLER_MANAGER}')"
  echo "  --image-kube-scheduler string              kube-scheduler image (default: '${IMAGE_KUBE_SCHEDULER}')"
  echo "  --image-fake-kubelet string                fake-kubelet image (default: '${IMAGE_FAKE_KUBELET}')"
}

function main() {
  if [[ "$#" -eq 0 ]]; then
    usage
    return 1
  fi

  local port=""
  local name="default"
  local args=()

  while [[ "$#" -gt 0 ]]; do
    key="${1}"
    case ${key} in
    -p | -p=* | --port | --port=*)
      [[ "${key#*=}" != "${key}" ]] && port="${key#*=}" || { port="${2}" && shift; }
      ;;
    -n | -n=* | --name | --name=*)
      [[ "${key#*=}" != "${key}" ]] && name="${key#*=}" || { name="${2}" && shift; }
      ;;
    -r | -r=* | --replicas | --replicas=* | --generate-replicas | --generate-replicas=*)
      [[ "${key#*=}" != "${key}" ]] && GENERATE_REPLICAS="${key#*=}" || { GENERATE_REPLICAS="${2}" && shift; }
      ;;
    --generate-node-name | --generate-node-name=*)
      [[ "${key#*=}" != "${key}" ]] && GENERATE_NODE_NAME="${key#*=}" || { GENERATE_NODE_NAME="${2}" && shift; }
      ;;
    --mock | --mock=*)
      [[ "${key#*=}" != "${key}" ]] && MOCK_FILENAME="${key#*=}" || { MOCK_FILENAME="${2}" && shift; }
      ;;
    --node-name | --node-name=*)
      [[ "${key#*=}" != "${key}" ]] && NODE_NAME="${key#*=}" || { NODE_NAME="${2}" && shift; }
      ;;
    --runtime | --runtime=*)
      [[ "${key#*=}" != "${key}" ]] && RUNTIME="${key#*=}" || { RUNTIME="${2}" && shift; }
      ;;
    --secure-port | --secure-port=*)
      [[ "${key#*=}" != "${key}" ]] && SECURE_PORT="${key#*=}" || { SECURE_PORT="${2}" && shift; }
      ;;
    --fake-version | --fake-version=*)
      [[ "${key#*=}" != "${key}" ]] && FAKE_VERSION="${key#*=}" || { FAKE_VERSION="${2}" && shift; }
      ;;
    --kube-version | --kube-version=*)
      [[ "${key#*=}" != "${key}" ]] && KUBE_VERSION="${key#*=}" || { KUBE_VERSION="${2}" && shift; }
      ;;
    --etcd-version | --etcd-version=*)
      [[ "${key#*=}" != "${key}" ]] && ETCD_VERSION="${key#*=}" || { ETCD_VERSION="${2}" && shift; }
      ;;
    --kube-image-prefix | --kube-image-prefix=*)
      [[ "${key#*=}" != "${key}" ]] && KUBE_IMAGE_PREFIX="${key#*=}" || { KUBE_IMAGE_PREFIX="${2}" && shift; }
      ;;
    --fake-image-prefix | --fake-image-prefix=*)
      [[ "${key#*=}" != "${key}" ]] && FAKE_IMAGE_PREFIX="${key#*=}" || { FAKE_IMAGE_PREFIX="${2}" && shift; }
      ;;
    --image-etcd | --image-etcd=*)
      [[ "${key#*=}" != "${key}" ]] && IMAGE_ETCD="${key#*=}" || { IMAGE_ETCD="${2}" && shift; }
      ;;
    --image-kube-apiserver | --image-kube-apiserver=*)
      [[ "${key#*=}" != "${key}" ]] && IMAGE_KUBE_APISERVER="${key#*=}" || { IMAGE_KUBE_APISERVER="${2}" && shift; }
      ;;
    --image-kube-controller-manager | --image-kube-controller-manager=*)
      [[ "${key#*=}" != "${key}" ]] && IMAGE_KUBE_CONTROLLER_MANAGER="${key#*=}" || { IMAGE_KUBE_CONTROLLER_MANAGER="${2}" && shift; }
      ;;
    --image-kube-scheduler | --image-kube-scheduler=*)
      [[ "${key#*=}" != "${key}" ]] && IMAGE_KUBE_SCHEDULER="${key#*=}" || { IMAGE_KUBE_SCHEDULER="${2}" && shift; }
      ;;
    --image-fake-kubelet | --image-fake-kubelet=*)
      [[ "${key#*=}" != "${key}" ]] && IMAGE_FAKE_KUBELET="${key#*=}" || { IMAGE_FAKE_KUBELET="${2}" && shift; }
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
    create_cluster "${name}" "${port}"
    ;;
  "delete")
    delete_cluster "${name}"
    ;;
  "list")
    list_cluster
    ;;
  "images")
    images
    ;;
  *)
    usage
    return 1
    ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "${@}"
fi
