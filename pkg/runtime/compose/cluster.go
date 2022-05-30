package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/k8s/kubectl"
	"github.com/wzshiming/fake-k8s/pkg/pki"
	"github.com/wzshiming/fake-k8s/pkg/prometheus"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"sigs.k8s.io/yaml"
)

type ClusterConfig struct {
	PrometheusImage            string
	EtcdImage                  string
	KubeApiserverImage         string
	KubeControllerManagerImage string
	KubeSchedulerImage         string
	FakeKubeletImage           string
	SecretPort                 bool
	QuietPull                  bool
	PrometheusPort             uint32
	GenerateNodeName           string
	GenerateReplicas           uint32
	NodeName                   string
}

type Cluster struct {
	name    string
	workdir string
	runtime string
}

func NewCluster(name, workdir, runtime string) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
		runtime: runtime,
	}
}

func (d *Cluster) Config() (*RawClusterConfig, error) {
	config, err := os.ReadFile(filepath.Join(d.workdir, rawClusterConfigName))
	if err != nil {
		return nil, err
	}
	c := RawClusterConfig{}
	err = yaml.Unmarshal(config, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *Cluster) InHostKubeconfig() (string, error) {
	c, err := d.Config()
	if err != nil {
		return "", err
	}

	return filepath.Join(c.Workdir, InHostKubeconfigName), nil
}

func (d *Cluster) Install(ctx context.Context, conf ClusterConfig) error {
	c := RawClusterConfig{
		Workdir:      d.workdir,
		Name:         d.name,
		Runtime:      d.runtime,
		UpCommand:    []string{"compose", "up", "-d"},
		DownCommand:  []string{"compose", "down"},
		StartCommand: []string{"start"},
		StopCommand:  []string{"stop"},
		Cluster:      conf,
	}
	if conf.QuietPull {
		c.UpCommand = append(c.UpCommand, "--quiet-pull")
	}
	config, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = os.MkdirAll(d.workdir, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(d.workdir, rawClusterConfigName), config, 0644)
	if err != nil {
		return err
	}
	err = installCluster(ctx, d.name, d.workdir, conf)
	if err != nil {
		return err
	}
	return nil
}

func (d *Cluster) Uninstall(ctx context.Context) error {
	c, err := d.Config()
	if err != nil {
		return err
	}
	err = uninstallCluster(ctx, c.Name, d.workdir)
	if err != nil {
		return err
	}
	return nil
}

func (d *Cluster) Up(ctx context.Context) error {
	c, err := d.Config()
	if err != nil {
		return err
	}
	output, err := utils.Exec(ctx, d.workdir, c.Runtime, c.UpCommand...)
	if err != nil {
		return fmt.Errorf("%s\n%s", err, output)
	}
	return nil
}

func (d *Cluster) Down(ctx context.Context) error {
	c, err := d.Config()
	if err != nil {
		return err
	}
	output, err := utils.Exec(ctx, d.workdir, c.Runtime, c.DownCommand...)
	if err != nil {
		return fmt.Errorf("%s\n%s", err, output)
	}
	return nil
}

func (d *Cluster) Start(ctx context.Context, name string) error {
	c, err := d.Config()
	if err != nil {
		return err
	}
	output, err := utils.Exec(ctx, d.workdir, c.Runtime, append(c.StartCommand, name)...)
	if err != nil {
		return fmt.Errorf("%s\n%s", err, output)
	}
	return nil
}

func (d *Cluster) Stop(ctx context.Context, name string) error {
	c, err := d.Config()
	if err != nil {
		return err
	}
	output, err := utils.Exec(ctx, d.workdir, c.Runtime, append(c.StopCommand, name)...)
	if err != nil {
		return fmt.Errorf("%s\n%s", err, output)
	}
	return nil
}

type RawClusterConfig struct {
	Name         string
	Workdir      string
	Runtime      string
	UpCommand    []string
	DownCommand  []string
	StartCommand []string
	StopCommand  []string
	Cluster      ClusterConfig
}

var (
	rawClusterConfigName    = "fake-k8s.yaml"
	InHostKubeconfigName    = "kubeconfig.yaml"
	InClusterKubeconfigName = "kubeconfig"
	EtcdDataDirName         = "etcd"
	PkiName                 = "pki"
	ComposeName             = "docker-compose.yaml"
	Prometheus              = "prometheus.yaml"
)

// installCluster installs a fake cluster.
func installCluster(ctx context.Context, name, workdir string, conf ClusterConfig) error {
	kubeconfigPath := filepath.Join(workdir, InHostKubeconfigName)
	prometheusPath := ""
	inClusterOnHostKubeconfigPath := filepath.Join(workdir, InClusterKubeconfigName)
	etcdPath := filepath.Join(workdir, EtcdDataDirName)
	pkiPath := filepath.Join(workdir, PkiName)
	composePath := filepath.Join(workdir, ComposeName)
	err := os.MkdirAll(etcdPath, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(pkiPath, 0755)
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
		prometheusPath = filepath.Join(workdir, Prometheus)
		prometheusData, err := prometheus.BuildPrometheus(prometheus.BuildPrometheusConfig{
			ProjectName:  name,
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
		EtcdDataPath:               etcdPath,
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
		Address:      scheme + "://127.0.0.1:" + strconv.Itoa(port),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}
	inClusterKubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  name,
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
	kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "clusters."+name+".server", scheme+"://127.0.0.1:"+strconv.Itoa(port))
	kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "contexts."+name+".cluster", name)
	if adminKeyPath != "" {
		kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "clusters."+name+".insecure-skip-tls-verify", "true")
		kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "contexts."+name+".user", name)
		kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "users."+name+".client-certificate", adminCertPath)
		kubectl.Run(ctx, utils.IOStreams{}, "config", "set", "users."+name+".client-key", adminKeyPath)
	}
	return nil
}

// uninstallCluster uninstall a fake cluster.
func uninstallCluster(ctx context.Context, name, workdir string) error {
	// unset the context in default kubeconfig
	kubectl.Run(ctx, utils.IOStreams{}, "config", "unset", "clusters."+name)
	kubectl.Run(ctx, utils.IOStreams{}, "config", "unset", "users."+name)
	kubectl.Run(ctx, utils.IOStreams{}, "config", "unset", "contexts."+name)

	// cleanup workdir
	os.RemoveAll(workdir)
	return nil
}
