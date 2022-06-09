package binary

import (
	"github.com/wzshiming/fake-k8s/pkg/runtime"
)

func init() {
	runtime.Register("binary", NewCluster)
}
