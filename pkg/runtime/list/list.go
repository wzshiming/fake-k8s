package list

import (
	"os"

	"github.com/wzshiming/fake-k8s/pkg/vars"
)

// ListClusters returns the list of clusters in the directory
func ListClusters(workdir string) ([]string, error) {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ret := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			ret = append(ret, entry.Name())
		}
	}
	return ret, nil
}

// ListImages returns the list of images
func ListImages() ([]string, error) {
	return []string{
		vars.EtcdImage,
		vars.KubeApiserverImage,
		vars.KubeControllerManagerImage,
		vars.KubeSchedulerImage,
		vars.FakeKubeletImage,
		vars.PrometheusImage,
	}, nil
}
