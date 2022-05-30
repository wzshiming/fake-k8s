package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wzshiming/fake-k8s/pkg/k8s/kubectl"
	"github.com/wzshiming/fake-k8s/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/scheme"
)

func Load(ctx context.Context, kubeconfig, src string) error {
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
	otherResource, err := load(objs, func(objs []runtime.Object) ([]runtime.Object, error) {
		inputRaw.Reset()
		outputRaw.Reset()

		encoder := json.NewEncoder(inputRaw)
		for _, obj := range objs {
			err = encoder.Encode(obj)
			if err != nil {
				return nil, err
			}
		}

		err = kubectl.Run(ctx, utils.IOStreams{
			In:     inputRaw,
			Out:    outputRaw,
			ErrOut: os.Stderr,
		}, "--kubeconfig", kubeconfig, "apply", "--force=true", "--overwrite=true", "--validate=false", "-o", "json", "-f", "-")
		if err != nil {
			for _, obj := range objs {
				fmt.Fprintf(os.Stderr, "%s/%s failed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName())
			}
			return nil, err
		}
		newObj, err := decodeObjects(outputRaw)
		if err != nil {
			return nil, err
		}
		for _, obj := range newObj {
			fmt.Fprintf(os.Stderr, "%s/%s succeed\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName())
		}
		return newObj, nil
	})
	if err != nil {
		return err
	}
	for _, obj := range otherResource {
		fmt.Fprintf(os.Stderr, "%s/%s skipped\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName())
	}
	return nil
}

func openFile(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	return os.Open(path)
}

func decodeObjects(data io.Reader) ([]runtime.Object, error) {
	builder := resource.NewLocalBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		Stream(data, "input").
		Flatten().
		ContinueOnError()

	result := builder.Do()

	if err := result.Err(); err != nil {
		return nil, err
	}
	infos, err := result.Infos()
	if err != nil {
		return nil, err
	}
	objects := make([]runtime.Object, 0, len(infos))
	for _, info := range infos {
		if info.Object != nil {
			objects = append(objects, info.Object)
		}
	}
	return objects, nil
}

func filter(input []runtime.Object, fun func(runtime.Object) bool) []runtime.Object {
	ret := []runtime.Object{}
	for _, i := range input {
		if fun(i) {
			ret = append(ret, i)
		}
	}
	return ret
}

func load(input []runtime.Object, apply func([]runtime.Object) ([]runtime.Object, error)) ([]runtime.Object, error) {
	applyResource := []runtime.Object{}
	otherResource := []runtime.Object{}

	for _, obj := range input {
		if oma, ok := obj.(metav1.ObjectMetaAccessor); ok {
			objMeta := oma.GetObjectMeta()

			// These are built-in resources that do not need to be created
			if obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" &&
				(objMeta.GetName() == "kube-public" ||
					objMeta.GetName() == "kube-node-lease" ||
					objMeta.GetName() == "kube-system" ||
					objMeta.GetName() == "default") {
				continue
			}

			refs := objMeta.GetOwnerReferences()
			if len(refs) != 0 && refs[0].Controller != nil && *refs[0].Controller {
				otherResource = append(otherResource, obj)
			} else {
				oma.GetObjectMeta().SetUID("")
				applyResource = append(applyResource, obj)
			}
		}
	}

	for len(applyResource) != 0 {
		nextApplyResource := []runtime.Object{}
		newResource, err := apply(applyResource)
		if err != nil {
			return nil, err
		}
		for _, newObj := range newResource {
			newObjMeta := newObj.(metav1.ObjectMetaAccessor).GetObjectMeta()
			newUid := newObjMeta.GetUID()
			newObjKind := newObj.GetObjectKind()
			newGVK := newObjKind.GroupVersionKind()
			newName := newObjMeta.GetName()
			newNamespace := newObjMeta.GetNamespace()

			remove := map[runtime.Object]struct{}{}
			nextResource := filter(otherResource, func(otherObj runtime.Object) bool {
				otherObjMeta := otherObj.(metav1.ObjectMetaAccessor).GetObjectMeta()
				otherRef := otherObjMeta.GetOwnerReferences()[0]
				otherGV, _ := schema.ParseGroupVersion(otherRef.APIVersion)
				otherGVK := schema.GroupVersionKind{
					Group:   otherGV.Group,
					Version: otherGV.Version,
					Kind:    otherRef.Kind,
				}
				if newGVK != otherGVK {
					return false
				}
				if newNamespace != "" && newNamespace != otherObjMeta.GetNamespace() {
					return false
				}
				if otherRef.Name != newName {
					return false
				}
				otherRef.UID = newUid
				otherObjMeta.SetUID("")
				remove[otherObj] = struct{}{}
				return true
			})
			if len(remove) != 0 {
				otherResource = filter(otherResource, func(otherObj runtime.Object) bool {
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
