package kresource

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseResourceYaml(content string) (*Resource, error) {
	r := &Resource{}

	d := yaml.NewDecoder(strings.NewReader(content))
	//d.KnownFields(true)
	err := d.Decode(&r.Unstructured.Object)
	if err != nil {
		return nil, err
	}
	r.Key.ApiVersion = r.Unstructured.GetAPIVersion()
	r.Key.Kind = r.Unstructured.GetKind()
	r.Key.MetaData.Name = r.Unstructured.GetName()
	r.Key.MetaData.Namespace = r.Unstructured.GetNamespace()
	return r, nil
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
