package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"strings"

	fakeruntime "github.com/wzshiming/fake-k8s/pkg/runtime"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	oyaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func Load(ctx context.Context, rt fakeruntime.Runtime, src string) error {
	file, err := openFile(src)
	if err != nil {
		return err
	}
	defer file.Close()

	objs, err := decodeObjects(file)
	if err != nil {
		return err
	}
	inputRaw := bytes.NewBuffer(nil)
	outputRaw := bytes.NewBuffer(nil)
	otherResource, err := load(objs, func(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
		inputRaw.Reset()
		outputRaw.Reset()

		encoder := json.NewEncoder(inputRaw)
		for _, obj := range objs {
			err = encoder.Encode(obj)
			if err != nil {
				return nil, err
			}
		}

		err = rt.KubectlInCluster(ctx, utils.IOStreams{
			In:     inputRaw,
			Out:    outputRaw,
			ErrOut: os.Stderr,
		}, "create", "--validate=false", "-o", "json", "-f", "-")
		if err != nil {
			for _, obj := range objs {
				fmt.Fprintf(os.Stderr, "%s/%s failed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
			}
		}
		newObj, err := decodeObjects(outputRaw)
		if err != nil {
			return nil, err
		}
		for _, obj := range newObj {
			fmt.Fprintf(os.Stderr, "%s/%s succeed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
		}
		return newObj, nil
	})
	if err != nil {
		return err
	}
	for _, obj := range otherResource {
		fmt.Fprintf(os.Stderr, "%s/%s skipped\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
	}
	return nil
}

func openFile(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	return os.Open(path)
}

func decodeObjects(data io.Reader) ([]*unstructured.Unstructured, error) {
	out := []*unstructured.Unstructured{}
	tmp := map[string]interface{}{}
	decoder := oyaml.NewDecoder(data)
	for {
		err := decoder.Decode(&tmp)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		data, err := oyaml.Marshal(tmp)
		if err != nil {
			return nil, err
		}
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			return nil, err
		}
		obj := &unstructured.Unstructured{}
		err = obj.UnmarshalJSON(data)
		if err != nil {
			return nil, err
		}

		if obj.IsList() {
			err = obj.EachListItem(func(object runtime.Object) error {
				out = append(out, object.DeepCopyObject().(*unstructured.Unstructured))
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			out = append(out, obj.DeepCopyObject().(*unstructured.Unstructured))
		}
	}
	return out, nil
}

func filter(input []*unstructured.Unstructured, fun func(*unstructured.Unstructured) bool) []*unstructured.Unstructured {
	ret := []*unstructured.Unstructured{}
	for _, i := range input {
		if fun(i) {
			ret = append(ret, i)
		}
	}
	return ret
}

func load(input []*unstructured.Unstructured, apply func([]*unstructured.Unstructured) ([]*unstructured.Unstructured, error)) ([]*unstructured.Unstructured, error) {
	applyResource := []*unstructured.Unstructured{}
	otherResource := []*unstructured.Unstructured{}

	for _, obj := range input {
		// These are built-in resources that do not need to be created
		if obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" &&
			(obj.GetName() == "kube-public" ||
				obj.GetName() == "kube-node-lease" ||
				obj.GetName() == "kube-system" ||
				obj.GetName() == "default") {
			continue
		}

		refs := obj.GetOwnerReferences()
		if len(refs) != 0 && refs[0].Controller != nil && *refs[0].Controller {
			otherResource = append(otherResource, obj)
		} else {
			applyResource = append(applyResource, obj)
		}
	}

	for len(applyResource) != 0 {
		nextApplyResource := []*unstructured.Unstructured{}
		newResource, err := apply(applyResource)
		if err != nil {
			return nil, err
		}
		if len(otherResource) == 0 {
			break
		}
		for i, newObj := range newResource {
			oldUid := applyResource[i].GetUID()
			newUid := newObj.GetUID()

			remove := map[*unstructured.Unstructured]struct{}{}
			nextResource := filter(otherResource, func(otherObj *unstructured.Unstructured) bool {
				otherRefs := otherObj.GetOwnerReferences()
				otherRef := &otherRefs[0]
				if otherRef.UID != oldUid {
					return false
				}
				otherRef.UID = newUid
				otherObj.SetOwnerReferences(otherRefs)
				remove[otherObj] = struct{}{}
				return true
			})
			if len(remove) != 0 {
				otherResource = filter(otherResource, func(otherObj *unstructured.Unstructured) bool {
					_, ok := remove[otherObj]
					return !ok
				})
				nextApplyResource = append(nextApplyResource, nextResource...)
			}
		}
		applyResource = nextApplyResource
	}
	return otherResource, nil
}
