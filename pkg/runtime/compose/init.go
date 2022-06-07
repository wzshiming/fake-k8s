package compose

import (
	"github.com/wzshiming/fake-k8s/pkg/runtime"
)

func init() {
	runtime.Register("docker", NewCluster)
	runtime.Register("nerdctl", NewCluster)
}
