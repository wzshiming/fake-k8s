package compose

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/pki"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string, logger log.Logger) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir, logger),
	}, nil
}

func (c *Cluster) Install(ctx context.Context, conf runtime.Config) error {
	err := c.Cluster.Install(ctx, conf)
	if err != nil {
		return err
	}

	kubeconfigPath := utils.PathJoin(conf.Workdir, runtime.InHostKubeconfigName)
	prometheusPath := ""
	inClusterOnHostKubeconfigPath := utils.PathJoin(conf.Workdir, runtime.InClusterKubeconfigName)
	pkiPath := utils.PathJoin(conf.Workdir, runtime.PkiName)
	composePath := utils.PathJoin(conf.Workdir, runtime.ComposeName)

	caCertPath := ""
	adminKeyPath := ""
	adminCertPath := ""
	inClusterKubeconfigPath := "/root/.kube/config"
	inClusterEtcdDataPath := "/etcd-data"
	InClusterPrometheusPath := "/etc/prometheus/prometheus.yml"
	inClusterAdminKeyPath := ""
	inClusterAdminCertPath := ""
	inClusterCACertPath := ""
	inClusterPort := 8080
	scheme := "http"

	// generate ca cert
	if conf.SecretPort {
		err := pki.DumpPki(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %s", err)
		}
		caCertPath = utils.PathJoin(pkiPath, "ca.crt")
		adminKeyPath = utils.PathJoin(pkiPath, "admin.key")
		adminCertPath = utils.PathJoin(pkiPath, "admin.crt")
		inClusterPkiPath := "/etc/kubernetes/pki/"
		inClusterCACertPath = utils.PathJoin(inClusterPkiPath, "ca.crt")
		inClusterAdminKeyPath = utils.PathJoin(inClusterPkiPath, "admin.key")
		inClusterAdminCertPath = utils.PathJoin(inClusterPkiPath, "admin.crt")
		inClusterPort = 6443
		scheme = "https"
	}

	// Setup prometheus
	if conf.PrometheusPort != 0 {
		prometheusPath = utils.PathJoin(conf.Workdir, runtime.Prometheus)
		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:  conf.Name,
			SecretPort:   conf.SecretPort,
			AdminCrtPath: inClusterAdminCertPath,
			AdminKeyPath: inClusterAdminKeyPath,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %s", err)
		}
		err = os.WriteFile(prometheusPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %s", err)
		}
	}

	port, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}

	// Setup compose
	dockercompose, err := BuildCompose(BuildComposeConfig{
		ProjectName:                conf.Name,
		ApiserverPort:              uint32(port),
		KubeconfigPath:             inClusterOnHostKubeconfigPath,
		AdminCertPath:              adminCertPath,
		AdminKeyPath:               adminKeyPath,
		CACertPath:                 caCertPath,
		InClusterKubeconfigPath:    inClusterKubeconfigPath,
		InClusterAdminCertPath:     inClusterAdminCertPath,
		InClusterAdminKeyPath:      inClusterAdminKeyPath,
		InClusterCACertPath:        inClusterCACertPath,
		InClusterEtcdDataPath:      inClusterEtcdDataPath,
		InClusterPrometheusPath:    InClusterPrometheusPath,
		PrometheusPath:             prometheusPath,
		EtcdImage:                  conf.EtcdImage,
		KubeApiserverImage:         conf.KubeApiserverImage,
		KubeControllerManagerImage: conf.KubeControllerManagerImage,
		KubeSchedulerImage:         conf.KubeSchedulerImage,
		FakeKubeletImage:           conf.FakeKubeletImage,
		PrometheusImage:            conf.PrometheusImage,
		SecretPort:                 conf.SecretPort,
		QuietPull:                  conf.QuietPull,
		PrometheusPort:             conf.PrometheusPort,
		GenerateNodeName:           conf.GenerateNodeName,
		GenerateReplicas:           conf.GenerateReplicas,
		NodeName:                   conf.NodeName,
		RuntimeConfig:              conf.RuntimeConfig,
		FeatureGates:               conf.FeatureGates,
	})
	if err != nil {
		return err
	}

	// Setup kubeconfig
	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  conf.Name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://127.0.0.1:" + strconv.Itoa(port),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}
	inClusterKubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  conf.Name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://" + conf.Name + "-kube-apiserver:" + strconv.Itoa(inClusterPort),
		AdminCrtPath: inClusterAdminCertPath,
		AdminKeyPath: inClusterAdminKeyPath,
	})
	if err != nil {
		return err
	}

	// Save config
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(inClusterOnHostKubeconfigPath, []byte(inClusterKubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(composePath, []byte(dockercompose), 0644)
	if err != nil {
		return err
	}

	// set the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".server", scheme+"://127.0.0.1:"+strconv.Itoa(port))
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".cluster", conf.Name)
	if conf.SecretPort {
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".insecure-skip-tls-verify", "true")
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".user", conf.Name)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-certificate", adminCertPath)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-key", adminKeyPath)
	}
	return nil
}

func (c *Cluster) Uninstall(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	// unset the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "clusters."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "users."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+conf.Name)

	err = c.Cluster.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	args := []string{"compose", "up", "-d"}
	if conf.QuietPull {
		args = append(args, "--quiet-pull")
	}
	err = utils.Exec(ctx, conf.Workdir, utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, args...)
	if err != nil {
		return err
	}

	for i := 0; ; i++ {
		ready, err := c.Ready(ctx)
		if ready {
			break
		}
		time.Sleep(time.Second)
		if i > 30 {
			return err
		}
	}
	return nil
}

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	args := []string{"compose", "down"}
	err = utils.Exec(ctx, conf.Workdir, utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, args...)
	if err != nil {
		c.Logger().Printf("Failed to down cluster: %v", err)
	}
	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, conf.Workdir, utils.IOStreams{}, conf.Runtime, "start", conf.Name+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, conf.Workdir, utils.IOStreams{}, conf.Runtime, "stop", conf.Name+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, conf.Name+"-"+name)
	err = utils.Exec(ctx, conf.Workdir, utils.IOStreams{
		ErrOut: out,
		Out:    out,
	}, conf.Runtime, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, false)
}

func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, true)
}
