package kresource

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourceKey struct {
	ApiVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	MetaData   ResourceMetaData `yaml:"metadata"`
}

func (key *ResourceKey) String() string {

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
func GetKey(r unstructured.Unstructured) *ResourceKey {
	if r.Object == nil {
		return nil
	}
	k := ResourceKey{}
	k.ApiVersion = r.GetAPIVersion()
	k.Kind = r.GetKind()
	k.MetaData.Name = r.GetName()
	namespace := r.GetNamespace()
	k.MetaData.Namespace = &namespace
	return &k
}
