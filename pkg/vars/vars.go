package vars

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/utils"
)

var (
	// ProjectName is the name of the project.
	ProjectName = "fake-k8s"

	// DefaultCluster the default cluster name
	DefaultCluster = "default"

	// TempDir creates a temporary directory with the given prefix.
	TempDir = utils.PathJoin(os.TempDir(), ProjectName, "clusters")

	// CacheDir creates a cache directory with the given prefix.
	CacheDir = utils.PathJoin(os.TempDir(), ProjectName, "cache")

	// Runtime is the runtime to use.
	Runtime = getEnv("RUNTIME", detectionRuntime())

	// GenerateReplicas is the number of replicas to generate.
	GenerateReplicas = getEnvInt("GENERATE_REPLICAS", 5)

	// GenerateNodeName is the name of the node to generate.
	GenerateNodeName = getEnv("GENERATE_NODE_NAME", "fake-")

	// NodeName is the name of the node to use.
	NodeName = strings.Split(getEnv("NODE_NAME", ""), ",")

	// PrometheusPort is the port to expose Prometheus metrics.
	PrometheusPort = getEnvInt("PROMETHEUS_PORT", 0)

	// FakeVersion is the version of the fake to use.
	FakeVersion = getEnv("FAKE_VERSION", "v0.7.3")

	// KubeVersion is the version of Kubernetes to use.
	KubeVersion = getEnv("KUBE_VERSION", "v1.24.1")

	// EtcdVersion is the version of etcd to use.
	EtcdVersion = getEnv("ETCD_VERSION", k8s.GetEtcdVersion(parseRelease(KubeVersion)))

	// PrometheusVersion is the version of Prometheus to use.
	PrometheusVersion = getEnv("PROMETHEUS_VERSION", "v2.35.0")

	// SecurePort is the Apiserver use TLS.
	SecurePort = getEnvBool("SECURE_PORT", parseRelease(KubeVersion) > 19)

	// QuietPull is the flag to quiet the pull.
	QuietPull = getEnvBool("QUIET_PULL", false)

	// KubeImagePrefix is the prefix of the kubernetes image.
	KubeImagePrefix = getEnv("KUBE_IMAGE_PREFIX", "k8s.gcr.io")

	// EtcdImagePrefix is the prefix of the etcd image.
	EtcdImagePrefix = getEnv("ETCD_IMAGE_PREFIX", KubeImagePrefix)

	// FakeImagePrefix is the prefix of the fake image.
	FakeImagePrefix = getEnv("FAKE_IMAGE_PREFIX", "ghcr.io/wzshiming/fake-kubelet")

	// PrometheusImagePrefix is the prefix of the Prometheus image.
	PrometheusImagePrefix = getEnv("PROMETHEUS_IMAGE_PREFIX", "docker.io/prom")

	// EtcdImage is the image of etcd.
	EtcdImage = getEnv("ETCD_IMAGE", joinImageURI(EtcdImagePrefix, "etcd", EtcdVersion))

	// KubeApiserverImage is the image of kube-apiserver.
	KubeApiserverImage = getEnv("KUBE_APISERVER_IMAGE", joinImageURI(KubeImagePrefix, "kube-apiserver", KubeVersion))

	// KubeControllerManagerImage is the image of kube-controller-manager.
	KubeControllerManagerImage = getEnv("KUBE_CONTROLLER_MANAGER_IMAGE", joinImageURI(KubeImagePrefix, "kube-controller-manager", KubeVersion))

	// KubeSchedulerImage is the image of kube-scheduler.
	KubeSchedulerImage = getEnv("KUBE_SCHEDULER_IMAGE", joinImageURI(KubeImagePrefix, "kube-scheduler", KubeVersion))

	// FakeKubeletImage is the image of fake-kubelet.
	FakeKubeletImage = getEnv("FAKE_KUBELET_IMAGE", joinImageURI(FakeImagePrefix, "fake-kubelet", FakeVersion))

	// PrometheusImage is the image of Prometheus.
	PrometheusImage = getEnv("PROMETHEUS_IMAGE", joinImageURI(PrometheusImagePrefix, "prometheus", PrometheusVersion))

	// KindNodeImagePrefix is the prefix of the kind node image.
	KindNodeImagePrefix = getEnv("KIND_NODE_IMAGE_PREFIX", "docker.io/kindest")

	// KindNodeImage is the image of kind node.
	KindNodeImage = getEnv("KIND_NODE_IMAGE", joinImageURI(KindNodeImagePrefix, "node", KubeVersion))

	// KubeBinaryPrefix is the prefix of the kubernetes binary.
	KubeBinaryPrefix = getEnv("KUBE_BINARY_PREFIX", "https://dl.k8s.io/release/"+KubeVersion+"/bin/"+runtime.GOOS+"/"+runtime.GOARCH)

	// KubeApiserverBinary is the binary of kube-apiserver.
	KubeApiserverBinary = getEnv("KUBE_APISERVER_BINARY", KubeBinaryPrefix+"/kube-apiserver"+BinSuffix)

	// KubeControllerManagerBinary is the binary of kube-controller-manager.
	KubeControllerManagerBinary = getEnv("KUBE_CONTROLLER_MANAGER_BINARY", KubeBinaryPrefix+"/kube-controller-manager"+BinSuffix)

	// KubeSchedulerBinary is the binary of kube-scheduler.
	KubeSchedulerBinary = getEnv("KUBE_SCHEDULER_BINARY", KubeBinaryPrefix+"/kube-scheduler"+BinSuffix)

	// MustKubectlBinary is the binary of kubectl.
	MustKubectlBinary = "https://dl.k8s.io/release/" + KubeVersion + "/bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/kubectl" + BinSuffix

	// KubectlBinary is the binary of kubectl.
	KubectlBinary = getEnv("KUBECTL_BINARY", KubeBinaryPrefix+"/kubectl"+BinSuffix)

	// EtcdBinaryPrefix is the prefix of the etcd binary.
	EtcdBinaryPrefix = getEnv("ETCD_BINARY_PREFIX", "https://github.com/etcd-io/etcd/releases/download")

	// EtcdBinaryTar is the binary of etcd.
	EtcdBinaryTar = getEnv("ETCD_BINARY_TAR", EtcdBinaryPrefix+"/v"+strings.TrimSuffix(EtcdVersion, "-0")+"/etcd-v"+strings.TrimSuffix(EtcdVersion, "-0")+"-"+runtime.GOOS+"-"+runtime.GOARCH+"."+func() string {
		if runtime.GOOS == "linux" {
			return "tar.gz"
		}
		return "zip"
	}())

	// FakeKubeletBinaryPrefix is the prefix of the fake kubelet binary.
	FakeKubeletBinaryPrefix = getEnv("FAKE_KUBELET_BINARY_PREFIX", "https://github.com/wzshiming/fake-kubelet/releases/download")

	// FakeKubeletBinary is the binary of fake-kubelet.
	FakeKubeletBinary = getEnv("FAKE_KUBELET_BINARY", FakeKubeletBinaryPrefix+"/"+FakeVersion+"/fake-kubelet_"+runtime.GOOS+"_"+runtime.GOARCH+BinSuffix)

	// PrometheusBinaryPrefix is the prefix of the Prometheus binary.
	PrometheusBinaryPrefix = getEnv("PROMETHEUS_BINARY_PREFIX", "https://github.com/prometheus/prometheus/releases/download")

	// PrometheusBinaryTar is the binary of Prometheus.
	PrometheusBinaryTar = getEnv("PROMETHEUS_BINARY_TAR", PrometheusBinaryPrefix+"/"+PrometheusVersion+"/prometheus-"+strings.TrimPrefix(PrometheusVersion, "v")+"."+runtime.GOOS+"-"+runtime.GOARCH+"."+func() string {
		if runtime.GOOS == "windows" {
			return "zip"
		}
		return "tar.gz"
	}())

	BinSuffix = func() string {
		if runtime.GOOS == "windows" {
			return ".exe"
		}
		return ""
	}()
)

// getEnv returns the value of the environment variable named by the key.
func getEnv(key, def string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return value
}

// getEnvInt returns the value of the environment variable named by the key.
func getEnvInt(key string, def int) int {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// getEnvBool returns the value of the environment variable named by the key.
func getEnvBool(key string, def bool) bool {
	v := getEnv(key, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// joinImageURI joins the image URI.
func joinImageURI(prefix, name, version string) string {
	return prefix + "/" + name + ":" + version
}

// parseRelease returns the release of the version.
func parseRelease(version string) int {
	release := strings.Split(version, ".")
	if len(release) < 2 {
		return 0
	}
	r, err := strconv.ParseInt(release[1], 10, 64)
	if err != nil {
		return 0
	}
	return int(r)
}

func detectionRuntime() string {
	if runtime.GOOS == "linux" {
		return "binary"
	}
	return "docker"
}
