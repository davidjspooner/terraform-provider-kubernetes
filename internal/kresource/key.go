package kresource

import (
	"fmt"
	"strings"
)

type MetaData struct {
	Name        string            `yaml:"name" tfsdk:"name"`
	Namespace   *string           `yaml:"namespace,omitempty" tfsdk:"namespace"`
	Labels      map[string]string `yaml:"labels,omitempty" tfsdk:"labels"`
	Annotations map[string]string `yaml:"annotations,omitempty" tfsdk:"annotations"`
}

type Key struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
}

func (key *Key) String() string {

	sb := strings.Builder{}
	fmt.Fprintf(&sb, "%s/%s", key.ApiVersion, key.Kind)
	if key.MetaData.Namespace != nil {
		fmt.Fprintf(&sb, "/%s", *key.MetaData.Namespace)
	} else {
		fmt.Fprintf(&sb, "/default")
	}
	fmt.Fprintf(&sb, "/%s", key.MetaData.Name)
	return sb.String()
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
	var namespace1, namespace2 string
	if a.MetaData.Namespace != nil {
		namespace1 = *a.MetaData.Namespace
	}
	if b.MetaData.Namespace != nil {
		namespace2 = *b.MetaData.Namespace
	}

	r = strings.Compare(namespace1, namespace2)
	if r != 0 {
		return r
	}

	return 0
}
