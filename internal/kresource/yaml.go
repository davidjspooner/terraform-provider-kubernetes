package kresource

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseYamlManifestList(content string) ([]unstructured.Unstructured, error) {
	var ul []unstructured.Unstructured

	d := yaml.NewDecoder(strings.NewReader(content))
	for {
		u := unstructured.Unstructured{}

		//d.KnownFields(true)
		err := d.Decode(&u)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		ul = append(ul, u)
	}
	if len(ul) == 0 {
		return nil, errors.New("no resources found")
	}
	return ul, nil
}

func ParseSingleYamlManifest(content string) (unstructured.Unstructured, error) {
	ul, err := ParseYamlManifestList(content)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	if len(ul) != 1 {
		return unstructured.Unstructured{}, fmt.Errorf("expected one resource, found %d", len(ul))
	}
	return ul[0], nil
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
