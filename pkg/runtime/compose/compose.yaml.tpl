version: "3.1"
services:

${{ if .PrometheusPath }}
  # Prometheus
  prometheus:
    container_name: "${{ .ProjectName }}-prometheus"
    image: ${{ .PrometheusImage }}
    restart: unless-stopped
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
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro
${{ end }}

  # Etcd
  etcd:
    container_name: "${{ .ProjectName }}-etcd"
    image: ${{ .EtcdImage }}
    restart: unless-stopped
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
${{ if .EtcdDataPath }}
    volumes:
      - ${{ .EtcdDataPath }}:${{ .InClusterEtcdDataPath }}:rw
${{ end }}

  # Kube-apiserver
  kube_apiserver:
    container_name: "${{ .ProjectName }}-kube-apiserver"
    image: ${{ .KubeApiserverImage }}
    restart: unless-stopped
    links:
      - etcd
    ports:
      - ${{ .ApiserverPort }}:6443
    command:
      - kube-apiserver
      - --admission-control
      - ""
      - --etcd-servers
      - http://${{ .ProjectName }}-etcd:2379
      - --etcd-prefix
      - /prefix/registry
      - --default-watch-cache-size
      - "10000"
      - --allow-privileged
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
    volumes:
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro


  # Kube-controller-manager
  kube_controller_manager:
    container_name: "${{ .ProjectName }}-kube-controller-manager"
    image: ${{ .KubeControllerManagerImage }}
    restart: unless-stopped
    links:
      - kube_apiserver
    command:
      - kube-controller-manager
      - --kubeconfig
      - ${{ .InClusterKubeconfigPath }}

${{ if .PrometheusPath }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10257"
      - --authorization-always-allow-paths
      - /healthz,/metrics
${{ end }}
    volumes:
      - ${{ .KubeconfigPath }}:${{ .InClusterKubeconfigPath }}:ro
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro

  # Kube-scheduler
  kube_scheduler:
    container_name: "${{ .ProjectName }}-kube-scheduler"
    image: ${{ .KubeSchedulerImage }}
    restart: unless-stopped
    links:
      - kube_apiserver
    command:
      - kube-scheduler
      - --kubeconfig
      - ${{ .InClusterKubeconfigPath }}

${{ if .PrometheusPath }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10259"
      - --authorization-always-allow-paths
      - /healthz,/metrics
${{ end }}
    volumes:
      - ${{ .KubeconfigPath }}:${{ .InClusterKubeconfigPath }}:ro
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro

  # Fake-kubelet
  fake_kubelet:
    container_name: "${{ .ProjectName }}-fake-kubelet"
    image: ${{ .FakeKubeletImage }}
    restart: unless-stopped
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
      - ${{ .AdminKeyPath }}:${{ .InClusterAdminKeyPath }}:ro
      - ${{ .AdminCertPath }}:${{ .InClusterAdminCertPath }}:ro
      - ${{ .CACertPath }}:${{ .InClusterCACertPath }}:ro

# Network
networks:
  default:
    name: ${{ .ProjectName }}
