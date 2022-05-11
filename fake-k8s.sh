#!/usr/bin/env bash

# initialize the global variables
function init_global_flags() {
  PROJECT_NAME="fake-k8s"
  TMPDIR="${TMPDIR:-/tmp}/${PROJECT_NAME}"
  TMPDIR="${TMPDIR//\/\//\/}"

  RUNTIME="${RUNTIME:-$(detection_runtime)}"

  MOCK="${MOCK:-}"
  if [[ "${MOCK}" != "" ]]; then
    GENERATE_REPLICAS="${GENERATE_REPLICAS:-0}"
  else
    GENERATE_REPLICAS="${GENERATE_REPLICAS:-5}"
  fi
  GENERATE_NODE_NAME="${GENERATE_NODE_NAME:-fake-}"
  NODE_NAME="${NODE_NAME:-}"

  FAKE_VERSION="${FAKE_VERSION:-v0.6.0}"
  KUBE_VERSION="${KUBE_VERSION:-v1.19.16}"
  ETCD_VERSION="${ETCD_VERSION:-$(get_etcd_version "${KUBE_VERSION}")}"

  # kubernetes v1.20 secure port must be enabled
  if [[ "$(get_release_version "${KUBE_VERSION}")" -ge "20" ]]; then
    SECURE_PORT="${SECURE_PORT:-true}"
  else
    SECURE_PORT="${SECURE_PORT:-false}"
  fi

  QUIET_PULL="${QUIET_PULL:-false}"

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
  local i
  for ((i = 0; i < 10; i++)); do
    candidate="$((low_bound + (RANDOM % range)))"
    # just works on bash
    if ! (echo "" >/dev/tcp/127.0.0.1/${candidate}) >/dev/null 2>&1; then
      echo ${candidate}
      return 0
    fi
  done

  for ((i = 0; i < "${range}"; i++)); do
    candidate="$((low_bound + i))"
    # just works on bash
    if ! (echo "" >/dev/tcp/127.0.0.1/${candidate}) >/dev/null 2>&1; then
      echo ${candidate}
      return 0
    fi
  done

  # fallback to low_bound - 1 port if no unusad port is available
  echo "$((low_bound - 1))"
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
      insecure-skip-tls-verify: true
      # certificate-authority-data: $(base64 <"${ca_crt_path}" | tr -d '\n')
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
      client-certificate-data: $(base64 <"${admin_crt_path}" | tr -d '\n')
      client-key-data: $(base64 <"${admin_key_path}" | tr -d '\n')
EOF
  fi
}

# build the docker-compose file with the given context
function build_compose() {
  local name="${1}"
  local port="${2}"
  local kubeconfig_path="${3}"
  local etcd_data="${4}"
  local admin_crt_path="${5}"
  local admin_key_path="${6}"
  local ca_crt_path="${7}"

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
    volumes:
      - ${etcd_data}:/etcd-data:rw
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
      - /etc/kubernetes/pki/admin.crt
      - --tls-private-key-file
      - /etc/kubernetes/pki/admin.key
      - --client-ca-file
      - /etc/kubernetes/pki/ca.crt
      - --service-account-key-file
      - /etc/kubernetes/pki/admin.key
      - --service-account-signing-key-file
      - /etc/kubernetes/pki/admin.key
      - --service-account-issuer
      - https://kubernetes.default.svc.cluster.local
    configs:
      - source: admin-crt
        target: /etc/kubernetes/pki/admin.crt
      - source: admin-key
        target: /etc/kubernetes/pki/admin.key
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
  kube_controller_manager:
    container_name: "${name}-kube-controller-manager"
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
      TAKE_OVER_ALL: "true"
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
        {{ with .status }}

        addresses:
        {{ with .addresses }}
        {{ YAML . 1 }}
        {{ else }}
        - address: {{ NodeIP }}
          type: InternalIP
        {{ end }}

        allocatable:
        {{ with .allocatable }}
        {{ YAML . 1 }}
        {{ else }}
          cpu: 1k
          memory: 1Ti
          pods: 1M
        {{ end }}

        capacity:
        {{ with .capacity }}
        {{ YAML . 1 }}
        {{ else }}
          cpu: 1k
          memory: 1Ti
          pods: 1M
        {{ end }}

        {{ with .nodeInfo }}
        nodeInfo:
          architecture: {{ with .architecture }} {{ . }} {{ else }} "amd64" {{ end }}
          bootID: {{ with .bootID }} {{ . }} {{ else }} "" {{ end }}
          containerRuntimeVersion: {{ with .containerRuntimeVersion }} {{ . }} {{ else }} "" {{ end }}
          kernelVersion: {{ with .kernelVersion }} {{ . }} {{ else }} "" {{ end }}
          kubeProxyVersion: {{ with .kubeProxyVersion }} {{ . }} {{ else }} "fake" {{ end }}
          kubeletVersion: {{ with .kubeletVersion }} {{ . }} {{ else }} "fake" {{ end }}
          machineID: {{ with .machineID }} {{ . }} {{ else }} "" {{ end }}
          operatingSystem: {{ with .operatingSystem }} {{ . }} {{ else }} "linux" {{ end }}
          osImage: {{ with .osImage }} {{ . }} {{ else }} "" {{ end }}
          systemUUID: {{ with .osImage }} {{ . }} {{ else }} "" {{ end }}
        {{ end }}

        phase: Running

        {{ end }}
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
  local dir="${1}"
  local admin_crt_path="${dir}/admin.crt"
  local admin_key_path="${dir}/admin.key"
  local admin_csr_path="${dir}/admin.csr"
  local ca_crt_path="${dir}/ca.crt"
  local ca_key_path="${dir}/ca.key"
  local openssl_conf_path="${dir}/openssl.cnf"

  # Generate the ca private key and certificate.
  if [[ ! -f "${ca_key_path}" ]]; then
    openssl genrsa -out "${ca_key_path}" 2048
  fi
  if [[ ! -f "${ca_crt_path}" ]]; then
    openssl req -sha256 -x509 -new -nodes -key "${ca_key_path}" -subj "/CN=fake-ca" -out "${ca_crt_path}" -days 36500
  fi

  # Generate the admin private key and certificate signing request and sign it with the ca.
  if [[ ! -f "${admin_key_path}" ]]; then
    openssl genrsa -out "${admin_key_path}" 2048
  fi
  if [[ ! -f "${admin_crt_path}" ]]; then
    if [[ ! -f "${openssl_conf_path}" ]]; then
      cat <<EOF >"${openssl_conf_path}"
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
IP.1 = 127.0.0.1
EOF
    fi
    if [[ ! -f "${admin_csr_path}" ]]; then
      openssl req -new -key "${admin_key_path}" -subj "/CN=fake-admin" -out "${admin_csr_path}" -config "${openssl_conf_path}"
    fi
    openssl x509 -sha256 -req -in "${admin_csr_path}" -CA "${ca_crt_path}" -CAkey "${ca_key_path}" -CAcreateserial -out "${admin_crt_path}" -days 36500 -extensions v3_req -extfile "${openssl_conf_path}"
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
    kubectl config set "clusters.${name}.insecure-skip-tls-verify" "true"
    # kubectl config set "clusters.${name}.certificate-authority-data" "$(base64 <"${ca_crt_path}" | tr -d '\n')"
    kubectl config set "users.${name}.client-certificate-data" "$(base64 <"${admin_crt_path}" | tr -d '\n')"
    kubectl config set "users.${name}.client-key-data" "$(base64 <"${admin_key_path}" | tr -d '\n')"
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

# output logs to stderr
function log() {
  echo "${*}" >&2
}

# output error logs to stderr
function error() {
  echo "Error: ${*}" >&2
}

# import to cluster from given resource and modify the resource uid
function mock_cluster() {
  local kubeconfig="${1}"
  local mock="${2}"
  local resources
  local line_resource
  local other_resource
  local apply_resource
  local next_resource
  local resource_old_uid
  local resource_uid
  local resource_version
  local resource_kind
  local resource_name
  local resource_namespace

  # take content from the file or stdin
  if [[ "${mock}" == "-" ]]; then
    resources="$(cat)"
  else
    resources="$(cat "${mock}")"
  fi
  if [[ "$(echo "${resources}" | jq -r '.kind')" == "List" ]]; then
    resources="$(echo "${resources}" | jq '.items | .[]')"
  fi

  resources="$(echo "${resources}" | jq 'select( .kind != "Namespace" or ( .metadata.name != "kube-public" and .metadata.name != "kube-node-lease" and .metadata.name != "kube-system" and .metadata.name != "default" ) )')"
  apply_resource="$(echo "${resources}" | jq 'select( .metadata.ownerReferences == null )')"
  other_resource="$(echo "${resources}" | jq 'select( .metadata.ownerReferences != null )')"

  log "Importing mock data"

  while [[ "${apply_resource}" != "" ]]; do
    line_resource="$(echo "${apply_resource}" | kubectl --kubeconfig="${kubeconfig}" apply --validate=false --force -f - -o go-template='{{ range .items }}{{ .metadata.uid }} {{ .apiVersion }} {{ .kind }} {{ .metadata.name }} {{ with .metadata.namespace }}{{ . }}{{ end }}{{"\n"}}{{- end }}')"
    apply_resource=""
    while read -r resource_uid resource_version resource_kind resource_name resource_namespace; do
      if [[ "${resource_uid}" == "" ]]; then
        continue
      fi
      log "Imported ${resource_kind} ${resource_name}"
      if [[ "${resource_namespace}" == "" ]]; then
        next_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion == \"${resource_version}\" and .metadata.ownerReferences[0].kind == \"${resource_kind}\" and .metadata.ownerReferences[0].name == \"${resource_name}\" )")"
      else
        next_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion == \"${resource_version}\" and .metadata.ownerReferences[0].kind == \"${resource_kind}\" and .metadata.ownerReferences[0].name == \"${resource_name}\" and .metadata.namespace == \"${resource_namespace}\" )")"
      fi

      if [[ "${next_resource}" == "" ]]; then
        continue
      fi

      if [[ "${resource_namespace}" == "" ]]; then
        other_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion != \"${resource_version}\" or .metadata.ownerReferences[0].kind != \"${resource_kind}\" or .metadata.ownerReferences[0].name != \"${resource_name}\" )")"
      else
        other_resource="$(echo "${other_resource}" | jq "select( .metadata.ownerReferences[0].apiVersion != \"${resource_version}\" or .metadata.ownerReferences[0].kind != \"${resource_kind}\" or .metadata.ownerReferences[0].name != \"${resource_name}\" or .metadata.namespace != \"${resource_namespace}\" )")"
      fi

      resource_old_uid="$(echo "${next_resource}" | jq -r '.metadata.ownerReferences[0].uid' | head -n 1)"
      apply_resource+="${next_resource//${resource_old_uid}/${resource_uid}}"

    done < <(echo "${line_resource}")
  done
}

# create a cluster
function create_cluster() {
  local name="${1}"
  local port="${2}"
  local full_name="${PROJECT_NAME}-${name}"
  local tmpdir="${TMPDIR}/clusters/${name}"
  local pkidir="${TMPDIR}/pki"
  local etcddir="${tmpdir}/etcd"
  local scheme="http"
  local incluster_port="8080"
  local admin_crt_path=""
  local admin_key_path=""
  local ca_crt_path=""
  local in_cluster_kubeconfig_path="${tmpdir}/kubeconfig"
  local local_kubeconfig_path="${tmpdir}/kubeconfig.yaml"
  local docker_compose_path="${tmpdir}/docker-compose.yaml"
  local up_args=()
  local i

  if [[ -f "${local_kubeconfig_path}" ]]; then
    log "kubectl --context=${full_name} get node"
    kubectl --context="${full_name}" get node >&2
    error "Cluster ${name} already exists"
    return 1
  fi

  # Checking dependent Installation
  if is_true "${SECURE_PORT}"; then
    if ! command_exist openssl; then
      error "OpenSSL needs to be installed with --secure-port=true"
      return 1
    fi
  fi
  if [[ "${MOCK}" != "" ]]; then
    if ! command_exist kubectl; then
      error "Kubectl needs to be installed with --mock"
      return 1
    fi
    if ! command_exist jq; then
      error "JQ needs to be installed with --mock"
      return 1
    fi
  fi

  if [[ "${port}" == "random" || "${port}" == "" ]]; then
    port="$(unusad_port)"
  fi

  mkdir -p "${tmpdir}" "${etcddir}"

  if is_true "${SECURE_PORT}"; then
    # generate pki
    mkdir -p "${pkidir}"
    gen_cert "${pkidir}"
    admin_crt_path="${pkidir}/admin.crt"
    admin_key_path="${pkidir}/admin.key"
    ca_crt_path="${pkidir}/ca.crt"
    scheme="https"
    incluster_port="6443"
  fi

  # Create in-cluster kubeconfig
  build_kubeconfig "${scheme}://${full_name}-kube-apiserver:${incluster_port}" "${admin_crt_path}" "${admin_key_path}" "${ca_crt_path}" >"${in_cluster_kubeconfig_path}"

  # Create local kubeconfig
  build_kubeconfig "${scheme}://127.0.0.1:${port}" "${admin_crt_path}" "${admin_key_path}" "${ca_crt_path}" >"${local_kubeconfig_path}"

  # Create cluster compose
  build_compose "${full_name}" "${port}" "${in_cluster_kubeconfig_path}" "${etcddir}" "${admin_crt_path}" "${admin_key_path}" "${ca_crt_path}" >"${docker_compose_path}"

  if is_true "${QUIET_PULL}"; then
    up_args+=("--quiet-pull")
  fi

  # Start cluster with compose
  "${RUNTIME}" compose -p "${full_name}" -f "${docker_compose_path}" up -d "${up_args[@]}"
  if [[ "${?}" != 0 ]]; then
    error "Failed create cluster ${name}"
    return 1
  fi

  if command_exist kubectl; then
    # Set up default kubeconfig
    set_default_kubeconfig "${full_name}" "${port}" "${admin_crt_path}" "${admin_key_path}" "${ca_crt_path}" >/dev/null 2>&1

    # Wait for apiserver to be ready
    log "kubectl --context=${full_name} get node"
    for ((i = 0; i < 10; i++)); do
      kubectl --context="${full_name}" get node >/dev/null 2>&1 && break
      sleep 1
    done
    kubectl --context="${full_name}" get node >&2
  fi

  if [[ "${MOCK}" != "" ]]; then
    # Stop kube-controller-manager and import mock data
    "${RUNTIME}" stop "${full_name}-kube-controller-manager" >/dev/null 2>&1
    mock_cluster "${local_kubeconfig_path}" "${MOCK}"

    # Recreate fake-kubelet
    build_compose "${full_name}" "${port}" "${in_cluster_kubeconfig_path}" "${etcddir}" "${admin_crt_path}" "${admin_key_path}" "${ca_crt_path}" >"${docker_compose_path}"

    # Start cluster with compose
    "${RUNTIME}" compose -p "${full_name}" -f "${docker_compose_path}" up -d "${up_args[@]}"

    log "kubectl --context=${full_name} get node"
    kubectl --context="${full_name}" get node >&2
  fi

  if ! command_exist kubectl; then
    log "kubeconfig is available at ${local_kubeconfig_path}"
  fi
  log "Created cluster ${name}"
}

# delete a cluster
function delete_cluster() {
  local name="${1}"
  local tmpdir="${TMPDIR}/clusters/${name}"
  local full_name="${PROJECT_NAME}-${name}"
  local docker_compose_path="${tmpdir}/docker-compose.yaml"

  if command_exist kubectl; then
    unset_default_kubeconfig "${full_name}" >/dev/null 2>&1
  fi

  if [[ -f "${docker_compose_path}" ]]; then
    "${RUNTIME}" compose -p "${full_name}" -f "${docker_compose_path}" kill
    "${RUNTIME}" compose -p "${full_name}" -f "${docker_compose_path}" down
  else
    "${RUNTIME}" compose -p "${full_name}" down
  fi

  rm -rf "${tmpdir}"
  log "Deleted cluster ${name}"
}

# list all clusters
function list_cluster() {
  for file in "${TMPDIR}"/clusters/*/kubeconfig.yaml; do
    if [[ -f "${file}" ]]; then
      echo "${file}" | grep -o -e "/\([^/]\+\)/kubeconfig\.yaml$" | sed "s|/kubeconfig.yaml$||" | sed "s|^/||"
    fi
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
  echo "  --mock string                              mock specifies the cluster from file (default: '${MOCK}')"
  echo "  --node-name strings                        extra node name (default: '${NODE_NAME}')"
  echo "  --runtime string                           runtime to use (default: '${RUNTIME}')"
  echo "  --secure-port                              use secure port"
  echo "  --quiet-pull                               pull without printing progress information"
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
      [[ "${key#*=}" != "${key}" ]] && MOCK="${key#*=}" || { MOCK="${2}" && shift; }
      ;;
    --node-name | --node-name=*)
      [[ "${key#*=}" != "${key}" ]] && NODE_NAME="${key#*=}" || { NODE_NAME="${2}" && shift; }
      ;;
    --runtime | --runtime=*)
      [[ "${key#*=}" != "${key}" ]] && RUNTIME="${key#*=}" || { RUNTIME="${2}" && shift; }
      ;;
    --secure-port | --secure-port=*)
      [[ "${key#*=}" != "${key}" ]] && SECURE_PORT="${key#*=}" || SECURE_PORT="true"
      ;;
    --quiet-pull | --quiet-pull=*)
      [[ "${key#*=}" != "${key}" ]] && QUIET_PULL="${key#*=}" || QUIET_PULL="true"
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
    error "Unknown command ${command}"
    return 1
    ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "${@}"
fi
