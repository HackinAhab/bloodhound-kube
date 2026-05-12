package workload

import (
	"encoding/base64"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
)

func secretDataToAnyMap(secret *corev1.Secret, decodeOpaqueValues bool) map[string]any {
	if secret == nil {
		return map[string]any{}
	}
	data := make(map[string]any)
	for key, value := range secret.Data {
		if decodeOpaqueValues {
			data[key] = secretValueForDisplay(value)
			continue
		}
		data[key] = base64.StdEncoding.EncodeToString(value)
	}
	for key, value := range secret.StringData {
		if _, exists := data[key]; !exists {
			data[key] = value
		}
	}
	return data
}

func secretValueForDisplay(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	if utf8.Valid(value) {
		return string(value)
	}
	return base64.StdEncoding.EncodeToString(value)
}
