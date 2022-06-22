package runtime

import (
	"context"
	"io"
	"time"

	"github.com/wzshiming/fake-k8s/pkg/utils"
)

type Config struct {
	Name    string
	Workdir string
	Runtime string

	PrometheusPort   uint32
	GenerateNodeName string
	GenerateReplicas uint32
	NodeName         string

	// For docker-compose
	EtcdImage                  string
	KubeApiserverImage         string
	KubeControllerManagerImage string
	KubeSchedulerImage         string
	FakeKubeletImage           string
	PrometheusImage            string

	// For kind
	KindNodeImage string

	// For binary
	KubeApiserverBinary         string
	KubeControllerManagerBinary string
	KubeSchedulerBinary         string
	FakeKubeletBinary           string
	EtcdBinaryTar               string
	PrometheusBinaryTar         string

	// Cache directory
	CacheDir string

	// For docker-compose and binary
	SecretPort bool

	// Pull image
	QuietPull bool

	// Feature gates of Kubernetes
	FeatureGates string

	// Runtime config of Kubernetes
	RuntimeConfig string
}

type Runtime interface {
	// Install the cluster
	Install(ctx context.Context, conf Config) error

	// Uninstall the cluster
	Uninstall(ctx context.Context) error

	// Ready check the cluster is ready
	Ready(ctx context.Context) (bool, error)

	// WaitReady wait the cluster is ready
	WaitReady(ctx context.Context, timeout time.Duration) error

	// Up start the cluster
	Up(ctx context.Context) error

	// Down stop the cluster
	Down(ctx context.Context) error

	// Start start a container
	Start(ctx context.Context, name string) error

	// Stop stop a container
	Stop(ctx context.Context, name string) error

	// Config return the cluster config
	Config() (*Config, error)

	// InHostKubeconfig return the kubeconfig in host
	InHostKubeconfig() (string, error)

	// Kubectl command
	Kubectl(ctx context.Context, stm utils.IOStreams, args ...string) error

	// KubectlInCluster command in cluster
	KubectlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error

	// Logs logs of a component
	Logs(ctx context.Context, name string, out io.Writer) error

	// LogsFollow follow logs of a component with follow
	LogsFollow(ctx context.Context, name string, out io.Writer) error
}
