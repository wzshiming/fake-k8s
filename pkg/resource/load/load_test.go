package load

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_load(t *testing.T) {
	controller := true
	type args struct {
		input []runtime.Object
	}
	tests := []struct {
		name        string
		args        args
		want        []runtime.Object
		wantErr     bool
		wantUpdated []runtime.Object
	}{
		{
			args: args{
				input: []runtime.Object{
					&appsv1.DaemonSet{
						ObjectMeta: metav1.ObjectMeta{
							UID: "1",
						},
					},
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							UID: "2",
							OwnerReferences: []metav1.OwnerReference{
								{
									Controller: &controller,
									UID:        "1",
								},
							},
						},
					},
				},
			},
			wantUpdated: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						UID: "10",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						UID: "20",
						OwnerReferences: []metav1.OwnerReference{
							{
								Controller: &controller,
								UID:        "10",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := []runtime.Object{}
			apply := func(objs []runtime.Object) ([]runtime.Object, error) {
				ret := []runtime.Object{}
				for _, obj := range objs {
					o := obj.DeepCopyObject()
					meta := o.(metav1.ObjectMetaAccessor).GetObjectMeta()
					meta.SetUID(meta.GetUID() + "0")
					ret = append(ret, o)
					updated = append(updated, o.DeepCopyObject())
				}
				return ret, nil
			}
			got, err := load(tt.args.input, apply)
			if (err != nil) != tt.wantErr {
				t.Errorf("load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("load() got = %v, want %v", got, tt.want)
			}

			if !equality.Semantic.DeepEqual(updated, tt.wantUpdated) {
				t.Errorf("load() got updated = \n%v, want updated \n%v", updated, tt.wantUpdated)
			}
		})
	}
}
