package k8s

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed kubeconfig.yaml.tpl
var kubeconfigYamlTpl string

var kubeconfigYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(kubeconfigYamlTpl))

func BuildKubeconfig(conf BuildKubeconfigConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kubeconfigYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build kubeconfig error: %s", err)
	}
	return buf.String(), nil
}

type BuildKubeconfigConfig struct {
	ProjectName  string
	SecretPort   bool
	Address      string
	AdminCrtPath string
	AdminKeyPath string
}
