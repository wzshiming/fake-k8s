package compose

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed compose.yaml.tpl
var composeYamlTpl string

var composeYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(composeYamlTpl))

func BuildCompose(conf BuildComposeConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute compose yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildComposeConfig struct {
	ProjectName string

	PrometheusImage            string
	EtcdImage                  string
	KubeApiserverImage         string
	KubeControllerManagerImage string
	KubeSchedulerImage         string
	FakeKubeletImage           string

	PrometheusPath          string
	EtcdDataPath            string
	AdminKeyPath            string
	AdminCertPath           string
	CACertPath              string
	KubeconfigPath          string
	InClusterAdminKeyPath   string
	InClusterAdminCertPath  string
	InClusterCACertPath     string
	InClusterKubeconfigPath string
	InClusterEtcdDataPath   string
	InClusterPrometheusPath string

	SecretPort bool
	QuietPull  bool

	ApiserverPort  uint32
	PrometheusPort uint32

	NodeName         string
	GenerateNodeName string
	GenerateReplicas uint32
}
