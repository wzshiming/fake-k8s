package binary

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

	bin := filepath.Join(conf.Workdir, "bin")

	kubeApiserverPath := filepath.Join(bin, "kube-apiserver")
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0755)
	if err != nil {
		return err
	}

	kubeControllerManagerPath := filepath.Join(bin, "kube-controller-manager")
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0755)
	if err != nil {
		return err
	}

	kubeSchedulerPath := filepath.Join(bin, "kube-scheduler")
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0755)
	if err != nil {
		return err
	}

	fakeKubeletPath := filepath.Join(bin, "fake-kubelet")
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.FakeKubeletBinary, fakeKubeletPath, 0755)
	if err != nil {
		return err
	}

	etcdPath := filepath.Join(bin, "etcd")
	err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd", 0755)
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := filepath.Join(bin, "prometheus")
		err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus", 0755)
		if err != nil {
			return err
		}
	}

	pids := filepath.Join(conf.Workdir, "pids")
	os.MkdirAll(pids, 0755)

	logs := filepath.Join(conf.Workdir, "logs")
	os.MkdirAll(logs, 0755)

	cmdlines := filepath.Join(conf.Workdir, "cmdlines")
	os.MkdirAll(cmdlines, 0755)

	etcdDataPath := filepath.Join(conf.Workdir, runtime.EtcdDataDirName)
	os.MkdirAll(etcdDataPath, 0755)

	if conf.SecretPort {
		pkiPath := filepath.Join(conf.Workdir, runtime.PkiName)
		os.MkdirAll(pkiPath, 0755)
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
	bin := filepath.Join(conf.Workdir, "bin")

	kubeApiserverPath := filepath.Join(bin, "kube-apiserver")
	kubeControllerManagerPath := filepath.Join(bin, "kube-controller-manager")
	kubeSchedulerPath := filepath.Join(bin, "kube-scheduler")
	fakeKubeletPath := filepath.Join(bin, "fake-kubelet")
	etcdPath := filepath.Join(bin, "etcd")
	etcdDataPath := filepath.Join(conf.Workdir, runtime.EtcdDataDirName)
	pkiPath := filepath.Join(conf.Workdir, runtime.PkiName)
	caCertPath := filepath.Join(pkiPath, "ca.crt")
	adminKeyPath := filepath.Join(pkiPath, "admin.key")
	adminCertPath := filepath.Join(pkiPath, "admin.crt")

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
		"http://0.0.0.0:" + etcdPeerPortStr,
		"--listen-peer-urls",
		"http://0.0.0.0:" + etcdPeerPortStr,
		"--advertise-client-urls",
		"http://0.0.0.0:" + etcdClientPortStr,
		"--listen-client-urls",
		"http://0.0.0.0:" + etcdClientPortStr,
		"--initial-cluster",
		"node0=http://0.0.0.0:" + etcdPeerPortStr,
		"--auto-compaction-retention",
		"1",
		"--quota-backend-bytes",
		"8589934592",
	}
	err = utils.ForkExec(conf.Workdir, etcdPath, etcdArgs...)
	if err != nil {
		return err
	}

	apiserverPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	apiserverPortStr := strconv.Itoa(apiserverPort)

	kubeApiserverArgs := []string{
		"--admission-control",
		"",
		"--etcd-servers",
		"http://127.0.0.1:" + etcdClientPortStr,
		"--etcd-prefix",
		"/prefix/registry",
		"--default-watch-cache-size",
		"10000",
		"--allow-privileged",
	}
	if conf.SecretPort {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--bind-address",
			"0.0.0.0",
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
			"0.0.0.0",
			"--insecure-port",
			apiserverPortStr,
			"--cert-dir",
			filepath.Join(conf.Workdir, "cert"),
		)
	}
	err = utils.ForkExec(conf.Workdir, kubeApiserverPath, kubeApiserverArgs...)
	if err != nil {
		return err
	}

	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  conf.Name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://127.0.0.1:" + apiserverPortStr,
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}

	kubeconfigPath := filepath.Join(conf.Workdir, runtime.InHostKubeconfigName)
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	for i := 0; ; i++ {
		ready, err := c.Ready(ctx)
		if ready {
			break
		}
		time.Sleep(time.Second)
		if i > 10 {
			return err
		}
	}

	kubeControllerManagerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}

	kubeControllerManagerPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	if conf.SecretPort {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--bind-address",
			"0.0.0.0",
			"--secure-port",
			strconv.Itoa(kubeControllerManagerPort),
			"--authorization-always-allow-paths",
			"/healthz,/metrics",
		)
	} else {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--address",
			"0.0.0.0",
			"--port",
			strconv.Itoa(kubeControllerManagerPort),
			"--secure-port",
			"0",
		)
	}

	err = utils.ForkExec(conf.Workdir, kubeControllerManagerPath, kubeControllerManagerArgs...)
	if err != nil {
		return err
	}

	kubeSchedulerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	kubeSchedulerPort, err := utils.GetUnusedPort()
	if err != nil {
		return err
	}
	if conf.SecretPort {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--bind-address",
			"0.0.0.0",
			"--secure-port",
			strconv.Itoa(kubeSchedulerPort),
			"--authorization-always-allow-paths",
			"/healthz,/metrics",
		)
	} else {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--address",
			"0.0.0.0",
			"--port",
			strconv.Itoa(kubeSchedulerPort),
			"--secure-port",
			"0",
		)
	}
	err = utils.ForkExec(conf.Workdir, kubeSchedulerPath, kubeSchedulerArgs...)
	if err != nil {
		return err
	}

	nodeTplPath := filepath.Join(conf.Workdir, "node.tpl")
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
			"0.0.0.0:"+strconv.Itoa(fakeKubeletPort),
		)
	}
	err = utils.ForkExec(conf.Workdir, fakeKubeletPath, fakeKubeletArgs...)
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
		prometheusConfigPath := filepath.Join(conf.Workdir, runtime.Prometheus)
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %s", err)
		}

		prometheusPath := filepath.Join(bin, "prometheus")
		prometheusArgs := []string{
			"--config.file",
			prometheusConfigPath,
			"--web.listen-address",
			"0.0.0.0:" + prometheusPortStr,
		}
		err = utils.ForkExec(conf.Workdir, prometheusPath, prometheusArgs...)
		if err != nil {
			return err
		}
	}

	// set the context in default kubeconfig
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "clusters."+conf.Name+".server", scheme+"://127.0.0.1:"+apiserverPortStr)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+conf.Name+".cluster", conf.Name)
	if conf.SecretPort {
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "clusters."+conf.Name+".insecure-skip-tls-verify", "true")
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "contexts."+conf.Name+".user", conf.Name)
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "users."+conf.Name+".client-certificate", adminCertPath)
		utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "set", "users."+conf.Name+".client-key", adminKeyPath)
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

	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "clusters."+conf.Name)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "users."+conf.Name)
	utils.Exec(ctx, "", utils.IOStreams{}, "kubectl", "config", "unset", "contexts."+conf.Name)

	bin := filepath.Join(conf.Workdir, "bin")
	kubeApiserverPath := filepath.Join(bin, "kube-apiserver")
	kubeControllerManagerPath := filepath.Join(bin, "kube-controller-manager")
	kubeSchedulerPath := filepath.Join(bin, "kube-scheduler")
	fakeKubeletPath := filepath.Join(bin, "fake-kubelet")
	etcdPath := filepath.Join(bin, "etcd")
	prometheusPath := filepath.Join(bin, "prometheus")

	err = utils.ForkExecKill(conf.Workdir, fakeKubeletPath)
	if err != nil {
		return fmt.Errorf("failed to kill fake-kubelet: %w", err)
	}

	err = utils.ForkExecKill(conf.Workdir, kubeSchedulerPath)
	if err != nil {
		return fmt.Errorf("failed to kill kube-scheduler: %w", err)
	}

	err = utils.ForkExecKill(conf.Workdir, kubeControllerManagerPath)
	if err != nil {
		return fmt.Errorf("failed to kill kube-controller-manager: %w", err)
	}

	err = utils.ForkExecKill(conf.Workdir, kubeApiserverPath)
	if err != nil {
		return fmt.Errorf("failed to kill kube-apiserver: %w", err)
	}

	err = utils.ForkExecKill(conf.Workdir, etcdPath)
	if err != nil {
		return fmt.Errorf("failed to kill etcd: %w", err)
	}

	if conf.PrometheusPort != 0 {
		err = utils.ForkExecKill(conf.Workdir, prometheusPath)
		if err != nil {
			return fmt.Errorf("failed to kill prometheus: %w", err)
		}
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := filepath.Join(conf.Workdir, "bin")
	svc := filepath.Join(bin, name)

	err = utils.ForkExecRestart(conf.Workdir, svc)
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

	bin := filepath.Join(conf.Workdir, "bin")
	svc := filepath.Join(bin, name)

	err = utils.ForkExecKill(conf.Workdir, svc)
	if err != nil {
		return fmt.Errorf("failed to kill %s: %w", name, err)
	}
	return nil
}
