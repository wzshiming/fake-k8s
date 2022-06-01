package cluster

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/wzshiming/fake-k8s/pkg/runtime/compose"
	"github.com/wzshiming/fake-k8s/pkg/vars"
)

type flagpole struct {
	Name                       string
	PrometheusPort             uint32
	SecurePort                 bool
	QuietPull                  bool
	EtcdImage                  string
	KubeApiserverImage         string
	KubeControllerManagerImage string
	KubeSchedulerImage         string
	FakeKubeletImage           string
	PrometheusImage            string
	GenerateReplicas           uint32
	GenerateNodeName           string
	NodeName                   []string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logr.Logger) *cobra.Command {
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
	cmd.HasFlags()
	cmd.Flags().StringVar(&flags.Name, "name", "default", "cluster name, config")
	cmd.Flags().Uint32Var(&flags.PrometheusPort, "prometheus-port", uint32(vars.PrometheusPort), "port to expose Prometheus metrics")
	cmd.Flags().BoolVar(&flags.SecurePort, "secure-port", vars.SecurePort, "apiserver use TLS")
	cmd.Flags().BoolVar(&flags.QuietPull, "quiet-pull", vars.QuietPull, "pull without printing progress information")
	cmd.Flags().StringVar(&flags.EtcdImage, "etcd-image", vars.EtcdImage, "image of etcd \n'${KUBE_IMAGE_PREFIX}/etcd:${ETCD_VERSION}'")
	cmd.Flags().StringVar(&flags.KubeApiserverImage, "kube-apiserver-image", vars.KubeApiserverImage, "image of kube-apiserver \n'${KUBE_IMAGE_PREFIX}/kube-apiserver:${KUBE_VERSION}'\n")
	cmd.Flags().StringVar(&flags.KubeControllerManagerImage, "kube-controller-manager-image", vars.KubeControllerManagerImage, "image of kube-controller-manager \n'${KUBE_IMAGE_PREFIX}/kube-controller-manager:${KUBE_VERSION}'\n")
	cmd.Flags().StringVar(&flags.KubeSchedulerImage, "kube-scheduler-image", vars.KubeSchedulerImage, "image of kube-scheduler \n'${KUBE_IMAGE_PREFIX}/kube-scheduler:${KUBE_VERSION}'\n")
	cmd.Flags().StringVar(&flags.FakeKubeletImage, "fake-kubelet-image", vars.FakeKubeletImage, "image of fake-kubelet \n'${FAKE_IMAGE_PREFIX}/fake-kubelet:${FAKE_VERSION}'\n")
	cmd.Flags().StringVar(&flags.PrometheusImage, "prometheus-image", vars.PrometheusImage, "image of Prometheus \n'${PROMETHEUS_IMAGE_PREFIX}/prometheus:${PROMETHEUS_VERSION}'\n")
	cmd.Flags().Uint32Var(&flags.GenerateReplicas, "generate-replicas", uint32(vars.GenerateReplicas), "replicas of the fake node")
	cmd.Flags().StringVar(&flags.GenerateNodeName, "generate-node-name", vars.GenerateNodeName, "node name of the fake node")
	cmd.Flags().StringArrayVar(&flags.NodeName, "node-name", vars.NodeName, "node name of the fake node")
	return cmd
}

func runE(ctx context.Context, logger logr.Logger, flags *flagpole) error {
	dc := compose.NewCluster(vars.ProjectName+"-"+flags.Name, filepath.Join(vars.TempDir, flags.Name), vars.Runtime)
	_, err := dc.Config()
	if err == nil {
		return fmt.Errorf("cluster %q is existing", flags.Name)
	}
	err = dc.Install(ctx, compose.ClusterConfig{
		PrometheusImage:            flags.PrometheusImage,
		EtcdImage:                  flags.EtcdImage,
		KubeApiserverImage:         flags.KubeApiserverImage,
		KubeControllerManagerImage: flags.KubeControllerManagerImage,
		KubeSchedulerImage:         flags.KubeSchedulerImage,
		FakeKubeletImage:           flags.FakeKubeletImage,
		SecretPort:                 flags.SecurePort,
		QuietPull:                  flags.QuietPull,
		PrometheusPort:             flags.PrometheusPort,
		GenerateNodeName:           flags.GenerateNodeName,
		GenerateReplicas:           flags.GenerateReplicas,
		NodeName:                   strings.Join(flags.NodeName, ","),
	})
	if err != nil {
		return err
	}

	err = dc.Up(ctx)
	if err != nil {
		return err
	}

	return nil
}
