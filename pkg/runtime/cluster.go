package runtime

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wzshiming/fake-k8s/pkg/utils"
	"sigs.k8s.io/yaml"
)

var (
	RawClusterConfigName    = "fake-k8s.yaml"
	InHostKubeconfigName    = "kubeconfig.yaml"
	InClusterKubeconfigName = "kubeconfig"
	EtcdDataDirName         = "etcd"
	PkiName                 = "pki"
	ComposeName             = "docker-compose.yaml"
	Prometheus              = "prometheus.yaml"
	KindName                = "kind.yaml"
	FakeKubeletDeploy       = "fake-kubelet-deploy.yaml"
	PrometheusDeploy        = "prometheus-deploy.yaml"
)

type Cluster struct {
	workdir string
	name    string
}

func NewCluster(name, workdir string) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
	}
}

func (c *Cluster) Config() (*Config, error) {
	config, err := os.ReadFile(filepath.Join(c.workdir, RawClusterConfigName))
	if err != nil {
		return nil, err
	}
	r := Config{}
	err = yaml.Unmarshal(config, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Cluster) InHostKubeconfig() (string, error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}

	return filepath.Join(conf.Workdir, InHostKubeconfigName), nil
}

func (c *Cluster) Load(ctx context.Context) (conf Config, err error) {
	file, err := os.ReadFile(filepath.Join(c.workdir, RawClusterConfigName))
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return Config{}, err
	}
	return conf, nil
}

func (c *Cluster) Install(ctx context.Context, conf Config) error {
	config, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	err = os.MkdirAll(c.workdir, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(c.workdir, RawClusterConfigName), config, 0644)
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

	// cleanup workdir
	os.RemoveAll(conf.Workdir)
	return nil
}

func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	conf, err := c.Config()
	if err != nil {
		return false, err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "--kubeconfig", filepath.Join(conf.Workdir, InHostKubeconfigName), "get", "node")
	if err != nil {
		return false, err
	}
	return true, nil
}
