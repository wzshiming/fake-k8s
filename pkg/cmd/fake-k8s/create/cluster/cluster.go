package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name                        string
	ApiserverPort               uint32
	PrometheusPort              uint32
	SecurePort                  bool
	QuietPull                   bool
	EtcdImage                   string
	KubeApiserverImage          string
	KubeControllerManagerImage  string
	KubeSchedulerImage          string
	FakeKubeletImage            string
	PrometheusImage             string
	KindNodeImage               string
	KubeApiserverBinary         string
	KubeControllerManagerBinary string
	KubeSchedulerBinary         string
	FakeKubeletBinary           string
	EtcdBinaryTar               string
	PrometheusBinaryTar         string
	GenerateReplicas            uint32
	GenerateNodeName            string
	NodeName                    []string
	Runtime                     string
	FeatureGates                string
	RuntimeConfig               string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.MaximumNArgs(1),
		Use:   "cluster [name]",
		Short: "Creates a fake Kubernetes cluster",
		Long:  "Creates a fake Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				flags.Name = vars.DefaultCluster
			} else {
				flags.Name = args[0]
			}
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().Uint32Var(&flags.ApiserverPort, "apiserver-port", uint32(vars.ApiserverPort), "port of the apiserver, default is random")
	cmd.Flags().Uint32Var(&flags.PrometheusPort, "prometheus-port", uint32(vars.PrometheusPort), `port to expose Prometheus metrics`)
	cmd.Flags().BoolVar(&flags.SecurePort, "secure-port", vars.SecurePort, `apiserver use TLS`)
	cmd.Flags().BoolVar(&flags.QuietPull, "quiet-pull", vars.QuietPull, `pull without printing progress information`)
	cmd.Flags().StringVar(&flags.EtcdImage, "etcd-image", vars.EtcdImage, `image of etcd, only for docker/nerdctl runtime
'${KUBE_IMAGE_PREFIX}/etcd:${ETCD_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverImage, "kube-apiserver-image", vars.KubeApiserverImage, `image of kube-apiserver, only for docker/nerdctl runtime
'${KUBE_IMAGE_PREFIX}/kube-apiserver:${KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerImage, "kube-controller-manager-image", vars.KubeControllerManagerImage, `image of kube-controller-manager, only for docker/nerdctl runtime
'${KUBE_IMAGE_PREFIX}/kube-controller-manager:${KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerImage, "kube-scheduler-image", vars.KubeSchedulerImage, `image of kube-scheduler, only for docker/nerdctl runtime
'${KUBE_IMAGE_PREFIX}/kube-scheduler:${KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.FakeKubeletImage, "fake-kubelet-image", vars.FakeKubeletImage, `image of fake-kubelet, only for docker/nerdctl/kind runtime
'${FAKE_IMAGE_PREFIX}/fake-kubelet:${FAKE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.PrometheusImage, "prometheus-image", vars.PrometheusImage, `image of Prometheus, only for docker/nerdctl/kind runtime
'${PROMETHEUS_IMAGE_PREFIX}/prometheus:${PROMETHEUS_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KindNodeImage, "kind-node-image", vars.KindNodeImage, `image of kind node, only for kind runtime
'${KIND_NODE_IMAGE_PREFIX}/node:${KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverBinary, "kube-apiserver-binary", vars.KubeApiserverBinary, `binary of kube-apiserver, only for binary runtime
'${KUBE_BINARY_PREFIX}/kube-apiserver'
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerBinary, "kube-controller-manager-binary", vars.KubeControllerManagerBinary, `binary of kube-controller-manager, only for binary runtime
'${KUBE_BINARY_PREFIX}/kube-controller-manager'
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerBinary, "kube-scheduler-binary", vars.KubeSchedulerBinary, `binary of kube-scheduler, only for binary runtime
'${KUBE_BINARY_PREFIX}/kube-scheduler'
`)
	cmd.Flags().StringVar(&flags.FakeKubeletBinary, "fake-kubelet-binary", vars.FakeKubeletBinary, `binary of fake-kubelet, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.EtcdBinaryTar, "etcd-binary-tar", vars.EtcdBinaryTar, `tar of etcd, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.PrometheusBinaryTar, "prometheus-binary-tar", vars.PrometheusBinaryTar, `tar of Prometheus, only for binary runtime
`)
	cmd.Flags().Uint32Var(&flags.GenerateReplicas, "generate-replicas", uint32(vars.GenerateReplicas), `replicas of the fake node`)
	cmd.Flags().StringVar(&flags.GenerateNodeName, "generate-node-name", vars.GenerateNodeName, `node name of the fake node`)
	cmd.Flags().StringArrayVar(&flags.NodeName, "node-name", vars.NodeName, `node name of the fake node`)
	cmd.Flags().StringVar(&flags.Runtime, "runtime", vars.Runtime, "runtime of the fake cluster ("+strings.Join(runtime.List(), " or ")+")")
	cmd.Flags().StringVar(&flags.FeatureGates, "feature-gates", vars.FeatureGates, "a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes")
	cmd.Flags().StringVar(&flags.RuntimeConfig, "runtime-config", vars.RuntimeConfig, "a set of key=value pairs that enable or disable built-in APIs")
	return cmd
}

func runE(ctx context.Context, logger log.Logger, flags *flagpole) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := utils.PathJoin(vars.TempDir, flags.Name)

	newRuntime, ok := runtime.Get(flags.Runtime)
	if !ok {
		return fmt.Errorf("runtime %q not found", flags.Runtime)
	}

	dc, err := newRuntime(name, workdir, logger)
	if err != nil {
		return err
	}
	_, err = dc.Config()
	if err == nil {
		logger.Printf("Cluster %q already exists", name)
		return nil
	}

	logger.Printf("Creating cluster %q", name)
	err = dc.Install(ctx, runtime.Config{
		Name:                        name,
		ApiserverPort:               flags.ApiserverPort,
		Workdir:                     workdir,
		Runtime:                     flags.Runtime,
		PrometheusImage:             flags.PrometheusImage,
		EtcdImage:                   flags.EtcdImage,
		KubeApiserverImage:          flags.KubeApiserverImage,
		KubeControllerManagerImage:  flags.KubeControllerManagerImage,
		KubeSchedulerImage:          flags.KubeSchedulerImage,
		FakeKubeletImage:            flags.FakeKubeletImage,
		KindNodeImage:               flags.KindNodeImage,
		KubeApiserverBinary:         flags.KubeApiserverBinary,
		KubeControllerManagerBinary: flags.KubeControllerManagerBinary,
		KubeSchedulerBinary:         flags.KubeSchedulerBinary,
		FakeKubeletBinary:           flags.FakeKubeletBinary,
		EtcdBinaryTar:               flags.EtcdBinaryTar,
		PrometheusBinaryTar:         flags.PrometheusBinaryTar,
		CacheDir:                    vars.CacheDir,
		SecretPort:                  flags.SecurePort,
		QuietPull:                   flags.QuietPull,
		PrometheusPort:              flags.PrometheusPort,
		GenerateNodeName:            flags.GenerateNodeName,
		GenerateReplicas:            flags.GenerateReplicas,
		NodeName:                    strings.Join(flags.NodeName, ","),
		FeatureGates:                flags.FeatureGates,
		RuntimeConfig:               flags.RuntimeConfig,
	})
	if err != nil {
		return fmt.Errorf("failed install cluster %q: %w", name, err)
	}

	logger.Printf("Starting cluster %q", name)
	err = dc.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed start cluster %q: %w", name, err)
	}

	logger.Printf("Wait for cluster %q to be ready", name)
	err = dc.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed wait for cluster %q be ready: %w", name, err)
	}

	logger.Printf("Cluster %q is ready", name)

	fmt.Fprintf(os.Stderr, "> kubectl --context %s get node\n", name)
	err = dc.Kubectl(ctx, utils.IOStreams{
		Out:    os.Stderr,
		ErrOut: os.Stderr,
	}, "--context", name, "get", "node")
	if err != nil {
		return err
	}
	return nil
}
