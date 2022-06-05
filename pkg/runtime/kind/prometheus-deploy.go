package kind

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed prometheus-deploy.yaml.tpl
var prometheusDeployYamlTpl string

var prometheusDeployYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(prometheusDeployYamlTpl))

func BuildPrometheusDeploy(conf BuildPrometheusDeployConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := prometheusDeployYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute prometheus deploy yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildPrometheusDeployConfig struct {
	PrometheusImage string
	Name            string
}
