global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s
alerting:
  alertmanagers:
  - follow_redirects: true
    enable_http2: true
    scheme: http
    timeout: 10s
    api_version: v2
    static_configs:
    - targets: []
scrape_configs:
- job_name: "prometheus"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .PrometheusPort }}
- job_name: "etcd"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .EtcdPort }}
- job_name: "fake-kubelet"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .FakeKubeletPort }}

${{ if .SecretPort }}
- job_name: "kube-apiserver"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "${{ .AdminCrtPath }}"
    key_file: "${{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - localhost:${{ .KubeApiserverPort }}
- job_name: "kube-controller-manager"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "${{ .AdminCrtPath }}"
    key_file: "${{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - localhost:${{ .KubeControllerManagerPort }}
- job_name: "kube-scheduler"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "${{ .AdminCrtPath }}"
    key_file: "${{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - localhost:${{ .KubeSchedulerPort }}
${{ else }}
- job_name: "kube-apiserver"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .KubeApiserverPort }}
- job_name: "kube-controller-manager"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .KubeControllerManagerPort }}
- job_name: "kube-scheduler"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:${{ .KubeSchedulerPort }}
${{ end }}