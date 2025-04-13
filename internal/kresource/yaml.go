package kresource

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseSingleYamlManifest(content string) (unstructured.Unstructured, error) {
	var u unstructured.Unstructured
	d := yaml.NewDecoder(strings.NewReader(content))
	err := d.Decode(&u.Object)
	if err != nil {
		return u, fmt.Errorf("error parsing yaml manifest: %w", err)
	}
	return u, nil
}

func FormatYaml(u unstructured.Unstructured) (string, error) {
	var b bytes.Buffer
	e := yaml.NewEncoder(&b)
	err := e.Encode(u.Object)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
