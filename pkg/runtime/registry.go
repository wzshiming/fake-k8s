package runtime

import (
	"context"
	"fmt"
)

type NewRuntime func(name, workdir string) (Runtime, error)

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

func Load(name, workdir string) (Runtime, error) {
	dc := NewCluster(name, workdir)
	conf, err := dc.Load(context.Background())
	if err != nil {
		return nil, err
	}
	nr, ok := Get(conf.Runtime)
	if !ok {
		return nil, fmt.Errorf("not found runtime %q", conf.Runtime)
	}
	return nr(name, workdir)
}

// List all registered runtime
func List() []string {
	var rts []string
	for name := range registry {
		rts = append(rts, name)
	}
	return rts
}
