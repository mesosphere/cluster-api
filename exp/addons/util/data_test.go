package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name        string
		obj         *unstructured.Unstructured
		wantDataKey string
		wantData    map[string]string
		wantErr     bool
	}{
		{
			name: "compress ConfigMap",
			obj: fakeObj(t, "ConfigMap", map[string]string{
				"key": "value",
			}),
			wantDataKey: "binaryData",
			wantData: map[string]string{
				"key": "H4sIAAAAAAAA/ypLzClNBQQAAP//NFh3HQUAAAA=",
			},
		},
		{
			name: "compress Secret",
			obj: fakeObj(t, "Secret", map[string]string{
				"key": "value",
			}),
			wantDataKey: "data",
			wantData: map[string]string{
				"key": "H4sIAAAAAAAA/ypLzClNBQQAAP//NFh3HQUAAAA=",
			},
		},
		{
			name: "fail to compress unknown kind",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Foobar",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compress(tt.obj)
			assert.Equal(t, tt.wantErr, err != nil, "error %s", err)

			if tt.wantErr {
				return
			}

			// Check the data, if necessary
			gotData, ok, err := unstructured.NestedStringMap(got.UnstructuredContent(), tt.wantDataKey)
			assert.True(t, ok)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantData, gotData)

			// Verify that Compress is idempotent
			got2, err := Compress(got)
			assert.NoError(t, err)
			assert.Equal(t, got, got2)
		})
	}
}

func fakeObj(t *testing.T, kind string, data map[string]string) *unstructured.Unstructured {
	t.Helper()

	binaryData := make(map[string][]byte, len(data))
	for k := range data {
		binaryData[k] = []byte(data[k])
	}

	var tmp runtime.Object
	switch kind {
	case string(addonsv1.ConfigMapClusterResourceSetResourceKind):
		tmp = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example",
				Namespace: "default",
			},
			BinaryData: binaryData,
		}
	case string(addonsv1.SecretClusterResourceSetResourceKind):
		tmp = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example",
				Namespace: "default",
			},
			Data: binaryData,
		}
	default:
		t.Fatalf("unknown kind %s", kind)
	}
	c := fake.NewClientBuilder().Build()
	obj := &unstructured.Unstructured{}
	if err := c.Scheme().Convert(tmp, obj, nil); err != nil {
		t.Fatalf("converting input to unstructured: %s", err)
	}
	return obj
}
