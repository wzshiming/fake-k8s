package main

import (
	"log"
	"os"

	_ "github.com/wzshiming/fake-k8s/pkg/runtime/binary"
	_ "github.com/wzshiming/fake-k8s/pkg/runtime/compose"
	_ "github.com/wzshiming/fake-k8s/pkg/runtime/kind"

	fakek8s "github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s"
)

func main() {
	logger := log.New(os.Stdout, "", 0)
	cmd := fakek8s.NewCommand(logger)
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
