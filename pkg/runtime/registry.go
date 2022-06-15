package runtime

import (
	"context"
	"fmt"
	"github.com/wzshiming/fake-k8s/pkg/log"
	"sort"
)

type NewRuntime func(name, workdir string, logger log.Logger) (Runtime, error)

var registry = map[string]NewRuntime{}

// Register a runtime
func Register(name string, rt NewRuntime) {
	registry[name] = rt
}

// Get a runtime
func Get(name string) (rt NewRuntime, ok bool) {
	rt, ok = registry[name]
	if ok {
		return rt, true
	}

	return nil, false
}

func Load(name, workdir string, logger log.Logger) (Runtime, error) {
	dc := NewCluster(name, workdir, logger)
	conf, err := dc.Load(context.Background())
	if err != nil {
		return nil, err
	}
	nr, ok := Get(conf.Runtime)
	if !ok {
		return nil, fmt.Errorf("not found runtime %q", conf.Runtime)
	}
	return nr(name, workdir, logger)
}

// List all registered runtime
func List() []string {
	var rts []string
	for name := range registry {
		rts = append(rts, name)
	}
	sort.Strings(rts)
	return rts
}
