package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/pki"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

func (c *Cluster) Install(ctx context.Context, conf runtime.Config) error {
	err := c.Cluster.Install(ctx, conf)
	if err != nil {
		return err
	}
	err = installCluster(ctx, conf.Name, conf.Workdir, conf)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Uninstall(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = uninstallCluster(ctx, conf.Name, conf.Workdir)
	if err != nil {
		return err
	}
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
		return err
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

// installCluster installs a fake cluster.
func installCluster(ctx context.Context, name, workdir string, conf runtime.Config) error {
	kubeconfigPath := filepath.Join(workdir, runtime.InHostKubeconfigName)
	prometheusPath := ""
	inClusterOnHostKubeconfigPath := filepath.Join(workdir, runtime.InClusterKubeconfigName)
	pkiPath := filepath.Join(workdir, runtime.PkiName)
	composePath := filepath.Join(workdir, runtime.ComposeName)
	err := os.MkdirAll(pkiPath, 0755)
	if err != nil {
		return err
	}

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
		caCertPath = filepath.Join(pkiPath, "ca.crt")
		adminKeyPath = filepath.Join(pkiPath, "admin.key")
		adminCertPath = filepath.Join(pkiPath, "admin.crt")
		inClusterPkiPath := "/etc/kubernetes/pki/"
		inClusterCACertPath = filepath.Join(inClusterPkiPath, "ca.crt")
		inClusterAdminKeyPath = filepath.Join(inClusterPkiPath, "admin.key")
		inClusterAdminCertPath = filepath.Join(inClusterPkiPath, "admin.crt")
		inClusterPort = 6443
		scheme = "https"
	}

	// Setup prometheus
	if conf.PrometheusPort != 0 {
		prometheusPath = filepath.Join(workdir, runtime.Prometheus)
		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:  name,
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
		ProjectName:                name,
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
	})
	if err != nil {
		return err
	}

	// Setup kubeconfig
	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://127.0.0.1:" + strconv.Itoa(port),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}
	inClusterKubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://" + name + "-kube-apiserver:" + strconv.Itoa(inClusterPort),
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
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "clusters."+name+".server", scheme+"://127.0.0.1:"+strconv.Itoa(port))
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+name+".cluster", name)
	if conf.SecretPort {
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "clusters."+name+".insecure-skip-tls-verify", "true")
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+name+".user", name)
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "users."+name+".client-certificate", adminCertPath)
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "users."+name+".client-key", adminKeyPath)
	}
	return nil
}

// uninstallCluster uninstall a fake cluster.
func uninstallCluster(ctx context.Context, name, workdir string) error {
	// unset the context in default kubeconfig
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "clusters."+name)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "users."+name)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "contexts."+name)

	return nil
}
