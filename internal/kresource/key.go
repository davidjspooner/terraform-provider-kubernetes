package kresource

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type MetaData struct {
	Name        string            `yaml:"name" tfsdk:"name"`
	Namespace   *string           `yaml:"namespace,omitempty" tfsdk:"namespace"`
	Labels      map[string]string `yaml:"labels,omitempty" tfsdk:"labels"`
	Annotations map[string]string `yaml:"annotations,omitempty" tfsdk:"annotations"`
}

func (m *MetaData) FromManifest(manifest *unstructured.Unstructured) {
	m.Name = manifest.GetName()
	s, _, _ := unstructured.NestedString(manifest.Object, "metadata", "namespace")
	if s != "" {
		m.Namespace = &s
	} else {
		m.Namespace = nil
	}
	m.Labels = manifest.GetLabels()
	m.Annotations = manifest.GetAnnotations()
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
func GetKey(r unstructured.Unstructured) *Key {
	if r.Object == nil {
		return nil
	}
	k := Key{}
	k.ApiVersion = r.GetAPIVersion()
	k.Kind = r.GetKind()
	k.MetaData.Name = r.GetName()
	namespace := r.GetNamespace()
	k.MetaData.Namespace = &namespace
	return &k
}
