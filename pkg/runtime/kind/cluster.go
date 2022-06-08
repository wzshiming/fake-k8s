package kind

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

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

	kindYaml, err := BuildKind(BuildKindConfig{
		PrometheusPort: conf.PrometheusPort,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(conf.Workdir, runtime.KindName), []byte(kindYaml), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	fakeKubeletDeploy, err := BuildFakeKubeletDeploy(BuildFakeKubeletDeployConfig{
		FakeKubeletImage: conf.FakeKubeletImage,
		Name:             conf.Name,
		NodeName:         conf.NodeName,
		GenerateNodeName: conf.GenerateNodeName,
		GenerateReplicas: conf.GenerateReplicas,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(conf.Workdir, runtime.FakeKubeletDeploy), []byte(fakeKubeletDeploy), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.FakeKubeletDeploy, err)
	}

	if conf.PrometheusPort != 0 {
		prometheusDeploy, err := BuildPrometheusDeploy(BuildPrometheusDeployConfig{
			PrometheusImage: conf.PrometheusImage,
			Name:            conf.Name,
		})
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(conf.Workdir, runtime.PrometheusDeploy), []byte(prometheusDeploy), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.PrometheusDeploy, err)
		}
	}
	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	err = utils.Exec(ctx, "", utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, "create", "cluster",
		"--config", filepath.Join(conf.Workdir, runtime.KindName),
		"--name", conf.Name,
		"--image", conf.KindNodeImage,
	)
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "cordon", conf.Name+"-control-plane")
	if err != nil {
		return err
	}

	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "inspect",
		conf.FakeKubeletImage,
	)
	if err != nil {
		err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "pull",
			conf.FakeKubeletImage,
		)
		if err != nil {
			return err
		}
	}

	err = utils.Exec(ctx, "", utils.IOStreams{}, "kind", "load", "docker-image",
		conf.FakeKubeletImage,
		"--name", conf.Name,
	)
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{
		ErrOut: os.Stderr,
	}, "kubectl", "apply", "-f", filepath.Join(conf.Workdir, runtime.FakeKubeletDeploy))
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "inspect",
			conf.PrometheusImage,
		)
		if err != nil {
			err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "pull",
				conf.PrometheusImage,
			)
			if err != nil {
				return err
			}
		}

		err = utils.Exec(ctx, "", utils.IOStreams{}, "kind", "load", "docker-image",
			conf.PrometheusImage,
			"--name", conf.Name,
		)
		if err != nil {
			return err
		}
		err = utils.Exec(ctx, "", utils.IOStreams{
			ErrOut: os.Stderr,
		}, "kubectl", "apply", "-f", filepath.Join(conf.Workdir, runtime.PrometheusDeploy))
		if err != nil {
			return err
		}
	}

	kubeconfig, err := c.InHostKubeconfig()
	if err != nil {
		return err
	}

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = utils.Exec(ctx, "", utils.IOStreams{
		Out: kubeconfigBuf,
	}, "kubectl", "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfig, kubeconfigBuf.Bytes(), 0644)
	if err != nil {
		return err
	}

	// set the context in default kubeconfig
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+conf.Name+".cluster", "kind-"+conf.Name)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+conf.Name+".user", "kind-"+conf.Name)
	return nil
}

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{
		ErrOut: os.Stderr,
	}, conf.Runtime, "delete", "cluster", "--name", conf.Name)
	if err != nil {
		return err
	}

	// unset the context in default kubeconfig
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "contexts."+conf.Name+".cluster")
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "contexts."+conf.Name+".user")
	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "exec", "-it", conf.Name+"-control-plane", "mv", "/etc/kubernetes/"+name+".yaml.bak", "/etc/kubernetes/manifests/"+name+".yaml")
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
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "exec", "-it", conf.Name+"-control-plane", "mv", "/etc/kubernetes/manifests/"+name+".yaml", "/etc/kubernetes/"+name+".yaml.bak")
	if err != nil {
		return err
	}
	return nil
}
