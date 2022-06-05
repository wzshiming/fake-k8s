package kind

import (
	"github.com/wzshiming/fake-k8s/pkg/runtime"
)

func init() {
	runtime.Register("kind", NewCluster)
}
