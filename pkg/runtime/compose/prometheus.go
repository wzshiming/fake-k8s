package compose

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed prometheus.yaml.tpl
var prometheusYamlTpl string

var prometheusYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(prometheusYamlTpl))

func BuildPrometheus(conf BuildPrometheusConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := prometheusYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build prometheus error: %s", err)
	}
	return buf.String(), nil
}

type BuildPrometheusConfig struct {
	ProjectName  string
	SecretPort   bool
	AdminCrtPath string
	AdminKeyPath string
}
