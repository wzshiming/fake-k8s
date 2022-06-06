package vars

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/utils"
)

var (
	// ProjectName is the name of the project.
	ProjectName = getEnv("PROJECT_NAME", "fake-k8s")

	// TempDir creates a temporary directory with the given prefix.
	TempDir = filepath.Join(os.TempDir(), ProjectName, "clusters")

	// Runtime is the runtime to use.
	Runtime = getEnv("RUNTIME", detectionRuntime())

	// Mock is the mock data to use.
	Mock = getEnv("MOCK", "")

	// GenerateReplicas is the number of replicas to generate.
	GenerateReplicas = getEnvInt("GENERATE_REPLICAS", func() int {
		if Mock == "" {
			return 5
		} else {
			return 0
		}
	}())

	// GenerateNodeName is the name of the node to generate.
	GenerateNodeName = getEnv("GENERATE_NODE_NAME", func() string {
		if Mock == "" {
			return "fake-"
		} else {
			return ""
		}
	}())

	// NodeName is the name of the node to use.
	NodeName = strings.Split(getEnv("NODE_NAME", ""), ",")

	// PrometheusPort is the port to expose Prometheus metrics.
	PrometheusPort = getEnvInt("PROMETHEUS_PORT", 0)

	// FakeVersion is the version of the fake to use.
	FakeVersion = getEnv("FAKE_VERSION", "v0.7.0")

	// KubeVersion is the version of Kubernetes to use.
	KubeVersion = getEnv("KUBE_VERSION", "v1.19.16")

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
	runtimes := []string{"docker", "nerdctl"}
	for _, r := range runtimes {
		if _, err := utils.LookPath(r); err == nil {
			return r
		}
	}
	return runtimes[0]
}
