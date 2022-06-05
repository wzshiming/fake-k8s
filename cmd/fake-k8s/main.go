package main

import (
	"fmt"
	"os"

	_ "github.com/wzshiming/fake-k8s/pkg/runtime/compose"
	_ "github.com/wzshiming/fake-k8s/pkg/runtime/kind"

	"github.com/go-logr/zapr"
	fakek8s "github.com/wzshiming/fake-k8s/pkg/cmd/fake-k8s"
	"go.uber.org/zap"
)

func main() {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	logger := zapr.NewLogger(zapLog)

	cmd := fakek8s.NewCommand(logger)
	err = cmd.Execute()
	if err != nil {
		logger.Error(err, "Failed")
		os.Exit(1)
	}
}
