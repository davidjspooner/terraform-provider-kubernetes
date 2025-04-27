package kube

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourceKey struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string  `yaml:"name"`
		Namespace *string `yaml:"namespace,omitempty"`
	} `yaml:"metadata"`
}

func (key *ResourceKey) String() string {

	sb := strings.Builder{}
	fmt.Fprintf(&sb, "%s/%s", key.ApiVersion, key.Kind)
	if key.Metadata.Namespace != nil {
		fmt.Fprintf(&sb, "/%s", *key.Metadata.Namespace)
	} else {
		fmt.Fprintf(&sb, "/default")
	}
	fmt.Fprintf(&sb, "/%s", key.Metadata.Name)
	return sb.String()
}

func CompareKeys(a, b *ResourceKey) int {
	r := strings.Compare(a.ApiVersion, b.ApiVersion)
	if r != 0 {
		return r
	}
	r = strings.Compare(a.Kind, b.Kind)
	if r != 0 {
		return r
	}
	var namespace1, namespace2 string
	if a.Metadata.Namespace != nil {
		namespace1 = *a.Metadata.Namespace
	}
	if b.Metadata.Namespace != nil {
		namespace2 = *b.Metadata.Namespace
	}

	r = strings.Compare(namespace1, namespace2)
	if r != 0 {
		return r
	}

	return 0
}
func GetKey(r unstructured.Unstructured) *ResourceKey {
	if r.Object == nil {
		return nil
	}
	k := ResourceKey{}
	k.ApiVersion = r.GetAPIVersion()
	k.Kind = r.GetKind()
	k.Metadata.Name = r.GetName()
	namespace := r.GetNamespace()
	k.Metadata.Namespace = &namespace
	return &k
}
