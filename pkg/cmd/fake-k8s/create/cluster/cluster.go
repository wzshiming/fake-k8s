package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/cmd"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name                        string
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
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger cmd.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a fake Kubernetes cluster",
		Long:  "Creates a fake Kubernetes cluster using container",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", "default", `cluster name`)
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
	return cmd
}

func runE(ctx context.Context, logger cmd.Logger, flags *flagpole) error {
	name := vars.ProjectName + "-" + flags.Name
	workdir := filepath.Join(vars.TempDir, flags.Name)

	newRuntime, ok := runtime.Get(flags.Runtime)
	if !ok {
		return fmt.Errorf("runtime %q not found", flags.Runtime)
	}

	dc, err := newRuntime(name, workdir)
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
	})
	if err != nil {
		return fmt.Errorf("failed install %q cluster: %w", name, err)
	}

	logger.Printf("Starting cluster %q", name)
	err = dc.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed start %q cluster: %w", name, err)
	}

	logger.Printf("Wait for cluster %q to be ready", name)
	for i := 0; ; i++ {
		ready, err := dc.Ready(ctx)
		if ready {
			break
		}
		time.Sleep(time.Second)
		if i > 30 {
			return err
		}
	}

	logger.Printf("Cluster %q is ready", name)

	fmt.Fprintf(os.Stderr, "> kubectl --context %s get node\n", name)
	err = utils.Exec(ctx, "", utils.IOStreams{
		Out:    os.Stderr,
		ErrOut: os.Stderr,
	}, "kubectl", "--context", name, "get", "node")
	if err != nil {
		return err
	}
	return nil
}
