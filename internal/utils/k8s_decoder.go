package utils

import (
	"encoding/json"
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	securityv1 "github.com/openshift/api/security/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type DecodedResource struct {
	Object runtime.Object
	GVK    schema.GroupVersionKind
	Raw    map[string]any
}

func init() {
	_ = securityv1.AddToScheme(scheme.Scheme)
	_ = routev1.Install(scheme.Scheme)
	_ = gatewayv1.Install(scheme.Scheme)
	_ = gatewayv1alpha2.Install(scheme.Scheme)
	_ = gatewayv1beta1.Install(scheme.Scheme)
}

func DecodeJSON(raw []byte) (DecodedResource, error) {
	decoder := scheme.Codecs.UniversalDeserializer()
	obj, gvk, err := decoder.Decode(raw, nil, nil)
	if err == nil && obj != nil && gvk != nil && gvk.Kind != "" {
		return DecodedResource{Object: obj, GVK: *gvk}, nil
	}

	var resource map[string]any
	if err := json.Unmarshal(raw, &resource); err != nil {
		return DecodedResource{}, fmt.Errorf("parse JSON: %w", err)
	}

	if resource == nil {
		return DecodedResource{}, nil
	}

	return DecodedResource{Raw: resource}, nil
}

func DecodeJSONToMap(raw []byte) (map[string]any, error) {
	decoded, err := DecodeJSON(raw)
	if err != nil {
		return nil, err
	}
	if decoded.Raw != nil {
		return decoded.Raw, nil
	}
	if decoded.Object != nil {
		return ToMap(decoded.Object)
	}
	return nil, nil
}

func ToMap(obj runtime.Object) (map[string]any, error) {
	if obj == nil {
		return nil, nil
	}
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("convert to map: %w", err)
	}
	return data, nil
}
