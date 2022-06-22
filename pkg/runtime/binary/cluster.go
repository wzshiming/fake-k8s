package binary

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nxadm/tail"
	"github.com/wzshiming/fake-k8s/pkg/k8s"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"github.com/wzshiming/fake-k8s/pkg/pki"
	"github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	"github.com/wzshiming/fake-k8s/pkg/vars"
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

	bin := utils.PathJoin(conf.Workdir, "bin")

	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	fakeKubeletPath := utils.PathJoin(bin, "fake-kubelet"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.FakeKubeletBinary, fakeKubeletPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	etcdPath := utils.PathJoin(bin, "etcd"+vars.BinSuffix)
	err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd"+vars.BinSuffix, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := utils.PathJoin(bin, "prometheus"+vars.BinSuffix)
		err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus"+vars.BinSuffix, 0755, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	etcdDataPath := utils.PathJoin(conf.Workdir, runtime.EtcdDataDirName)
	os.MkdirAll(etcdDataPath, 0755)

	if conf.SecretPort {
		pkiPath := utils.PathJoin(conf.Workdir, runtime.PkiName)
		err = pki.DumpPki(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %s", err)
		}
	}

	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	scheme := "http"
	if conf.SecretPort {
		scheme = "https"
	}
	bin := utils.PathJoin(conf.Workdir, "bin")

	localAddress := "127.0.0.1"
	serveAddress := "0.0.0.0"

	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver")
	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager")
	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler")
	fakeKubeletPath := utils.PathJoin(bin, "fake-kubelet")
	etcdPath := utils.PathJoin(bin, "etcd")
	etcdDataPath := utils.PathJoin(conf.Workdir, runtime.EtcdDataDirName)
	pkiPath := utils.PathJoin(conf.Workdir, runtime.PkiName)
	caCertPath := utils.PathJoin(pkiPath, "ca.crt")
	adminKeyPath := utils.PathJoin(pkiPath, "admin.key")
	adminCertPath := utils.PathJoin(pkiPath, "admin.crt")

	etcdPeerPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	etcdPeerPortStr := strconv.Itoa(etcdPeerPort)

	etcdClientPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	etcdClientPortStr := strconv.Itoa(etcdClientPort)

	etcdArgs := []string{
		"--data-dir",
		etcdDataPath,
		"--name",
		"node0",
		"--initial-advertise-peer-urls",
		"http://" + localAddress + ":" + etcdPeerPortStr,
		"--listen-peer-urls",
		"http://" + localAddress + ":" + etcdPeerPortStr,
		"--advertise-client-urls",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--listen-client-urls",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--initial-cluster",
		"node0=http://" + localAddress + ":" + etcdPeerPortStr,
		"--auto-compaction-retention",
		"1",
		"--quota-backend-bytes",
		"8589934592",
	}
	err = utils.ForkExec(ctx, conf.Workdir, etcdPath, etcdArgs...)
	if err != nil {
		return err
	}

	apiserverPort := int(conf.ApiserverPort)
	if apiserverPort == 0 {
		apiserverPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
	}
	apiserverPortStr := strconv.Itoa(apiserverPort)

	kubeApiserverArgs := []string{
		"--admission-control",
		"",
		"--etcd-servers",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--etcd-prefix",
		"/prefix/registry",
		"--allow-privileged",
	}
	if conf.RuntimeConfig != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--runtime-config",
			conf.RuntimeConfig,
		)
	}
	if conf.FeatureGates != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	if conf.SecretPort {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--bind-address",
			serveAddress,
			"--secure-port",
			apiserverPortStr,
			"--tls-cert-file",
			adminCertPath,
			"--tls-private-key-file",
			adminKeyPath,
			"--client-ca-file",
			caCertPath,
			"--service-account-key-file",
			adminKeyPath,
			"--service-account-signing-key-file",
			adminKeyPath,
			"--service-account-issuer",
			"https://kubernetes.default.svc.cluster.local",
		)
	} else {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--insecure-bind-address",
			serveAddress,
			"--insecure-port",
			apiserverPortStr,
			"--cert-dir",
			utils.PathJoin(conf.Workdir, "cert"),
		)
	}
	err = utils.ForkExec(ctx, conf.Workdir, kubeApiserverPath, kubeApiserverArgs...)
	if err != nil {
		return err
	}

	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  conf.Name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://" + localAddress + ":" + apiserverPortStr,
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}

	kubeconfigPath := utils.PathJoin(conf.Workdir, runtime.InHostKubeconfigName)
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = c.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for kube-apiserver ready: %v", err)
	}

	kubeControllerManagerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	if conf.FeatureGates != "" {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	kubeControllerManagerPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	if conf.SecretPort {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--bind-address",
			localAddress,
			"--secure-port",
			strconv.Itoa(kubeControllerManagerPort),
			"--authorization-always-allow-paths",
			"/healthz,/metrics",
		)
	} else {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--address",
			localAddress,
			"--port",
			strconv.Itoa(kubeControllerManagerPort),
			"--secure-port",
			"0",
		)
	}

	err = utils.ForkExec(ctx, conf.Workdir, kubeControllerManagerPath, kubeControllerManagerArgs...)
	if err != nil {
		return err
	}

	kubeSchedulerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	if conf.FeatureGates != "" {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	kubeSchedulerPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	if conf.SecretPort {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--bind-address",
			localAddress,
			"--secure-port",
			strconv.Itoa(kubeSchedulerPort),
			"--authorization-always-allow-paths",
			"/healthz,/metrics",
		)
	} else {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--address",
			localAddress,
			"--port",
			strconv.Itoa(kubeSchedulerPort),
			"--secure-port",
			"0",
		)
	}
	err = utils.ForkExec(ctx, conf.Workdir, kubeSchedulerPath, kubeSchedulerArgs...)
	if err != nil {
		return err
	}

	nodeTplPath := utils.PathJoin(conf.Workdir, "node.tpl")
	err = os.WriteFile(nodeTplPath, nodeTpl, 0644)
	if err != nil {
		return err
	}

	fakeKubeletArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
		"--take-over-all",
		"--node-name",
		conf.NodeName,
		"--generate-node-name",
		conf.GenerateNodeName,
		"--generate-replicas",
		strconv.Itoa(int(conf.GenerateReplicas)),
		"--node-template-file",
		nodeTplPath,
	}
	var fakeKubeletPort int
	if conf.PrometheusPort != 0 {
		fakeKubeletPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		fakeKubeletArgs = append(fakeKubeletArgs,
			"--server-address",
			localAddress+":"+strconv.Itoa(fakeKubeletPort),
		)
	}
	err = utils.ForkExec(ctx, conf.Workdir, fakeKubeletPath, fakeKubeletArgs...)
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		prometheusPortStr := strconv.Itoa(int(conf.PrometheusPort))

		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:               conf.Name,
			SecretPort:                conf.SecretPort,
			AdminCrtPath:              adminCertPath,
			AdminKeyPath:              adminKeyPath,
			PrometheusPort:            int(conf.PrometheusPort),
			EtcdPort:                  etcdClientPort,
			KubeApiserverPort:         apiserverPort,
			KubeControllerManagerPort: kubeControllerManagerPort,
			KubeSchedulerPort:         kubeSchedulerPort,
			FakeKubeletPort:           fakeKubeletPort,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %s", err)
		}
		prometheusConfigPath := utils.PathJoin(conf.Workdir, runtime.Prometheus)
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %s", err)
		}

		prometheusPath := utils.PathJoin(bin, "prometheus")
		prometheusArgs := []string{
			"--config.file",
			prometheusConfigPath,
			"--web.listen-address",
			serveAddress + ":" + prometheusPortStr,
		}
		err = utils.ForkExec(ctx, conf.Workdir, prometheusPath, prometheusArgs...)
		if err != nil {
			return err
		}
	}

	// set the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".server", scheme+"://"+localAddress+":"+apiserverPortStr)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".cluster", conf.Name)
	if conf.SecretPort {
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".insecure-skip-tls-verify", "true")
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".user", conf.Name)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-certificate", adminCertPath)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-key", adminKeyPath)
	}
	return nil
}

//go:embed node.tpl
var nodeTpl []byte

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "clusters."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "users."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+conf.Name)

	bin := utils.PathJoin(conf.Workdir, "bin")
	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver")
	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager")
	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler")
	fakeKubeletPath := utils.PathJoin(bin, "fake-kubelet")
	etcdPath := utils.PathJoin(bin, "etcd")
	prometheusPath := utils.PathJoin(bin, "prometheus")

	err = utils.ForkExecKill(ctx, conf.Workdir, fakeKubeletPath)
	if err != nil {
		c.Logger().Printf("failed to kill fake-kubelet: %s", err)
	}

	err = utils.ForkExecKill(ctx, conf.Workdir, kubeSchedulerPath)
	if err != nil {
		c.Logger().Printf("failed to kill kube-scheduler: %s", err)
	}

	err = utils.ForkExecKill(ctx, conf.Workdir, kubeControllerManagerPath)
	if err != nil {
		c.Logger().Printf("failed to kill kube-controller-manager: %s", err)
	}

	err = utils.ForkExecKill(ctx, conf.Workdir, kubeApiserverPath)
	if err != nil {
		c.Logger().Printf("failed to kill kube-apiserver: %s", err)
	}

	err = utils.ForkExecKill(ctx, conf.Workdir, etcdPath)
	if err != nil {
		c.Logger().Printf("failed to kill etcd: %s", err)
	}

	if conf.PrometheusPort != 0 {
		err = utils.ForkExecKill(ctx, conf.Workdir, prometheusPath)
		if err != nil {
			c.Logger().Printf("failed to kill prometheus: %s", err)
		}
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")
	svc := utils.PathJoin(bin, name)

	err = utils.ForkExecRestart(ctx, conf.Workdir, svc)
	if err != nil {
		return fmt.Errorf("failed to restart %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")
	svc := utils.PathJoin(bin, name)

	err = utils.ForkExecKill(ctx, conf.Workdir, svc)
	if err != nil {
		return fmt.Errorf("failed to kill %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", filepath.Base(name)+".log")

	f, err := os.OpenFile(logs, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	io.Copy(out, f)
	return nil
}

func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", filepath.Base(name)+".log")

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	defer t.Stop()

	go func() {
		for line := range t.Lines {
			out.Write([]byte(line.Text + "\n"))
		}
	}()
	<-ctx.Done()
	return nil
}
