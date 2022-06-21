version: "3.1"
services:

  # Etcd
  etcd:
    container_name: "${{ .ProjectName }}-etcd"
    image: ${{ .EtcdImage }}
    restart: always
    command:
      - etcd
      - --data-dir
      - ${{ .InClusterEtcdDataPath }}
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
      - --auto-compaction-retention
      - "1"
      - --quota-backend-bytes
      - "8589934592"

  # Kube-apiserver
  kube_apiserver:
    container_name: "${{ .ProjectName }}-kube-apiserver"
    image: ${{ .KubeApiserverImage }}
    restart: always
    links:
      - etcd
    ports:
${{ if .SecretPort }}
      - ${{ .ApiserverPort }}:6443
${{ else }}
      - ${{ .ApiserverPort }}:8080
${{ end }}
    command:
      - kube-apiserver
      - --admission-control
      - ""
      - --etcd-servers
      - http://${{ .ProjectName }}-etcd:2379
      - --etcd-prefix
      - /prefix/registry
      - --allow-privileged
${{ if .RuntimeConfig }}
      - --runtime-config
      - ${{ .RuntimeConfig }}
${{ end }}
${{ if .FeatureGates }}
      - --feature-gates
      - ${{ .FeatureGates }}
${{ end }}
${{ if .SecretPort }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "6443"
      - --tls-cert-file
      - ${{ .InClusterAdminCertPath }}
      - --tls-private-key-file
      - ${{ .InClusterAdminKeyPath }}
      - --client-ca-file
      - ${{ .InClusterCACertPath }}
      - --service-account-key-file
      - ${{ .InClusterAdminKeyPath }}
      - --service-account-signing-key-file
      - ${{ .InClusterAdminKeyPath }}
      - --service-account-issuer
      - https://kubernetes.default.svc.cluster.local
${{ else }}
      - --insecure-bind-address
      - 0.0.0.0
      - --insecure-port
      - "8080"
${{ end }}

${{ if .SecretPort }}
    volumes:
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}


  # Kube-controller-manager
  kube_controller_manager:
    container_name: "${{ .ProjectName }}-kube-controller-manager"
    image: ${{ .KubeControllerManagerImage }}
    restart: always
    links:
      - kube_apiserver
    command:
      - kube-controller-manager
      - --kubeconfig
      - ${{ .InClusterKubeconfigPath }}
${{ if .FeatureGates }}
      - --feature-gates
      - ${{ .FeatureGates }}
${{ end }}
${{ if .PrometheusPath }}
${{ if .SecretPort }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10257"
      - --authorization-always-allow-paths
      - /healthz,/metrics
${{ else }}
      - --address
      - 0.0.0.0
      - --port
      - "10252"
${{ end }}
${{ end }}
    volumes:
      - ${{ .KubeconfigPath }}:${{ .InClusterKubeconfigPath }}:ro
${{ if .SecretPort }}
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}

  # Kube-scheduler
  kube_scheduler:
    container_name: "${{ .ProjectName }}-kube-scheduler"
    image: ${{ .KubeSchedulerImage }}
    restart: always
    links:
      - kube_apiserver
    command:
      - kube-scheduler
      - --kubeconfig
      - ${{ .InClusterKubeconfigPath }}
${{ if .FeatureGates }}
      - --feature-gates
      - ${{ .FeatureGates }}
${{ end }}
${{ if .PrometheusPath }}
${{ if .SecretPort }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10259"
      - --authorization-always-allow-paths
      - /healthz,/metrics
${{ else }}
      - --address
      - 0.0.0.0
      - --port
      - "10251"
${{ end }}
${{ end }}
    volumes:
      - ${{ .KubeconfigPath }}:${{ .InClusterKubeconfigPath }}:ro
${{ if .SecretPort }}
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}

  # Fake-kubelet
  fake_kubelet:
    container_name: "${{ .ProjectName }}-fake-kubelet"
    image: ${{ .FakeKubeletImage }}
    restart: always
    command:
      - --kubeconfig
      - ${{ .InClusterKubeconfigPath }}
    links:
      - kube_apiserver
    environment:
      TAKE_OVER_ALL: "true"
      NODE_NAME: "${{ .NodeName }}"
      GENERATE_NODE_NAME: "${{ .GenerateNodeName }}"
      GENERATE_REPLICAS: "${{ .GenerateReplicas }}"
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
${{ if .PrometheusPath }}
      SERVER_ADDRESS: :8080
${{ end }}
    volumes:
      - ${{ .KubeconfigPath }}:${{ .InClusterKubeconfigPath }}:ro
${{ if .SecretPort }}
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}

${{ if .PrometheusPath }}
  # Prometheus
  prometheus:
    container_name: "${{ .ProjectName }}-prometheus"
    image: ${{ .PrometheusImage }}
    restart: always
    links:
      - kube_controller_manager
      - kube_scheduler
      - kube_apiserver
      - etcd
      - fake_kubelet
    command:
      - --config.file
      - ${{ .InClusterPrometheusPath }}
    ports:
      - ${{ .PrometheusPort }}:9090
    volumes:
      - ${{ .PrometheusPath }}:${{ .InClusterPrometheusPath }}:ro
${{ if .SecretPort }}
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}
${{ end }}

# Network
networks:
  default:
    name: ${{ .ProjectName }}
