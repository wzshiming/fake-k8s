---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fake-kubelet
  namespace: kube-system
  labels:
    app: fake-kubelet
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: fake-kubelet
  labels:
    app: fake-kubelet
rules:
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - watch
      - list
      - create
      - get
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - update
      - patch
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - watch
      - list
      - delete
      - update
      - patch
  - apiGroups:
      - ""
    resources:
      - pods/status
    verbs:
      - update
      - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: fake-kubelet
  labels:
    app: fake-kubelet
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: fake-kubelet
subjects:
  - kind: ServiceAccount
    name: fake-kubelet
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: fake-kubelet
  namespace: kube-system
  labels:
    app: fake-kubelet
data:
  pod_status_template: |-
    {{ $startTime := .metadata.creationTimestamp }}
    conditions:
    - lastTransitionTime: {{ $startTime }}
      status: "True"
      type: Initialized
    - lastTransitionTime: {{ $startTime }}
      status: "True"
      type: Ready
    - lastTransitionTime: {{ $startTime }}
      status: "True"
      type: ContainersReady
    - lastTransitionTime: {{ $startTime }}
      status: "True"
      type: PodScheduled
    {{ range .spec.readinessGates }}
    - lastTransitionTime: {{ $startTime }}
      status: "True"
      type: {{ .conditionType }}
    {{ end }}
    containerStatuses:
    {{ range .spec.containers }}
    - image: {{ .image }}
      name: {{ .name }}
      ready: true
      restartCount: 0
      state:
        running:
          startedAt: {{ $startTime }}
    {{ end }}
    initContainerStatuses:
    {{ range .spec.initContainers }}
    - image: {{ .image }}
      name: {{ .name }}
      ready: true
      restartCount: 0
      state:
        terminated:
          exitCode: 0
          finishedAt: {{ $startTime }}
          reason: Completed
          startedAt: {{ $startTime }}
    {{ end }}
    {{ with .status }}
    hostIP: {{ with .hostIP }} {{ . }} {{ else }} {{ NodeIP }} {{ end }}
    podIP: {{ with .podIP }} {{ . }} {{ else }} {{ PodIP }} {{ end }}
    {{ end }}
    phase: Running
    startTime: {{ $startTime }}
  node_template: |-
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
  node_heartbeat_template: |-
    conditions:
    - lastHeartbeatTime: {{ Now }}
      lastTransitionTime: {{ StartTime }}
      message: kubelet is posting ready status
      reason: KubeletReady
      status: "True"
      type: Ready
    - lastHeartbeatTime: {{ Now }}
      lastTransitionTime: {{ StartTime }}
      message: kubelet has sufficient disk space available
      reason: KubeletHasSufficientDisk
      status: "False"
      type: OutOfDisk
    - lastHeartbeatTime: {{ Now }}
      lastTransitionTime: {{ StartTime }}
      message: kubelet has sufficient memory available
      reason: KubeletHasSufficientMemory
      status: "False"
      type: MemoryPressure
    - lastHeartbeatTime: {{ Now }}
      lastTransitionTime: {{ StartTime }}
      message: kubelet has no disk pressure
      reason: KubeletHasNoDiskPressure
      status: "False"
      type: DiskPressure
    - lastHeartbeatTime: {{ Now }}
      lastTransitionTime: {{ StartTime }}
      message: RouteController created a route
      reason: RouteCreated
      status: "False"
      type: NetworkUnavailable
  node_initialization_template: |-
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
---
apiVersion: v1
kind: Pod
metadata:
  name: fake-kubelet
  namespace: kube-system
spec:
  containers:
    - name: fake-kubelet
      image: ${{ .FakeKubeletImage }}
      imagePullPolicy: IfNotPresent
      env:
        - name: NODE_NAME
          value: "${{ .NodeName }}" # This is to specify a single Node, use GENERATE_NODE_NAME and GENERATE_REPLICAS to generate multiple nodes
        - name: GENERATE_NODE_NAME
          value: "${{ .GenerateNodeName }}"
        - name: GENERATE_REPLICAS
          value: "${{ .GenerateReplicas }}"
        - name: TAKE_OVER_LABELS_SELECTOR
          value: type=fake-kubelet
        - name: TAKE_OVER_ALL
          value: "false"
        - name: CIDR
          value: 10.0.0.1/24
        - name: HEALTH_ADDRESS # deprecated: use SERVER_ADDRESS instead
          value: :8080
        - name: SERVER_ADDRESS
          value: :8080
        - name: NODE_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: POD_STATUS_TEMPLATE
          valueFrom:
            configMapKeyRef:
              name: fake-kubelet
              key: pod_status_template
        - name: NODE_TEMPLATE
          valueFrom:
            configMapKeyRef:
              name: fake-kubelet
              key: node_template
        - name: NODE_HEARTBEAT_TEMPLATE
          valueFrom:
            configMapKeyRef:
              name: fake-kubelet
              key: node_heartbeat_template
        - name: NODE_INITIALIZATION_TEMPLATE
          valueFrom:
            configMapKeyRef:
              name: fake-kubelet
              key: node_initialization_template
      livenessProbe:
        httpGet:
          path: /health
          port: 8080
          scheme: HTTP
        initialDelaySeconds: 2
        timeoutSeconds: 2
        periodSeconds: 10
        failureThreshold: 3
      readinessProbe:
        httpGet:
          path: /health
          port: 8080
          scheme: HTTP
        initialDelaySeconds: 2
        timeoutSeconds: 2
        periodSeconds: 10
        failureThreshold: 3
      resources:
        requests:
          cpu: 500m
          memory: 100Mi
  serviceAccount: fake-kubelet
  serviceAccountName: fake-kubelet
  restartPolicy: Always
  hostNetwork: true
  nodeName: ${{ .Name }}-control-plane
---