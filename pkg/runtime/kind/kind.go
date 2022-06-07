package kind

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed kind.yaml.tpl
var kindYamlTpl string

var kindYamlTemplate = template.Must(template.New("_").Delims("${{", "}}").Parse(kindYamlTpl))

func BuildKind(conf BuildKindConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kindYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kind yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildKindConfig struct {
	PrometheusPort uint32
}
