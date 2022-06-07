package kind

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed fake-kubelet-deploy.yaml.tpl
var fakeKubeletDeployYamlTpl string

var fakeKubeletDeployYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(fakeKubeletDeployYamlTpl))

func BuildFakeKubeletDeploy(conf BuildFakeKubeletDeployConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := fakeKubeletDeployYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute fake kubelet deploy yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildFakeKubeletDeployConfig struct {
	FakeKubeletImage string
	Name             string
	NodeName         string
	GenerateNodeName string
	GenerateReplicas uint32
}
