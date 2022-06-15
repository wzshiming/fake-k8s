package runtime

import (
	"context"
	"os"
	"os/exec"

	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
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
	conf    *Config
	logger  log.Logger
}

func NewCluster(name, workdir string, logger log.Logger) *Cluster {
	return &Cluster{
		name:    name,
		workdir: workdir,
		logger:  logger,
	}
}

func (c *Cluster) Logger() log.Logger {
	return c.logger
}

func (c *Cluster) Config() (*Config, error) {
	if c.conf != nil {
		return c.conf, nil
	}
	config, err := os.ReadFile(utils.PathJoin(c.workdir, RawClusterConfigName))
	if err != nil {
		return nil, err
	}
	conf := Config{}
	err = yaml.Unmarshal(config, &conf)
	if err != nil {
		return nil, err
	}
	c.conf = &conf
	return c.conf, nil
}

func (c *Cluster) InHostKubeconfig() (string, error) {
	conf, err := c.Config()
	if err != nil {
		return "", err
	}

	return utils.PathJoin(conf.Workdir, InHostKubeconfigName), nil
}

func (c *Cluster) Load(ctx context.Context) (conf Config, err error) {
	file, err := os.ReadFile(utils.PathJoin(c.workdir, RawClusterConfigName))
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

	bin := utils.PathJoin(conf.Workdir, "bin")

	_, err = exec.LookPath("kubectl")
	if err != nil {
		kubectlPath := utils.PathJoin(bin, "kubectl"+vars.BinSuffix)
		err = utils.DownloadWithCache(ctx, conf.CacheDir, vars.KubectlBinary, kubectlPath, 0755, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(c.workdir, RawClusterConfigName), config, 0644)
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

	err = c.Kubectl(ctx, utils.IOStreams{}, "--kubeconfig", utils.PathJoin(conf.Workdir, InHostKubeconfigName), "get", "node")
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cluster) Kubectl(ctx context.Context, stm utils.IOStreams, args ...string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		bin := utils.PathJoin(conf.Workdir, "bin")
		kubectlPath = utils.PathJoin(bin, "kubectl"+vars.BinSuffix)
	}
	return utils.Exec(ctx, "", stm, kubectlPath, args...)
}
