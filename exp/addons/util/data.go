package util

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
)

func IsCompressed(src *unstructured.Unstructured) bool {
	annotations := src.GetAnnotations()
	av, ok := annotations[addonsv1.ClusterResourceSetCompressedAnnotation]
	if !ok {
		return false
	}
	return av == "true"
}

func Compress(src *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	var field string
	switch kind := src.GetKind(); kind {
	case string(addonsv1.ConfigMapClusterResourceSetResourceKind):
		field = "binaryData"
	case string(addonsv1.SecretClusterResourceSetResourceKind):
		field = "data"
	default:
		return nil, fmt.Errorf("resource kind is %q, must be ConfigMap or Secret", kind)
	}

	dst := src.DeepCopy()
	if IsCompressed(src) {
		return dst, nil
	}

	dataCopy, ok, err := unstructured.NestedStringMap(src.UnstructuredContent(), field)
	if err != nil {
		return nil, fmt.Errorf("reading data from field %q: %w", field, err)
	}
	if !ok {
		return nil, fmt.Errorf("field %q not found", field)
	}

	for key := range dataCopy {
		cv, err := compressValue(dataCopy[key])
		if err != nil {
			return nil, fmt.Errorf("compressing key %q value: %w", key, err)
		}
		dataCopy[key] = cv
	}

	if err := unstructured.SetNestedStringMap(dst.UnstructuredContent(), dataCopy, field); err != nil {
		return nil, fmt.Errorf("writing compressed data: %w", err)
	}

	annotations := dst.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[addonsv1.ClusterResourceSetCompressedAnnotation] = "true"
	dst.SetAnnotations(annotations)

	return dst, nil
}

func Decompress(src *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	var field string
	switch kind := src.GetKind(); kind {
	case string(addonsv1.ConfigMapClusterResourceSetResourceKind):
		field = "binaryData"
	case string(addonsv1.SecretClusterResourceSetResourceKind):
		field = "data"
	default:
		return nil, fmt.Errorf("resource kind is %q, must be ConfigMap or Secret", kind)
	}

	dst := src.DeepCopy()
	if !IsCompressed(src) {
		return dst, nil
	}

	dataCopy, ok, err := unstructured.NestedMap(src.UnstructuredContent(), field)
	if err != nil {
		return nil, fmt.Errorf("reading data from field %q: %w", field, err)
	}
	if !ok {
		return nil, fmt.Errorf("field %q not found", field)
	}

	for key := range dataCopy {
		cv, ok := dataCopy[key].([]byte)
		if !ok {
			return nil, fmt.Errorf("reading key %q value: %w", key, err)
		}
		v, err := decompressValue(cv)
		if err != nil {
			return nil, fmt.Errorf("decompressing key %q value: %w", key, err)
		}
		dataCopy[key] = v
	}

	if err := unstructured.SetNestedMap(dst.UnstructuredContent(), dataCopy, field); err != nil {
		return nil, fmt.Errorf("writing decompressed data: %w", err)
	}

	annotations := dst.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[addonsv1.ClusterResourceSetCompressedAnnotation] = "false"
	dst.SetAnnotations(annotations)

	return dst, nil
}

// compressValue takes a value that is base64-encoded. It base64-decodes this value, zips it, base64-encodes it again,
// and finally returns it.
func compressValue(v string) (string, error) {
	src := strings.NewReader(v)
	b64r := base64.NewDecoder(base64.StdEncoding, src)

	var dst bytes.Buffer
	b64w := base64.NewEncoder(base64.StdEncoding, &dst)
	zw := gzip.NewWriter(b64w)

	if _, err := io.Copy(zw, b64r); err != nil {
		return "", err
	}
	// Flush any partially written blocks.
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := b64w.Close(); err != nil {
		return "", err
	}

	return dst.String(), nil
}

// decompressValue takes a value that is zipped and base64-encoded (in that order). It base64-decodes this value, unzips
// it, base64-encodes it again, and finally returns it.
func decompressValue(v []byte) ([]byte, error) {
	src := bytes.NewReader(v)
	b64r := base64.NewDecoder(base64.StdEncoding, src)
	zr, err := gzip.NewReader(b64r)
	if err != nil {
		return nil, err
	}

	var dst bytes.Buffer
	zw := base64.NewEncoder(base64.StdEncoding, &dst)

	if _, err := io.Copy(zw, zr); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if err := zr.Close(); err != nil {
		return nil, err
	}

	return dst.Bytes(), nil
}
