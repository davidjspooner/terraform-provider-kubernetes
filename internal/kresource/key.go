package kresource

import (
	"strings"
)

type MetaData struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type Key struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
}

func (key *Key) String() string {
	return key.ApiVersion + "/" + key.Kind + "/" + key.MetaData.Namespace + "/" + key.MetaData.Name
}

func CompareKeys(a, b *Key) int {
	r := strings.Compare(a.ApiVersion, b.ApiVersion)
	if r != 0 {
		return r
	}
	r = strings.Compare(a.Kind, b.Kind)
	if r != 0 {
		return r
	}
	r = strings.Compare(a.MetaData.Namespace, b.MetaData.Namespace)
	if r != 0 {
		return r
	}

	return 0
}
